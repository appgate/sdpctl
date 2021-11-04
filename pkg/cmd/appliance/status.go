package config

import (
	"fmt"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
)

type upgradeStatusOptions struct {
	Config     *config.Config
	APIClient  func(Config *config.Config) (*openapi.APIClient, error)
	Timeout    int
	url        string
	provider   string
	debug      bool
	insecure   bool
	apiversion int
	cacert     string
}

// NewUpgradeStatusCmd return a new upgrade status command
func NewUpgradeStatusCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeStatusOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		Timeout:   10,
		debug:     f.Config.Debug,
	}
	var upgradeStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "upgrade status",
		Long:  `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeStatusRun(c, args, &opts)
		},
	}

	upgradeStatusCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	upgradeStatusCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.Url, "appgate sdp controller API URL")
	upgradeStatusCmd.PersistentFlags().IntVarP(&opts.apiversion, "apiversion", "", f.Config.Version, "peer API version")
	upgradeStatusCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	upgradeStatusCmd.PersistentFlags().StringVarP(&opts.cacert, "cacert", "", "", "Path to the controller's CA cert file in PEM or DER format")

	return upgradeStatusCmd
}

func upgradeStatusRun(cmd *cobra.Command, args []string, opts *upgradeStatusOptions) error {
	fmt.Println("upgrade status placeholder")
	return nil
}
