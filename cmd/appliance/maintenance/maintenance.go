package maintenance

import (
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

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
