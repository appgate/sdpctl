package appliance

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type statsOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug     bool
	json      bool
}

var (
	filterStatsHelp string = `Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
Available keywords to filter on are: name, id, status, state and function`
)

// NewStatsCmd return a new appliance stats list command
func NewStatsCmd(f *factory.Factory) *cobra.Command {
	opts := statsOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var listCmd = &cobra.Command{
		Use:     "stats",
		Short:   docs.ApplianceStatsDocs.Short,
		Long:    docs.ApplianceStatsDocs.Long,
		Example: docs.ApplianceStatsDocs.ExampleString(),
		Aliases: []string{"status"},
		RunE: func(c *cobra.Command, args []string) error {
			return statsRun(c, args, &opts)
		},
	}
	listCmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	listCmd.PersistentFlags().StringToStringP("include", "i", map[string]string{}, "Include appliance stats. Adheres to the same syntax and key-value pairs as '--exclude'")
	listCmd.PersistentFlags().StringToStringP("exclude", "e", map[string]string{}, filterStatsHelp)
	return listCmd
}

func statsRun(cmd *cobra.Command, args []string, opts *statsOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), appliancepkg.DefaultCommandFilter)
	ctx := util.BaseAuthContext(a.Token)
	stats, _, err := a.ApplianceStatus(ctx, filter, orderBy, descending)
	if err != nil {
		return err
	}
	if opts.json {
		j, err := json.MarshalIndent(&stats, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", string(j))
		return nil
	}
	w := util.NewPrinter(opts.Out, 4)
	diskHeader := "Disk"
	if cfg.Version >= 18 {
		diskHeader += " (used / total)"
	}
	w.AddHeader("Name", "Status", "Function", "CPU", "Memory", "Network out/in", diskHeader, "Version", "Sessions")
	for _, s := range stats.GetData() {
		version := s.GetApplianceVersion()
		if v, err := appliancepkg.ParseVersionString(version); err == nil {
			version = v.String()
		}
		w.AddLine(
			s.GetName(),
			s.GetStatus(),
			appliancepkg.ApplianceActiveFunctions(s),
			fmt.Sprintf("%g%%", s.GetCpu()),
			fmt.Sprintf("%g%%", s.GetMemory()),
			statsNetworkPrettyPrint(s.GetDetails().Network),
			statsDiskUsage(s),
			version,
			s.GetNumberOfSessions(),
		)
	}
	w.Print()
	return nil
}

func statsNetworkPrettyPrint(n *openapi.NetworkInfo) string {
	busiestNic := n.GetBusiestNic()
	nicDetails := n.GetDetails()[busiestNic]
	return fmt.Sprintf("%s / %s", nicDetails.GetTxSpeed(), nicDetails.GetRxSpeed())
}

func statsDiskUsage(stats openapi.ApplianceWithStatus) string {
	if diskInfo := stats.GetDetails().Disk; diskInfo != nil {
		used, total := diskInfo.GetUsed(), diskInfo.GetTotal()
		percentUsed := (float32(used) / float32(total)) * 100
		return fmt.Sprintf("%.2f%% (%s / %s)", percentUsed, appliancepkg.PrettyBytes(float64(used)), appliancepkg.PrettyBytes(float64(total)))
	}
	return fmt.Sprintf("%g%%", stats.GetDisk())
}
