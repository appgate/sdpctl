package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type listOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	json      bool
}

// NewListCmd return a new maintenance enable command
func NewListCmd(f *factory.Factory) *cobra.Command {
	opts := listOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		RunE: func(c *cobra.Command, args []string) error {
			return listRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return cmd
}

func listRun(cmd *cobra.Command, args []string, opts *listOptions) error {
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

	// filter out the only the controller stats
	controllers := make([]openapi.StatsAppliancesListAllOfData, 0)
	notController := "n/a"
	for _, s := range stats.GetData() {
		ctrl := s.GetController()
		if ctrl.GetStatus() != notController {
			controllers = append(controllers, s)
		}
	}
	if opts.json {
		j, err := json.MarshalIndent(&controllers, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", string(j))
		return nil
	}
	w := util.NewPrinter(opts.Out, 4)
	removeNewLine := regexp.MustCompile(`\r?\n`)
	w.AddHeader("Name", "Maintenance mode", "Details")
	for _, s := range controllers {
		ctrl := s.GetController()
		w.AddLine(
			s.GetName(),
			ctrl.GetMaintenanceMode(),
			removeNewLine.ReplaceAllString(ctrl.GetDetails(), " "),
		)

	}
	w.Print()
	return nil
}
