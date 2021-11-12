package cmd

import (
	"github.com/appgate/appgatectl/cmd/appliance/upgrade"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type applianceOptions struct {
	Config *configuration.Config
}

// NewApplianceCmd return a new appliance command
func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	opts := applianceOptions{
		Config: f.Config,
	}
	cmd := &cobra.Command{
		Use:   "appliance",
		Short: "interact with appliance",
		Long:  `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return applianceRun(c, args, &opts)
		},
	}

	cmd.AddCommand(upgrade.NewUpgradeCmd(f))

	return cmd
}

func applianceRun(cmd *cobra.Command, args []string, opts *applianceOptions) error {
	log.Infof("Placeholder command")
	return nil
}
