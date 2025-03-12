package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/queue"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type upgradeCancelOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	SpinnerOut    func() io.Writer
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug         bool
	delete        bool
	NoInteractive bool
	defaultfilter map[string]map[string]string
	timeout       time.Duration
	ciMode        bool
}

// NewUpgradeCancelCmd return a new upgrade status command
func NewUpgradeCancelCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeCancelOptions{
		Config:     f.Config,
		Appliance:  f.Appliance,
		debug:      f.Config.Debug,
		Out:        f.IOOutWriter,
		SpinnerOut: f.GetSpinnerOutput(),
		timeout:    DefaultTimeout,
		defaultfilter: map[string]map[string]string{
			"include": {},
			"exclude": {
				"active": "false",
			},
		},
	}
	var upgradeCancelCmd = &cobra.Command{
		Use:     "cancel",
		Short:   docs.ApplianceUpgradeCancelDoc.Short,
		Long:    docs.ApplianceUpgradeCancelDoc.Long,
		Example: docs.ApplianceUpgradeCancelDoc.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCancelRun(c, &opts)
		},
	}

	flags := upgradeCancelCmd.Flags()
	flags.BoolVar(&opts.NoInteractive, "no-interactive", false, "suppress interactive prompt with auto accept")
	flags.BoolVar(&opts.delete, "delete", false, "Delete all upgrade files from the Controller")

	return upgradeCancelCmd
}

func upgradeCancelRun(cmd *cobra.Command, opts *upgradeCancelOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	spinnerOut := opts.SpinnerOut()
	if a.ApplianceStats == nil {
		a.ApplianceStats = &appliancepkg.ApplianceStatus{
			Appliance: a,
		}
	}
	if a.UpgradeStatusWorker == nil {
		a.UpgradeStatusWorker = &appliancepkg.UpgradeStatus{
			Appliance: a,
		}
	}
	ciFlag, err := cmd.Flags().GetBool("ci-mode")
	if err != nil {
		return err
	}
	opts.ciMode = ciFlag

	ctx := context.WithValue(context.Background(), appliancepkg.Caller, cmd.CalledAs())
	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), opts.defaultfilter)
	stats, _, err := a.ApplianceStatus(ctx, nil, orderBy, descending)
	if err != nil {
		return err
	}
	allAppliances, err := a.List(ctx, filter, orderBy, descending)
	if err != nil {
		return err
	}
	appliances, offline, _ := appliancepkg.FilterAvailable(allAppliances, stats.GetData())

	noneIdleAppliances := make([]openapi.Appliance, 0)
	for _, app := range appliances {
		s, err := a.UpgradeStatus(ctx, app.GetId())
		if err != nil {
			return err
		}
		if s.GetStatus() != appliancepkg.UpgradeStatusIdle {
			noneIdleAppliances = append(noneIdleAppliances, app)
		}
	}
	if len(noneIdleAppliances) == 0 {
		log.Infof("Did not find any appliances to perform cancel on")
		return nil
	}
	msg, err := showCancelList(noneIdleAppliances, offline)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", msg)
	if !opts.NoInteractive {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	cancel := func(ctx context.Context, appliances []openapi.Appliance, workers int) error {
		var (
			count = len(appliances)
			// qw is the FIFO queue that will run Upgrade cancel concurrently on number of workers.
			qw = queue.New(count, workers)
			// wantedStatus is the desired state for the queued jobs, we need to limit these jobs, and run them in order
			wantedStatus = []string{
				appliancepkg.UpgradeStatusIdle,
			}
			undesiredStatus = []string{
				appliancepkg.UpgradeStatusReady,
				appliancepkg.UpgradeStatusFailed,
			}
		)
		retryCancel := func(ctx context.Context, appliance openapi.Appliance) error {
			return backoff.Retry(func() error {
				return a.UpgradeCancel(ctx, appliance.GetId())
			}, backoff.NewExponentialBackOff())
		}

		type queueStruct struct {
			appliance openapi.Appliance
			tracker   *tui.Tracker
		}

		var bars *tui.Progress
		if !opts.ciMode {
			bars = tui.New(ctx, spinnerOut)
			defer bars.Wait()
		}
		for _, appliance := range appliances {
			var t *tui.Tracker
			if !opts.ciMode {
				t = bars.AddTracker(appliance.GetName(), "waiting", "cancelled")
				go t.Watch(appliancepkg.StatusNotBusy, undesiredStatus)
			}
			qs := queueStruct{
				appliance: appliance,
				tracker:   t,
			}
			qw.Push(qs)
		}
		err := qw.Work(func(v interface{}) error {
			ctx, cancel := context.WithTimeout(ctx, opts.timeout)
			defer cancel()
			// When cancelling upgrade on a appliance, we will verified that both the upgrade status is OK,
			// and that the apppliance is not busy to avoid race condition When running to many operations
			// on multiple appliances at once.
			qs := v.(queueStruct)
			if err := retryCancel(ctx, qs.appliance); err != nil {
				return fmt.Errorf("Upgrade cancel for %s failed, %w", qs.appliance.GetName(), err)
			}
			if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, qs.appliance, wantedStatus, undesiredStatus, qs.tracker); err != nil {
				return err
			}
			return a.ApplianceStats.WaitForApplianceStatus(ctx, qs.appliance, appliancepkg.StatusNotBusy, qs.tracker)
		})
		if err != nil {
			return err
		}

		return nil
	}
	fmt.Fprintf(opts.Out, "[%s] Cancelling pending upgrades:\n", time.Now().Format(time.RFC3339))
	// workers is intentionally a fixed value of 2
	// because otherwise its a high risk of triggering failure from 1 or more appliances
	if err := cancel(ctx, noneIdleAppliances, 2); err != nil {
		return err
	}

	if opts.delete {
		files, err := a.ListFiles(context.Background(), nil, false)
		if err != nil {
			return err
		}
		for _, f := range files {
			log.Infof("Deleting file %q from the Controller file repository", f.GetName())
			if err := a.DeleteFile(ctx, f.GetName()); err != nil {
				log.Warningf("Unable to delete file %q %s", f.GetName(), err)
			}
		}
		return nil
	}

	return nil
}

const cancelApplianceUpgrade = `
cancelling upgrade on the following appliance:
{{range .Appliances}}
  - {{.Name -}}
{{end}}
{{ if .Offline }}
The following appliances are offline and will be excluded:
{{range .Offline}}
  - {{.Name -}}
{{end}}
{{end}}
`

func showCancelList(online, offline []openapi.Appliance) (string, error) {
	type stub struct {
		Appliances []openapi.Appliance
		Offline    []openapi.Appliance
	}

	data := stub{
		Appliances: online,
		Offline:    offline,
	}
	t := template.Must(template.New("").Parse(cancelApplianceUpgrade))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
