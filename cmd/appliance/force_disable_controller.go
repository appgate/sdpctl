package appliance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"text/template"

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

func NewForceDisableControllerCmd(f *factory.Factory) *cobra.Command {
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
	if a.ApplianceStats == nil {
		a.ApplianceStats = &appliancepkg.ApplianceStatus{
			Appliance: a,
		}
	}
	changeAPI := change.ApplianceChange{
		APIClient: a.APIClient,
		Token:     a.Token,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appliances, err := a.List(ctx, nil)
	if err != nil {
		return err
	}

	primaryHost, err := cfg.GetHost()
	if err != nil {
		return err
	}
	primaryController, err := appliancepkg.FindPrimaryController(appliances, primaryHost, true)
	if err != nil {
		return err
	}
	log.WithField("controller", primaryController.GetName()).Info("Found primary Controller")

	if util.InSlice(primaryController.GetHostname(), args) {
		return errors.New("Illegal operation. Disabling the primary Controller is not allowed")
	}

	rawControllers := appliancepkg.GroupByFunctions(appliances)[appliancepkg.FunctionController]
	// Remove primary Controller from list
	controllers := []openapi.Appliance{}
	for _, ctrl := range rawControllers {
		if ctrl.GetId() == primaryController.GetId() {
			continue
		}
		controllers = append(controllers, ctrl)
	}

	if len(controllers) <= 0 {
		return fmt.Errorf("No controllers to disable")
	}

	stats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	statData := stats.GetData()

	// Error is ignored here since the function returns an error when either a controller or logserver is offline, which is fine at this point
	controllers, offline, _ := appliancepkg.FilterAvailable(controllers, statData)

	// Sort slices by appliance name
	sort.SliceStable(controllers, func(i, j int) bool { return controllers[i].GetName() < controllers[j].GetName() })
	sort.SliceStable(offline, func(i, j int) bool { return offline[i].GetName() < offline[j].GetName() })
	sort.SliceStable(statData, func(i, j int) bool { return statData[i].GetName() < statData[j].GetName() })

	unselectedOffline := []openapi.Appliance{}
	if len(args) <= 0 {
		selectable := []string{}
		for _, ctrl := range controllers {
			selectable = append(selectable, fmt.Sprintf("%s (%s)", ctrl.GetName(), ctrl.GetHostname()))
		}
		for _, ctrl := range offline {
			selectable = append(selectable, fmt.Sprintf("%s (%s) [OFFLINE]", ctrl.GetName(), ctrl.GetHostname()))
		}
		qs := &survey.MultiSelect{
			PageSize: len(selectable),
			Message:  "Select Controllers to force disable",
			Options:  selectable,
		}
		selected := []string{}
		if err := prompt.SurveyAskOne(qs, &selected); err != nil {
			return err
		}
		if len(selected) <= 0 {
			return errors.New("No Controllers selected to disable")
		}
		for _, ctrl := range controllers {
			for _, s := range selected {
				if strings.Contains(s, ctrl.GetName()) {
					args = append(args, ctrl.GetHostname())
				}
			}
		}
		for _, ctrl := range offline {
			for _, s := range selected {
				if strings.Contains(s, ctrl.GetName()) {
					args = append(args, ctrl.GetHostname())
				} else {
					unselectedOffline = append(unselectedOffline, ctrl)
				}
			}
		}
	}
	log.WithField("controllers", args).Debug("selected")

	disableList := []openapi.Appliance{}
	announceList := []openapi.Appliance{}
	for _, ctrl := range controllers {
		if util.InSlice(ctrl.GetHostname(), args) {
			disableList = append(disableList, ctrl)
		} else {
			announceList = append(announceList, ctrl)
		}
	}
	for _, ctrl := range offline {
		if util.InSlice(ctrl.GetHostname(), args) {
			disableList = append(disableList, ctrl)
		}
	}

	// Summary
	summary, err := printSummary(statData, primaryController.GetId(), disableList, unselectedOffline)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Out, summary)
	if !opts.NoInteractive {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	var p *tui.Progress
	var t *tui.Tracker
	if !opts.CiMode {
		p = tui.New(ctx, opts.SpinnerOut())
		defer p.Wait()

		msg := "disabling controller"
		if len(disableList) > 1 {
			msg += "s"
		}
		t = p.AddTracker(msg, "done")
		go func(t *tui.Tracker) {
			t.Watch([]string{"done"}, []string{})
		}(t)
		defer t.Update("done")
	}

	if t != nil {
		t.Update("disabling")
	}
	res, changeID, err := a.ForceDisableControllers(ctx, disableList)
	if err != nil {
		if t != nil {
			t.Fail(err.Error())
		}
		return err
	}

	// Re-add primary controller and sort the list
	announceList = append(announceList, *primaryController)
	sort.SliceStable(announceList, func(i, j int) bool {
		return announceList[i].GetName() > announceList[j].GetName()
	})

	var errs *multierror.Error
	if o, ok := res.GetOfflineControllersOk(); ok {
		for _, a := range announceList {
			if util.InSlice(a.GetId(), o) {
				errs = multierror.Append(errs, fmt.Errorf("Failed to send disable command to Controller %s", a.GetName()))
			}
		}
		if errs != nil {
			return errs.ErrorOrNil()
		}
	}

	var wg1 sync.WaitGroup
	for _, a := range announceList {
		wg1.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, a openapi.Appliance) {
			defer wg.Done()
			if _, err := changeAPI.RetryUntilCompleted(ctx, changeID, a.GetId()); err != nil {
				errs = multierror.Append(errs, err)
			}
		}(ctx, &wg1, a)
	}
	wg1.Wait()

	if t != nil {
		t.Update("re-allocating IP:s")
	}
	changeID, err = a.RepartitionIPAllocations(ctx)
	if err != nil {
		return err
	}
	var wg2 sync.WaitGroup
	for _, ctrl := range announceList {
		wg2.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ctrl openapi.Appliance) {
			defer wg.Done()
			if _, err := changeAPI.RetryUntilCompleted(ctx, changeID, ctrl.GetId()); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("IP re-allocation failed for %s: %s", ctrl.GetName(), err.Error()))
			}
			if err := a.ApplianceStats.WaitForApplianceStatus(ctx, ctrl, appliancepkg.StatusNotBusy, nil); err != nil {
				errs = multierror.Append(errs, err)
			}
		}(ctx, &wg2, ctrl)
	}
	for _, ctrl := range disableList {
		wg2.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ctrl openapi.Appliance) {
			defer wg.Done()
			if err := a.ApplianceStats.WaitForApplianceStatus(ctx, ctrl, appliancepkg.StatusNotBusy, nil); err != nil {
				errs = multierror.Append(errs, err)
			}
		}(ctx, &wg2, ctrl)
	}
	wg2.Wait()

	return errs.ErrorOrNil()
}

const summaryTPLString string = `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

{{ .DisableTable }}{{ if .ShowOfflineTable }}

WARNING:
The following Controllers are unreachable and will likely not receive the announcement. Please confirm that these controllers are, in fact, offline before continuing:

{{ .OfflineTable }}{{ end }}
`

func printSummary(stats []openapi.StatsAppliancesListAllOfData, primaryControllerID string, disable, offline []openapi.Appliance) (string, error) {
	type stub struct {
		DisableTable, OfflineTable string
		ShowOfflineTable           bool
	}
	disableBuffer := &bytes.Buffer{}
	dt := util.NewPrinter(disableBuffer, 4)
	dt.AddHeader("Name", "Hostname", "Status", "Version")

	offlineBuffer := &bytes.Buffer{}
	ot := util.NewPrinter(offlineBuffer, 4)
	ot.AddHeader("Name", "Hostname", "Status", "Version")

	data := stub{
		ShowOfflineTable: false,
	}
	for _, s := range stats {
		for _, a := range disable {
			if s.GetId() == a.GetId() {
				dt.AddLine(a.GetName(), a.GetHostname(), s.GetStatus(), s.GetVersion())
			}
		}
		for _, a := range offline {
			if s.GetId() == a.GetId() {
				data.ShowOfflineTable = true
				ot.AddLine(a.GetName(), a.GetHostname(), s.GetStatus(), s.GetVersion())
			}
		}
	}
	dt.Print()
	ot.Print()

	data.DisableTable = disableBuffer.String()
	data.OfflineTable = offlineBuffer.String()

	tpl := template.Must(template.New("").Parse(summaryTPLString))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
