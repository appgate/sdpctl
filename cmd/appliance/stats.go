package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
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
		Use:   "stats",
		Short: `show appliance stats`,
		RunE: func(c *cobra.Command, args []string) error {
			return statusRun(c, args, &opts)
		},
	}
	listCmd.PersistentFlags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return listCmd
}

func statusRun(cmd *cobra.Command, args []string, opts *statsOptions) error {
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
	w := tabwriter.NewWriter(opts.Out, 4, 4, 8, ' ', tabwriter.DiscardEmptyColumns)
	fmt.Fprintln(w, "Name\tStatus\tFunction\tCPU\tMemory\tNetwork out/in\tDisk\tVersion")
	for _, s := range stats.GetData() {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.GetName(),
			s.GetStatus(),
			statsActiveFunction(s),
			fmt.Sprintf("%g%%", s.GetCpu()),
			fmt.Sprintf("%g%%", s.GetMemory()),
			statsNetworkPrettyPrint(s.GetNetwork()),
			fmt.Sprintf("%g%%", s.GetDisk()),
			s.GetVersion(),
		)
	}
	w.Flush()
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
			functions = append(functions, "log server")
		}
	}
	if v, ok := s.GetLogForwarderOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, "log forwader")
		}
	}
	if v, ok := s.GetControllerOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, "controller")
		}
	}
	if v, ok := s.GetConnectorOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, "Connector")
		}
	}
	if v, ok := s.GetGatewayOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, "gateway")
		}
	}
	return strings.Join(functions, ", ")
}
