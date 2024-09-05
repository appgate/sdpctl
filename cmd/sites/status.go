package sites

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type SiteStatusOptions struct {
	SitesOptions
	json bool
}

func NewSitesStatusCmd(f *factory.Factory, parentOpts *SitesOptions) *cobra.Command {
	opts := SiteStatusOptions{}
	opts.SitesAPI = parentOpts.SitesAPI
	opts.Out = parentOpts.Out
	opts.CiMode = parentOpts.CiMode
	opts.NoInteractive = parentOpts.NoInteractive

	cmd := &cobra.Command{
		Use: "status [<site-id>]",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			siteStatuses, err := opts.SitesAPI.SiteStatus(ctx)
			if err != nil {
				return err
			}

			if opts.json {
				data, err := json.MarshalIndent(siteStatuses, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.Out, string(data))
			} else {
				p := util.NewPrinter(opts.Out, 4)
				p.AddHeader("Name", "Status")
				for _, s := range siteStatuses {
					p.AddLine(s.GetName(), s.GetStatus())
				}
				p.Print()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.json, "json", false, "")

	return cmd
}
