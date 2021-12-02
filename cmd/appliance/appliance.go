package cmd

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

	return cmd
}
