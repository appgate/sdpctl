package maintenance

import (
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

// NewDisableCmd return a new maintenance disable command
func NewDisableCmd(f *factory.Factory) *cobra.Command {
	opts := toggleOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		enabled:   false,
	}

	var cmd = &cobra.Command{
		Use:     "disable <applianceUUID>",
		Short:   docs.MaintenanceDisable.Short,
		Long:    docs.MaintenanceDisable.Long,
		Example: docs.MaintenanceDisable.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return toggleArgs(cmd, &opts, args)
		},
		RunE: func(c *cobra.Command, args []string) error {
			return toggleRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return cmd
}
