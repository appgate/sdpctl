package sites

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type SitesListOptions struct {
	SitesOptions
	ids, siteNames []string
	json           bool
}

func NewSitesListCmd(parentOpts *SitesOptions) *cobra.Command {
	opts := &SitesListOptions{}

	cmd := &cobra.Command{
		Use:     "list [<site-id|site-name>]...",
		Aliases: []string{"ls"},
		Short:   docs.SitesListDocs.Short,
		Long:    docs.SitesListDocs.Long,
		Example: docs.SitesListDocs.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.SitesAPI = parentOpts.SitesAPI
			opts.Out = parentOpts.Out
			opts.CiMode = parentOpts.CiMode
			opts.NoInteractive = parentOpts.NoInteractive
			if opts.SitesAPI == nil {
				return fmt.Errorf("internal error: no sites API available")
			}

			if len(args) > 0 {
				for _, a := range args {
					if util.IsUUID(a) {
						opts.ids = append(opts.ids, a)
						continue
					}
					opts.siteNames = append(opts.siteNames, a)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := util.BaseAuthContext(opts.SitesAPI.Token)
			sites, err := opts.SitesAPI.ListSites(ctx)
			if err != nil {
				return err
			}
			if len(sites) <= 0 {
				fmt.Fprintln(opts.Out, "No sites configured in the collective")
				return nil
			}

			// Filter on arguments (ids and site-names)
			if len(opts.ids) > 0 || len(opts.siteNames) > 0 {
				sites = util.Filter(sites, func(s openapi.SiteWithStatus) bool {
					return util.InSlice(s.GetId(), opts.ids) || util.InSliceFunc(s.GetName(), opts.siteNames, func(predicate string, siteName string) bool {
						return strings.Contains(siteName, predicate)
					})
				})
			}
			if len(sites) <= 0 {
				fmt.Fprintln(opts.Out, "No sites available matching the arguments")
				return nil
			}

			if opts.json {
				o, err := json.MarshalIndent(sites, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.Out, string(o))
			} else {
				p := util.NewPrinter(opts.Out, 4)
				p.AddHeader("Site Name", "Short Name", "ID", "Tags", "Description", "Status")
				for _, s := range sites {

					p.AddLine(
						util.StringAbbreviate(s.GetName()),
						util.StringAbbreviate(s.GetShortName()),
						util.StringAbbreviate(s.GetId()),
						s.GetTags(),
						util.StringAbbreviate(s.GetDescription()),
						util.StringAbbreviate(s.GetStatus()),
					)
				}
				p.Print()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.json, "json", false, "")

	return cmd
}
