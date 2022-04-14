package appliance

import (
	"context"
	"io"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
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
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		defaultFilter: map[string]map[string]string{
			"filter":  {},
			"exclude": {},
		},
	}
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: `List all Appgate SDP Appliances`,
		Long: `List all Appliances in the Appgate SDP Collective. The appliances will be listed in no particular order. Using without arguments
will print a table view with a limited set of information. Using the command with the provided '--json' flag will print out a more detailed
list view in json format. The list command can also be combined with the global '--filter' and '--exclude' flags`,
		Example: `sdpctl appliance list
sdpctl appliance list --json
sdpctl appliance list --filter=<key>=<value>`,
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
	filter := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	allAppliances, err := a.List(ctx, filter)
	if err != nil {
		return err
	}
	if opts.json {
		return util.PrintJSON(opts.Out, allAppliances)
	}

	p := util.NewPrinter(opts.Out)
	p.AddHeader("Name", "ID", "Hostname", "Site", "Activated")
	for _, a := range allAppliances {
		p.AddLine(a.GetName(), a.GetId(), a.GetHostname(), a.GetSiteName(), a.GetActivated())
	}
	p.Print()

	return nil
}
