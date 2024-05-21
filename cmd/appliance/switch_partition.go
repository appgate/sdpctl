package appliance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type options struct {
	cfg            *configuration.Config
	api            *appliancepkg.Appliance
	id             *uuid.UUID
	out            io.Writer
	spinnerOut     func() io.Writer
	canPrompt      bool
	ciMode         bool
	applianceStats *openapi.StatsAppliancesListAllOfData
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
			opts.id = &uid
			return nil
		}),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			api, err := f.Appliance(opts.cfg)
			if err != nil {
				return err
			}
			opts.api = api

			ctx := context.Background()
			if opts.id == nil {
				id, err := appliancepkg.PromptSelect(ctx, api, nil, nil, false)
				if err != nil {
					return err
				}
				uid, err := uuid.Parse(id)
				if err != nil {
					return err
				}
				opts.id = &uid
			}

			if opts.id == nil {
				return fmt.Errorf("failed to switch partition: no appliance identifier provided")
			}

			stats, _, err := api.Stats(ctx, nil, nil, false)
			if err != nil {
				return err
			}
			for _, s := range stats.GetData() {
				if opts.id.String() != s.GetId() {
					continue
				}
				opts.applianceStats = &s
			}

			minVersion, _ := version.NewVersion("6.2.6-0")
			currentVersion, err := version.NewVersion(opts.applianceStats.GetVersion())
			if err != nil {
				return err
			}
			i, err := appliancepkg.CompareVersionsAndBuildNumber(minVersion, currentVersion)
			if err != nil {
				return err
			}
			if i < 0 {
				return fmt.Errorf("minimum supported version for the 'switch-partition' command is 6.2.6. current version is %s", currentVersion.String())
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return switchPartitionRunE(opts)
		},
	}

	return cmd
}

func switchPartitionRunE(opts *options) error {
	ctx := context.Background()
	api := opts.api
	if api.ApplianceStats == nil {
		api.ApplianceStats = &appliancepkg.ApplianceStatus{
			Appliance: api,
		}
	}

	appliance, err := api.Get(ctx, opts.id.String())
	if err != nil {
		return fmt.Errorf("failed to get appliance: %w", err)
	}

	volume := opts.applianceStats.GetVolumeNumber()
	log := logrus.WithFields(logrus.Fields{
		"id":     opts.id.String(),
		"volume": volume,
	})
	log.Info("initial appliance stats")

	summary, err := switchPartitionSummary(appliance.GetName())
	if err != nil {
		return err
	}
	fmt.Fprintln(opts.out, summary)
	if opts.canPrompt {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	var p *tui.Progress
	var t *tui.Tracker
	if !opts.ciMode {
		p = tui.New(ctx, opts.spinnerOut())
		t = p.AddTracker(appliance.GetName(), opts.applianceStats.GetState(), "finished")
		go t.Watch(appliancepkg.StatusNotBusy, []string{"error"})
	}

	log.Info("switching partition")
	err = backoff.Retry(func() error {
		log.Debug("calling switch-partition on appliance")
		return api.ApplianceSwitchPartition(ctx, opts.id.String())
	}, backoff.NewExponentialBackOff())
	if err != nil {
		if t != nil {
			t.Update("error")
			t.Fail(err.Error())
		}
		log.WithError(err).Error("partition switch failed")
		return fmt.Errorf("partition switch failed on appliance %s: %v", opts.id, err)
	}

	log.Info("polling for appliance state")
	if err := api.ApplianceStats.WaitForApplianceState(ctx, *appliance, appliancepkg.StatReady, t); err != nil {
		if t != nil {
			t.Update("error")
			t.Fail(err.Error())
		}
		log.WithError(err).Error("failed to get appliance state")
		return fmt.Errorf("failed to get appliance stats: %w", err)
	}

	if err := api.ApplianceStats.WaitForApplianceStatus(ctx, *appliance, appliancepkg.StatusNotBusy, t); err != nil {
		return err
	}

	if p != nil {
		p.Wait()
	}

	// verify partition switch
	stats, _, err := api.Stats(ctx, nil, nil, false)
	if err != nil {
		return fmt.Errorf("partition switch failed: %w", err)
	}
	var newVolume float32
	for _, a := range stats.GetData() {
		if a.GetId() == appliance.GetId() {
			newVolume = a.GetVolumeNumber()
		}
	}
	log.WithField("new_volume", newVolume).Info("new stats after partition switch")

	if newVolume == volume {
		return fmt.Errorf("partition switch failed: volume number is the same as before executing the command")
	}

	fmt.Fprintf(opts.out, "switched partition on %s", appliance.GetName())

	return nil
}

func switchPartitionSummary(applianceName string) (string, error) {
	type tplStub struct {
		Name string
	}
	tplString := `Confirm partition switch on appliance {{ .Name }}`

	tpl, err := template.New("").Parse(tplString)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, tplStub{Name: applianceName}); err != nil {
		return "", err
	}

	return buf.String(), nil
}
