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
		Use:              "appliance",
		Short:            "interact with appliances",
		TraverseChildren: true,
	}
	pFlags := cmd.PersistentFlags()
	pFlags.StringToStringP("filter", "f", map[string]string{}, "Filter appliances")
	pFlags.StringToStringP("exclude", "e", map[string]string{}, "Exclude appliances")

	cmd.AddCommand(upgrade.NewUpgradeCmd(f))
	cmd.AddCommand(backup.NewCmdBackup(f))
	cmd.AddCommand(NewListCmd(f))
	cmd.AddCommand(NewStatsCmd(f))

	return cmd
}
