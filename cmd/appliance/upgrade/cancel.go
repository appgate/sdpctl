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
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type upgradeCancelOptions struct {
	Config     *configuration.Config
	Out        io.Writer
	Appliance  func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Token      string
	url        string
	provider   string
	debug      bool
	insecure   bool
	apiversion int
	cacert     string
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
		Use:  "cancel",
		Long: `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCancelRun(c, args, &opts)
		},
	}

	upgradeCancelCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	upgradeCancelCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	upgradeCancelCmd.PersistentFlags().IntVarP(&opts.apiversion, "apiversion", "", f.Config.Version, "peer API version")
	upgradeCancelCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	upgradeCancelCmd.PersistentFlags().StringVarP(&opts.cacert, "cacert", "", "", "Path to the controller's CA cert file in PEM or DER format")

	return upgradeCancelCmd
}

func upgradeCancelRun(cmd *cobra.Command, args []string, opts *upgradeCancelOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	appliances, err := a.GetAll(ctx)
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
	g, ctx := errgroup.WithContext(ctx)
	for _, appliance := range noneIdleAppliances {
		i := appliance
		g.Go(func() error {
			log.Infof("Cancel upgrade on %s - %s", i.GetId(), i.GetName())
			return a.UpgradeCancel(ctx, i.GetId())
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	log.Infof("Upgrade cancelled on %d appliances", len(noneIdleAppliances))
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
