package appliance

import (
	"context"
	"io"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type listOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug         bool
	json          bool
	defaultFilter map[string]map[string]string
}

// NewListCmd return a new appliance list command
func NewListCmd(f *factory.Factory) *cobra.Command {
	opts := listOptions{
		Config:        f.Config,
		Appliance:     f.Appliance,
		debug:         f.Config.Debug,
		Out:           f.IOOutWriter,
		defaultFilter: appliancepkg.DefaultCommandFilter,
	}
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   docs.ApplianceListDoc.Short,
		Long:    docs.ApplianceListDoc.Short,
		Example: docs.ApplianceListDoc.ExampleString(),
		Aliases: []string{"ls"},
		RunE: func(c *cobra.Command, args []string) error {
			return listRun(c, args, &opts)
		},
	}
	listCmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return listCmd
}

func listRun(cmd *cobra.Command, args []string, opts *listOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	allAppliances, err := a.List(ctx, filter, orderBy, descending)
	if err != nil {
		return err
	}
	if opts.json {
		return util.PrintJSON(opts.Out, allAppliances)
	}

	p := util.NewPrinter(opts.Out, 4)
	p.AddHeader("Name", "ID", "Hostname", "Site", "Activated")
	for _, a := range allAppliances {
		p.AddLine(a.GetName(), a.GetId(), a.GetHostname(), a.GetSiteName(), a.GetActivated())
	}
	p.Print()

	return nil
}
