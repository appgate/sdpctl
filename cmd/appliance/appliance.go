package appliance

import (
	"github.com/appgate/sdpctl/cmd/appliance/backup"
	"github.com/appgate/sdpctl/cmd/appliance/files"
	"github.com/appgate/sdpctl/cmd/appliance/functions"
	"github.com/appgate/sdpctl/cmd/appliance/maintenance"
	"github.com/appgate/sdpctl/cmd/appliance/upgrade"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

var (
	filterHelp string = `Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function`
	orderByHelp string = `Order appliance lists by keywords, i.e. 'name', 'id' etc. Accepts a comma seperated list of keywords, where first mentioned has priority. Applies to the 'appliance list' and 'appliance stats' commands.`
)

// NewApplianceCmd return a new appliance command
func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "appliance",
		Short:            docs.ApplianceRootDoc.Short,
		Long:             docs.ApplianceRootDoc.Long,
		Aliases:          []string{"app", "a"},
		TraverseChildren: true,
	}
	pFlags := cmd.PersistentFlags()
	pFlags.StringToStringP("include", "i", map[string]string{}, "Include appliances. Adheres to the same syntax and key-value pairs as '--exclude'")
	pFlags.StringToStringP("exclude", "e", map[string]string{}, filterHelp)
	pFlags.StringSlice("order-by", []string{"name"}, orderByHelp)
	pFlags.Bool("descending", false, "Change the direction of sort order when using the '--order-by' flag. Using this will reverse the sort order for all keywords specified in the '--order-by' flag.")

	cmd.AddCommand(
		upgrade.NewUpgradeCmd(f),
		backup.NewCmdBackup(f),
		NewListCmd(f),
		NewStatsCmd(f),
		NewMetricCmd(f),
		NewResolveNameCmd(f),
		NewResolveNameStatusCmd(f),
		NewLogsCmd(f),
		files.NewFilesCmd(f),
		maintenance.NewMaintenanceCmd(f),
		NewSeedCmd(f),
		NewForceDisableControllerCmd(f),
		functions.NewApplianceFunctionsCmd(f),
		NewSwitchPartitionCmd(f),
	)

	return cmd
}
