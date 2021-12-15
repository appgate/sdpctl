package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/spf13/cobra"
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
	filter, _ := util.ParseFilterFlag(cmd)
	allAppliances, err := a.List(ctx, filter)
	if err != nil {
		return err
	}
	if opts.json {
		j, err := json.MarshalIndent(&allAppliances, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", string(j))
		return nil
	}
	w := tabwriter.NewWriter(opts.Out, 4, 4, 8, ' ', tabwriter.DiscardEmptyColumns)
	fmt.Fprintln(w, "Name\tHostname\tSite\tActivated")
	for _, a := range allAppliances {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\n", a.GetName(), a.GetHostname(), a.GetSiteName(), a.GetActivated())
	}
	w.Flush()
	return nil
}
