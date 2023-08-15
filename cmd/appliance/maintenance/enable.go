package maintenance

import (
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

// NewEnableCmd return a new maintenance enable command
func NewEnableCmd(f *factory.Factory) *cobra.Command {
	opts := toggleOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		enabled:   true,
	}
	var cmd = &cobra.Command{
		Use:     "enable <applianceUUID>",
		Short:   docs.MaintenanceEnable.Short,
		Long:    docs.MaintenanceEnable.Long,
		Example: docs.MaintenanceEnable.ExampleString(),
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
