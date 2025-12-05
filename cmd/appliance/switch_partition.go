package appliance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type options struct {
	cfg            *configuration.Config
	api            *appliancepkg.Appliance
	ids            *[]uuid.UUID
	out            io.Writer
	spinnerOut     func() io.Writer
	canPrompt      bool
	ciMode         bool
	applianceStats *[]openapi.ApplianceWithStatus
}

func NewSwitchPartitionCmd(f *factory.Factory) *cobra.Command {
	opts := &options{
		cfg:        f.Config,
		out:        f.IOOutWriter,
		spinnerOut: f.GetSpinnerOutput(),
		canPrompt:  f.CanPrompt(),
	}
	cmd := &cobra.Command{
		Use:     "switch-partition",
		Short:   docs.ApplianceSwitchPartitionDocs.Short,
		Long:    docs.ApplianceSwitchPartitionDocs.Long,
		Example: docs.ApplianceSwitchPartitionDocs.ExampleString(),
		Annotations: map[string]string{
			"MinAPIVersion": "19",
		},
		Args: cobra.MatchAll(cobra.MaximumNArgs(1), func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				if !opts.canPrompt {
					return fmt.Errorf("no TTY present and no appliance ID provided")
				}
				return nil
			}
			id := args[0]
			var err error
			if !util.IsUUID(id) {
				return fmt.Errorf("'%s' is not a valid appliance ID", id)
			}

			uid, err := uuid.Parse(id)
			if err != nil {
				return err
			}
			opts.ids = &[]uuid.UUID{uid}
			return nil
		}),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			api, err := f.Appliance(opts.cfg)
			if err != nil {
				return err
			}
			opts.api = api
			filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), appliancepkg.DefaultCommandFilter)
			ctx := util.BaseAuthContext(api.Token)
			if opts.ids == nil {
				ids, err := appliancepkg.PromptMultiSelect(ctx, api, filter, orderBy, descending)
				if err != nil {
					return err
				}
				opts.ids = &[]uuid.UUID{}
				for _, idStr := range ids {
					uid, err := uuid.Parse(idStr)
					if err != nil {
						return err
					}
					*opts.ids = append(*opts.ids, uid)
				}
			}

			if opts.ids == nil || len(*opts.ids) == 0 {
				return fmt.Errorf("failed to switch partition: no appliance identifier provided")
			}

			stats, _, err := api.ApplianceStatus(ctx, nil, nil, false)
			if err != nil {
				return err
			}
			opts.applianceStats = &[]openapi.ApplianceWithStatus{}
			for _, uid := range *opts.ids {

				for _, s := range stats.GetData() {
					if uid.String() != s.GetId() {
						continue
					}
					*opts.applianceStats = append(*opts.applianceStats, s)
				}

				minVersion, _ := version.NewVersion("6.2.10-0")
				currentVersion, err := version.NewVersion((*opts.applianceStats)[len(*opts.applianceStats)-1].GetApplianceVersion())
				if err != nil {
					return err
				}
				i, err := appliancepkg.CompareVersionsAndBuildNumber(minVersion, currentVersion)
				if err != nil {
					return err
				}
				if i < 0 {
					return fmt.Errorf("minimum supported version for the 'switch-partition' command is 6.2.10. current version is %s", currentVersion.String())
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return switchPartitionRunE(opts)
		},
	}

	return cmd
}

type partitionSwitched struct {
	applianceID string
}

type switchPartitionInfo struct {
	appliance *openapi.Appliance
	volume    *int32
	logger    *logrus.Entry
}

func switchPartitionRunE(opts *options) error {
	api := opts.api
	ctx := util.BaseAuthContext(opts.api.Token)
	if api.ApplianceStats == nil {
		api.ApplianceStats = &appliancepkg.ApplianceStatus{
			Appliance: api,
		}
	}

	var applianceNames []string
	var partitionsToSwitch []switchPartitionInfo
	for i, id := range *opts.ids {
		appliance, err := api.Get(ctx, id.String())
		if err != nil {
			return fmt.Errorf("failed to get appliance: %w", err)
		}

		volume := (*opts.applianceStats)[i].GetDetails().VolumeNumber
		log := logrus.WithFields(logrus.Fields{
			"id":     id.String(),
			"volume": *volume,
		})
		log.Info("initial appliance stats")
		partitionsToSwitch = append(partitionsToSwitch, switchPartitionInfo{
			appliance: appliance,
			volume:    volume,
			logger:    log,
		})
		applianceNames = append(applianceNames, appliance.GetName())
	}

	summary, err := switchPartitionSummary(applianceNames)
	if err != nil {
		return err
	}
	fmt.Fprintln(opts.out, summary)
	if opts.canPrompt {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	var (
		wg                 sync.WaitGroup
		count              = len(*opts.ids)
		partitionsSwitched = make(chan partitionSwitched, count)
		errorChannel       = make(chan error, count)
		progressBars       *tui.Progress
	)

	wg.Add(count)

	if !opts.ciMode {
		progressBars = tui.New(ctx, opts.spinnerOut())
	}
	for i, info := range partitionsToSwitch {
		var t *tui.Tracker
		if !opts.ciMode {
			t = progressBars.AddTracker(info.appliance.GetName(), (*opts.applianceStats)[i].GetState(), "finished")
			go t.Watch(appliancepkg.StatusNotBusy, []string{"error"})
		}
		go func(switchInfo switchPartitionInfo, tracker *tui.Tracker) {
			defer wg.Done()
			partitionSwitchDone, err := doSwitchPartition(ctx, opts.api, switchInfo, tracker)
			if tracker != nil {
				tracker.Update("finished")
			}
			if err != nil {
				errorChannel <- fmt.Errorf("Partition switch failed for %s: %s", switchInfo.appliance.GetName(), err)
				return
			}

			partitionsSwitched <- partitionSwitchDone
		}(info, t)
	}
	go func() {
		wg.Wait()
		close(partitionsSwitched)
		close(errorChannel)
	}()
	wg.Wait()

	var errs *multierror.Error
	for err := range errorChannel {
		errs = multierror.Append(errs, err)
	}
	if errs.ErrorOrNil() != nil {
		return errs.ErrorOrNil()
	}

	// verify partition switch
	stats, _, err := api.ApplianceStatus(ctx, nil, nil, false)
	if err != nil {
		return fmt.Errorf("partition switch failed: %w", err)
	}
	for _, switchInfo := range partitionsToSwitch {
		var newVolume int32
		for _, a := range stats.GetData() {
			if a.GetId() == switchInfo.appliance.GetId() {
				newVolume = *a.GetDetails().VolumeNumber
			}
		}
		logrus.WithField("new_volume", newVolume).Info("new stats after partition switch")

		if newVolume == *switchInfo.volume {
			errs = multierror.Append(errs, fmt.Errorf("partition switch failed; volume number is unchanged for: %s", switchInfo.appliance.Name))
		} else {
			fmt.Fprintf(opts.out, "switched partition on %s\n", switchInfo.appliance.GetName())
		}
	}

	return errs.ErrorOrNil()
}

func switchPartitionSummary(applianceNames []string) (string, error) {
	type tplStub struct {
		Names string
	}
	tplString := `Confirm partition switch on appliances {{ .Names }}`

	tpl, err := template.New("").Parse(tplString)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, tplStub{Names: strings.Join(applianceNames, ", ")}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func doSwitchPartition(ctx context.Context, api *appliancepkg.Appliance, switchInfo switchPartitionInfo, tracker *tui.Tracker) (partitionSwitched, error) {
	log := switchInfo.logger
	appliance := switchInfo.appliance
	log.Info("switching partition")
	err := backoff.Retry(func() error {
		log.Debug("calling switch-partition on appliance")
		return api.ApplianceSwitchPartition(ctx, *appliance.Id)
	}, backoff.NewExponentialBackOff())
	if err != nil {
		if tracker != nil {
			tracker.Update("error")
			tracker.Fail(err.Error())
		}
		log.WithError(err).Error("partition switch failed")
		return partitionSwitched{
			applianceID: *appliance.Id,
		}, fmt.Errorf("partition switch failed on appliance %s: %v", *appliance.Id, err)
	}
	time.Sleep(time.Duration(3) * time.Second) // Wait a bit before polling for state
	log.Info("polling for appliance state")
	if err := api.ApplianceStats.WaitForApplianceState(ctx, *appliance, appliancepkg.StatReady, tracker); err != nil {
		if tracker != nil {
			tracker.Update("error")
			tracker.Fail(err.Error())
		}
		log.WithError(err).Error("failed to get appliance state")
		return partitionSwitched{
			applianceID: *appliance.Id,
		}, fmt.Errorf("failed to get appliance stats: %w", err)
	}

	if err := api.ApplianceStats.WaitForApplianceStatus(ctx, *appliance, appliancepkg.StatusNotBusy, tracker); err != nil {
		return partitionSwitched{
			applianceID: *appliance.Id,
		}, err
	}
	return partitionSwitched{
		applianceID: *appliance.Id,
	}, nil
}
