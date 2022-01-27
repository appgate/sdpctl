package appliance

import (
	"github.com/appgate/appgatectl/cmd/appliance/backup"
	"github.com/appgate/appgatectl/cmd/appliance/upgrade"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

var (
	filterHelp string = `Filter appliances using a comma seperated list of key-value pairs. Example: '--filter name=controller,site=<site-id> etc.'.
Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function|roles|role`
)

// NewApplianceCmd return a new appliance command
func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "appliance",
		Short:            "interact with appliances",
		Aliases:          []string{"app", "a"},
		TraverseChildren: true,
	}
	pFlags := cmd.PersistentFlags()
	pFlags.Bool("no-interactive", false, "suppress interactive prompt with auto accept")
	pFlags.StringToStringP("filter", "f", map[string]string{}, filterHelp)
	pFlags.StringToStringP("exclude", "e", map[string]string{}, "Exclude appliances. Adheres to the same syntax and key-value pairs as '--filter'")

	cmd.AddCommand(upgrade.NewUpgradeCmd(f))
	cmd.AddCommand(backup.NewCmdBackup(f))
	cmd.AddCommand(NewListCmd(f))
	cmd.AddCommand(NewStatsCmd(f))
	cmd.AddCommand(NewMetricCmd(f))
	cmd.AddCommand(NewResolveNameCmd(f))

	return cmd
}
