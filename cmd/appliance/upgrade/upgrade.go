package upgrade

import (
	"time"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

const (
	DefaultTimeout = 30 * time.Minute
)

// NewUpgradeCmd return a new upgrade command
func NewUpgradeCmd(f *factory.Factory) *cobra.Command {
	var upgradeCmd = &cobra.Command{
		Use:              "upgrade",
		TraverseChildren: true,
		Short:            docs.ApplianceUpgradeDoc.Short,
		Long:             docs.ApplianceUpgradeDoc.Long,
	}

	upgradeCmd.AddCommand(NewUpgradeStatusCmd(f))
	upgradeCmd.AddCommand(NewPrepareUpgradeCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCancelCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCompleteCmd(f))

	flags := upgradeCmd.PersistentFlags()
	flags.DurationP("timeout", "t", 30*time.Minute, "Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on.")

	return upgradeCmd
}
