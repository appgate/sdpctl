package cmd

import (
	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/cmd/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type applianceOptions struct {
	Config *config.Config
}

// NewApplianceCmd return a new appliance command
func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	opts := applianceOptions{
		Config: f.Config,
	}
	return &cobra.Command{
		Use:   "appliance",
		Short: "interact with appliance",
		Long:  `TODO`,
		RunE: func(c *cobra.Command, args []string) error {
			return applianceRun(c, args, &opts)
		},
	}
}

func applianceRun(cmd *cobra.Command, args []string, opts *applianceOptions) error {
	log.Infof("Placeholder command")
	return nil
}
