package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
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
	return listCmd
}

func statsRun(cmd *cobra.Command, args []string, opts *statsOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	stats, _, err := a.Stats(ctx)
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
	w.AddHeader("Name", "Status", "Function", "CPU", "Memory", "Network out/in", "Disk", "Version")
	for _, s := range stats.GetData() {
		version := s.GetVersion()
		if v, err := appliancepkg.ParseVersionString(version); err == nil {
			version = v.String()
		}
		w.AddLine(
			s.GetName(),
			s.GetStatus(),
			statsActiveFunction(s),
			fmt.Sprintf("%g%%", s.GetCpu()),
			fmt.Sprintf("%g%%", s.GetMemory()),
			statsNetworkPrettyPrint(s.GetNetwork()),
			fmt.Sprintf("%g%%", s.GetDisk()),
			version,
		)
	}
	w.Print()
	return nil
}

func statsNetworkPrettyPrint(n openapi.StatsAppliancesListAllOfNetwork) string {
	return fmt.Sprintf("%s / %s", n.GetTxSpeed(), n.GetRxSpeed())
}

const na = "n/a"

func statsActiveFunction(s openapi.StatsAppliancesListAllOfData) string {
	functions := make([]string, 0)
	if v, ok := s.GetLogServerOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, appliancepkg.FunctionLogServer)
		}
	}
	if v, ok := s.GetLogForwarderOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, appliancepkg.FunctionLogForwarder)
		}
	}
	if v, ok := s.GetControllerOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, appliancepkg.FunctionController)
		}
	}
	if v, ok := s.GetConnectorOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, appliancepkg.FunctionConnector)
		}
	}
	if v, ok := s.GetGatewayOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, appliancepkg.FunctionGateway)
		}
	}
	return strings.Join(functions, ", ")
}
