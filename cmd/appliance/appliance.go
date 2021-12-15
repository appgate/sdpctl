package appliance

import (
	"github.com/appgate/appgatectl/cmd/appliance/backup"
	"github.com/appgate/appgatectl/cmd/appliance/upgrade"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

// NewApplianceCmd return a new appliance command
func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appliance",
		Short: "interact with appliances",
	}
	cmd.AddCommand(upgrade.NewUpgradeCmd(f))
	cmd.AddCommand(backup.NewCmdBackup(f))
	cmd.AddCommand(NewListCmd(f))
	cmd.AddCommand(NewStatsCmd(f))
	cmd.PersistentFlags().StringSliceP("filter", "f", []string{}, "Filter appliances")
	cmd.PersistentFlags().StringSliceP("exclude", "e", []string{}, "Exclude appliances")

	return cmd
}
