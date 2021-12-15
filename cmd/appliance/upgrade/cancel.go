package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type upgradeCancelOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Token     string
	url       string
	provider  string
	debug     bool
	insecure  bool
	delete    bool
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
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCancelRun(c, args, &opts)
		},
	}

	flags := upgradeCancelCmd.Flags()
	flags.BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	flags.StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	flags.StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	flags.BoolVar(&opts.delete, "delete", false, "Delete all upgrade files from the controller")

	return upgradeCancelCmd
}

func upgradeCancelRun(cmd *cobra.Command, args []string, opts *upgradeCancelOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	filter, _ := util.ParseFilterFlag(cmd)
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
	if err := prompt.AskConfirmation(); err != nil {
		return err
	}
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
