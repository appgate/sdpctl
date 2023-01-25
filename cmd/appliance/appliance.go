package appliance

import (
	"github.com/appgate/sdpctl/cmd/appliance/backup"
	"github.com/appgate/sdpctl/cmd/appliance/files"
	"github.com/appgate/sdpctl/cmd/appliance/maintenance"
	"github.com/appgate/sdpctl/cmd/appliance/upgrade"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

var (
	filterHelp string = `Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function`
	orderByHelp string = ``
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
	pFlags.Bool("descending", false, "")

	cmd.AddCommand(upgrade.NewUpgradeCmd(f))
	cmd.AddCommand(backup.NewCmdBackup(f))
	cmd.AddCommand(NewListCmd(f))
	cmd.AddCommand(NewStatsCmd(f))
	cmd.AddCommand(NewMetricCmd(f))
	cmd.AddCommand(NewResolveNameCmd(f))
	cmd.AddCommand(NewResolveNameStatusCmd(f))
	cmd.AddCommand(NewLogsCmd(f))
	cmd.AddCommand(files.NewFilesCmd(f))
	cmd.AddCommand(maintenance.NewMaintenanceCmd(f))
	cmd.AddCommand(NewSeedCmd(f))
	cmd.AddCommand(NewForceDisableControllerCmd(f))

	return cmd
}
