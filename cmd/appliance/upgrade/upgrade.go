package upgrade

import (
	"fmt"

	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
)

type upgradeOptions struct {
	Config    *configuration.Config
	APIClient func(Config *configuration.Config) (*openapi.APIClient, error)
	Timeout   int
	url       string
	provider  string
	debug     bool
	insecure  bool
	cacert    string
}

// NewUpgradeCmd return a new upgrade command
func NewUpgradeCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		Timeout:   10,
		debug:     f.Config.Debug,
	}
	var upgradeCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade an appliance",
		Long:  `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeRun(c, args, &opts)
		},
	}

	upgradeCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	upgradeCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	upgradeCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	upgradeCmd.PersistentFlags().StringVarP(&opts.cacert, "cacert", "", "", "Path to the controller's CA cert file in PEM or DER format")

	upgradeCmd.AddCommand(NewUpgradeStatusCmd(f))
	upgradeCmd.AddCommand(NewPrepareUpgradeCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCancelCmd(f))

	return upgradeCmd
}

func upgradeRun(cmd *cobra.Command, args []string, opts *upgradeOptions) error {
	fmt.Println("upgrade placeholder")
	return nil
}
