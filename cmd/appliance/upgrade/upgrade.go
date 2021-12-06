package upgrade

import (
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

// NewUpgradeCmd return a new upgrade command
func NewUpgradeCmd(f *factory.Factory) *cobra.Command {

	var upgradeCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "Perform appliance upgrade on the Appgate SDP Collective",
		Long: `Appgate SDP upgrade script.
© 2021 Cyxtera Cybersecurity, Inc. d/b/a Appgate
All rights reserved. Appgate is a trademark of Cyxtera Cybersecurity, Inc. d/b/a Appgate

https://www.appgate.com

For more documentation on the upgrade process, go to:
    https://sdphelp.appgate.com/adminguide/v5.5/upgrading-appliances.html?anchor=collective-upgrade`,
	}

	upgradeCmd.AddCommand(NewUpgradeStatusCmd(f))
	upgradeCmd.AddCommand(NewPrepareUpgradeCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCancelCmd(f))
	upgradeCmd.AddCommand(NewUpgradeCompleteCmd(f))

	return upgradeCmd
}
