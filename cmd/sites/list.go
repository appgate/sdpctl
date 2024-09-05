package sites

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type SitesListOptions struct {
	SitesOptions
	json bool
}

func NewSitesListCmd(parentOpts *SitesOptions) *cobra.Command {
	opts := &SitesListOptions{}
	opts.SitesAPI = parentOpts.SitesAPI
	opts.Out = parentOpts.Out
	opts.CiMode = parentOpts.CiMode
	opts.NoInteractive = parentOpts.NoInteractive

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   docs.SitesListDocs.Short,
		Long:    docs.SitesListDocs.Long,
		Example: docs.SitesListDocs.ExampleString(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			sites, err := opts.SitesAPI.ListSites(ctx)
			if err != nil {
				return err
			}

			if opts.json {
				o, err := json.MarshalIndent(sites, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.Out, string(o))
			} else {
				p := util.NewPrinter(opts.Out, 4)
				p.AddHeader("Site Name", "Short Name", "ID", "Tags", "Description")
				for _, s := range sites {
					p.AddLine(s.GetName(), s.GetShortName(), s.GetId(), s.GetTags(), s.GetDescription())
				}
				p.Print()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.json, "json", false, "")

	return cmd
}
