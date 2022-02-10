package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/briandowns/spinner"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type upgradeCancelOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug         bool
	delete        bool
	NoInteractive bool
}

// NewUpgradeCancelCmd return a new upgrade status command
func NewUpgradeCancelCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeCancelOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var upgradeCancelCmd = &cobra.Command{
		Use:   "cancel",
		Short: `Cancel a prepared upgrade`,
		Long: `Cancel a prepared upgrade. The command will attempt to cancel upgrades on
Appliances that are not in the 'idle' upgrade state. Cancelling will remove the uploaded
upgrade image from the Appliance.

Note that you can cancel upgrades on specific appliances by using the '--filter' and/or
'--exclude' flags in combination with this command.`,
		Example: `# Cancel upgrade on all Appgate SDP Appliances
$ sdpctl appliance upgrade cancel

# Cancel upgrade on specific appliance, a gateway in this case
$ sdpctl appliance upgrade cancel --filter=role=gateway`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCancelRun(c, args, &opts)
		},
	}

	flags := upgradeCancelCmd.Flags()
	flags.BoolVar(&opts.NoInteractive, "no-interactive", false, "suppress interactive prompt with auto accept")
	flags.BoolVar(&opts.delete, "delete", false, "Delete all upgrade files from the controller")

	return upgradeCancelCmd
}

func upgradeCancelRun(cmd *cobra.Command, args []string, opts *upgradeCancelOptions) error {
	spin := spinner.New(spinner.CharSets[33], 100*time.Millisecond, spinner.WithFinalMSG("ok\n"))
	spin.Writer = opts.Out
	spin.Suffix = " cancelling"
	defer spin.Stop()
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	filter := util.ParseFilteringFlags(cmd.Flags())
	appliances, err := a.List(ctx, filter)
	if err != nil {
		return err
	}

	noneIdleAppliances := make([]openapi.Appliance, 0)
	for _, app := range appliances {
		s, err := a.UpgradeStatus(ctx, app.Id)
		if err != nil {
			return err
		}
		if s.GetStatus() != appliancepkg.UpgradeStatusIdle {
			noneIdleAppliances = append(noneIdleAppliances, app)
		}
	}
	if len(noneIdleAppliances) == 0 {
		log.Infof("did not find any appliances to perform cancel on.")
		return nil
	}
	msg, err := showCancelList(noneIdleAppliances)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", msg)
	if !opts.NoInteractive {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}
	spin.Start()

	cancel := func(ctx context.Context, appliances []openapi.Appliance) ([]openapi.Appliance, error) {
		g, ctx := errgroup.WithContext(ctx)
		cancelChan := make(chan openapi.Appliance, len(appliances))
		for _, appliance := range noneIdleAppliances {
			i := appliance
			g.Go(func() error {
				log.Infof("Cancel upgrade on %s - %s", i.GetId(), i.GetName())
				if err := a.UpgradeCancel(ctx, i.GetId()); err != nil {
					return err
				}
				select {
				case cancelChan <- i:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			})
		}
		go func() {
			g.Wait()
			close(cancelChan)
		}()
		result := make([]openapi.Appliance, 0)
		for r := range cancelChan {
			result = append(result, r)
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return result, nil
	}
	fmt.Fprint(opts.Out, "Cancelling pending upgrades...")
	cancelled, err := cancel(ctx, noneIdleAppliances)
	if err != nil {
		return err
	}
	log.Infof("Upgrade cancelled on %d/%d appliances", len(cancelled), len(noneIdleAppliances))

	if opts.delete {
		files, err := a.ListFiles(context.Background())
		if err != nil {
			return err
		}
		for _, f := range files {
			log.Infof("deleting file %q from controller file repository", f.GetName())
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
`

func showCancelList(a []openapi.Appliance) (string, error) {
	type stub struct {
		Appliances []openapi.Appliance
	}

	data := stub{
		Appliances: a,
	}
	t := template.Must(template.New("").Parse(cancelApplianceUpgrade))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
