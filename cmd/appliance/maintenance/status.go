package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	json      bool
}

// NewStatusCmd return a new maintenance enable command
func NewStatusCmd(f *factory.Factory) *cobra.Command {
	opts := statusOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:   "status",
		Short: docs.MaintenanceStatusDoc.Short,
		Long:  docs.MaintenanceStatusDoc.Long,
		Annotations: map[string]string{
			"MinAPIVersion": "18",
			"ErrorMessage":  "sdpctl appliance maintenance status requires appliance version higher or equal to 6.1 with API Version 18",
		},
		RunE: func(c *cobra.Command, args []string) error {
			return listRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return cmd
}

func listRun(cmd *cobra.Command, args []string, opts *statusOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	stats, _, err := a.DeprecatedStats(ctx, nil, nil, false)
	if err != nil {
		return err
	}

	// filter out the only the Controller stats
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
