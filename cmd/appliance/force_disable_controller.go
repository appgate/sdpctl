package appliance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type cmdOpts struct {
	Config        *configuration.Config
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Out           io.Writer
	SpinnerOut    func() io.Writer
	NoInteractive bool
	CiMode        bool
}

func NewForceDisableControllerCMD(f *factory.Factory) *cobra.Command {
	opts := cmdOpts{
		Appliance:  f.Appliance,
		Config:     f.Config,
		Out:        f.IOOutWriter,
		SpinnerOut: f.GetSpinnerOutput(),
	}
	cmd := &cobra.Command{
		Use:         "force-disable-controller [hostname...]",
		Short:       docs.ApplianceForceDisableControllerDocs.Short,
		Long:        docs.ApplianceForceDisableControllerDocs.Long,
		Example:     docs.ApplianceForceDisableControllerDocs.ExampleString(),
		Annotations: map[string]string{"MinAPIVersion": "18"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var errs *multierror.Error
			var err error

			if opts.NoInteractive, err = cmd.Flags().GetBool("no-interactive"); err != nil {
				errs = multierror.Append(errs, err)
			}

			if !f.CanPrompt() {
				opts.NoInteractive = true
			}

			if len(args) <= 0 && opts.NoInteractive {
				errs = multierror.Append(errs, errors.New("No arguments provided while running in no-interactive mode"))
			}

			if opts.CiMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				errs = multierror.Append(errs, err)
			}

			return errs.ErrorOrNil()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return forceDisableControllerRunE(opts, args)
		},
	}

	return cmd
}

func forceDisableControllerRunE(opts cmdOpts, args []string) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appliances, err := a.List(ctx, nil)
	if err != nil {
		return err
	}
	controllers := appliancepkg.GroupByFunctions(appliances)[appliancepkg.FunctionController]
	log.WithField("controllers", controllers).Debug("controller list fetched")

	if len(args) <= 0 {
		hostnames := []string{}
		for _, ctrl := range controllers {
			hostnames = append(hostnames, ctrl.GetHostname())
		}
		qs := &survey.MultiSelect{
			PageSize: len(hostnames),
			Message:  "Select controllers to force disable",
			Options:  hostnames,
		}
		if err := prompt.SurveyAskOne(qs, &args); err != nil {
			return err
		}
		if len(args) <= 0 {
			return errors.New("No controllers selected to disable")
		}
	}

	announceList := []openapi.Appliance{}
	disableList := []string{}
	for _, ctrl := range controllers {
		if util.InSlice(ctrl.GetHostname(), args) {
			disableList = append(disableList, ctrl.GetId())
		} else {
			announceList = append(announceList, ctrl)
		}
	}

	type changeStruct struct {
		changeID   string
		controller openapi.Appliance
		tracker    *tui.Tracker
	}
	type confirmationStruct struct {
		changeDetails   changeStruct
		deadControllers []string
	}
	var p *tui.Progress
	if !opts.CiMode {
		p = tui.New(ctx, opts.SpinnerOut())
		defer p.Wait()
	}
	confirmationChan := make(chan confirmationStruct)
	errChan := make(chan error, len(announceList))
	var wg1 sync.WaitGroup
	// Used for synronizing API calls when force-disabling controller
	timer := time.NewTimer(2 * time.Second)
	for _, ctrl := range announceList {
		wg1.Add(1)
		var t *tui.Tracker
		if !opts.CiMode {
			t = p.AddTracker(ctrl.GetName(), "complete")
			go func(t *tui.Tracker) {
				t.Watch([]string{"complete"}, []string{"failed"})
			}(t)
		}
		go func(ctx context.Context, wg *sync.WaitGroup, timer *time.Timer, confirmationChan chan confirmationStruct, errChan chan error, ctrl openapi.Appliance) {
			defer wg.Done()
			hostname := ctrl.GetHostname()
			<-timer.C
			if t != nil {
				t.Update("disabling")
			}
			response, changeID, err := a.ForceDisableControllers(ctx, hostname, disableList)
			if err != nil {
				errChan <- err
				if t != nil {
					t.Fail(err.Error())
				}
				return
			}
			deadControllers := response.GetOfflineControllers()
			confirmationChan <- confirmationStruct{
				deadControllers: deadControllers,
				changeDetails: changeStruct{
					changeID:   changeID,
					controller: ctrl,
					tracker:    t,
				},
			}
		}(ctx, &wg1, timer, confirmationChan, errChan, ctrl)
	}

	var errs *multierror.Error
	go func(errs *multierror.Error) {
		for err := range errChan {
			// Abort if any controller fails
			cancel()
			log.WithError(err).Error("force disable controller command error")
			errs = multierror.Append(errs, err)
		}
	}(errs)

	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(confirmationChan)
	}(&wg1)

	deadControllers := []string{}
	changeList := []changeStruct{}
	for c := range confirmationChan {
		if len(c.deadControllers) > 0 {
			for _, v := range c.deadControllers {
				deadControllers = util.AppendIfMissing(deadControllers, v)
			}
		}
		changeList = append(changeList, c.changeDetails)
	}

	if len(deadControllers) > 0 {
		fmt.Fprintln(opts.Out, "Some Controllers seem to be offline and unable to recieve the force-disable-controller request.")
		fmt.Fprintln(opts.Out, "Please confirm that the following Controllers are in fact offline and unreachable before continuing:")
		for _, c := range deadControllers {
			fmt.Fprintln(opts.Out, c)
		}
		if !opts.NoInteractive {
			if err := prompt.AskConfirmation("All listed Controllers are offline"); err != nil {
				return err
			}
		}
	}

	var wg2 sync.WaitGroup
	for _, c := range changeList {
		wg2.Add(1)
		go func(wg *sync.WaitGroup, changeDetails changeStruct) {
			defer wg.Done()
			ac := change.ApplianceChange{
				APIClient: a.APIClient,
				Token:     a.Token,
			}
			ch, err := ac.RetryUntilCompleted(ctx, changeDetails.changeID, changeDetails.controller.GetId())
			if err != nil {
				errChan <- err
				if changeDetails.tracker != nil {
					changeDetails.tracker.Fail(err.Error())
				}
				return
			}
			log.WithFields(log.Fields{
				"controller": changeDetails.controller.GetName(),
				"change-ID":  ch.GetId(),
				"status":     ch.GetStatus(),
				"result":     ch.GetResult(),
				"details":    ch.GetDetails(),
			}).Info("change was successfully applied to Controller. Re-allocating IPs")
			hostname := changeDetails.controller.GetHostname()
			if !opts.CiMode {
				changeDetails.tracker.Update("re-allocating IPs")
			}
			changeID, err := a.RepartitionIPAllocations(ctx, hostname)
			if err != nil {
				errChan <- err
				if changeDetails.tracker != nil {
					changeDetails.tracker.Fail(err.Error())
				}
				return
			}
			ch, err = ac.RetryUntilCompleted(ctx, changeID, changeDetails.controller.GetId())
			if err != nil {
				errChan <- err
				if changeDetails.tracker != nil {
					changeDetails.tracker.Fail(err.Error())
				}
				return
			}
			if changeDetails.tracker != nil {
				changeDetails.tracker.Update("complete")
			}
			log.WithFields(log.Fields{
				"controller": changeDetails.controller.GetName(),
				"change-ID":  ch.GetId(),
				"status":     ch.GetStatus(),
				"result":     ch.GetResult(),
				"details":    ch.GetDetails(),
			}).Info("IPs successfully re-allocated")
		}(&wg2, c)
	}

	wg2.Wait()

	return errs.ErrorOrNil()
}
