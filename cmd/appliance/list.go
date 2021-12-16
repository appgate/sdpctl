package appliance

import (
	"context"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/spf13/cobra"
	"io"
)

type listOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug     bool
	json      bool
}

// NewListCmd return a new appliance list command
func NewListCmd(f *factory.Factory) *cobra.Command {
	opts := listOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   `list all appliances`,
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
	filter := util.ParseFilteringFlags(cmd.Flags())
	allAppliances, err := a.List(ctx, filter)
	if err != nil {
		return err
	}
	if opts.json {
		return util.PrintJson(opts.Out, allAppliances)
	}

	p := util.NewPrinter(opts.Out)
	p.AddHeader("Name", "Hostname", "Site", "Activated")
	for _, a := range allAppliances {
		p.AddLine(a.GetName(), a.GetHostname(), a.GetSiteName(), a.GetActivated())
	}
	p.Print()

	return nil
}
