package upgrade

import (
	"time"

	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

// NewUpgradeCmd return a new upgrade command
func NewUpgradeCmd(f *factory.Factory) *cobra.Command {
	var upgradeCmd = &cobra.Command{
		Use:              "upgrade",
		TraverseChildren: true,
		Short:            "Perform appliance upgrade on the Appgate SDP Collective",
		Long: `The upgrade procedure is divided into two parts,
  - prepare: Upload the image new appliance image to the Appgate SDP Collective.
  - complete: Install a prepared upgrade on the secondary partition and perform a reboot to make the second partition the primary.

Additional subcommands included are:
  - status: view the current upgrade status on all appliances.
  - cancel: Cancel a prepared upgrade.
`,
	}

	upgradeCmd.AddCommand(NewUpgradeStatusCmd(f))
	upgradeCmd.AddCommand(NewPrepareUpgradeCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCancelCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCompleteCmd(f))

	flags := upgradeCmd.PersistentFlags()
	flags.DurationP("timeout", "t", 30*time.Minute, "Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on.")
	flags.Int("throttle", 5, "Upgrade is done in batches using a throttle value. You can control the throttle using this flag.")

	return upgradeCmd
}
