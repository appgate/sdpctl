package maintenance

import (
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

func filter(hostname string) map[string]map[string]string {
	return map[string]map[string]string{
		"include": {
			"function": "controller",
		},
		"exclude": {
			"hostname": hostname,
		},
	}
}

// NewMaintenanceCmd return a new subcommand for maintenance
func NewMaintenanceCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "maintenance",
		TraverseChildren: true,
		Short:            docs.MaintenanceRootDoc.Short,
		Long:             docs.MaintenanceRootDoc.Long,
	}
	cmd.AddCommand(NewToggleCmd(f))
	cmd.AddCommand(NewEnableCmd(f))
	cmd.AddCommand(NewDisableCmd(f))
	return cmd
}
