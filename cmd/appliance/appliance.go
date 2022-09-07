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
	filterHelp string = `Filter appliances using a comma separated list of key-value pairs. Example: '--include name=controller,site=<site-id> etc.'.
Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function`
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
	pFlags.StringToStringP("include", "i", map[string]string{}, filterHelp)
	pFlags.StringToStringP("exclude", "e", map[string]string{}, "Exclude appliances. Adheres to the same syntax and key-value pairs as '--include'")

	cmd.AddCommand(upgrade.NewUpgradeCmd(f))
	cmd.AddCommand(backup.NewCmdBackup(f))
	cmd.AddCommand(NewListCmd(f))
	cmd.AddCommand(NewStatsCmd(f))
	cmd.AddCommand(NewMetricCmd(f))
	cmd.AddCommand(NewResolveNameCmd(f))
	cmd.AddCommand(NewResolveNameStatusCmd(f))
	cmd.AddCommand(files.NewFilesCmd(f))
	cmd.AddCommand(maintenance.NewToggleCmd(f))

	return cmd
}
