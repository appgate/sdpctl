package sites

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type ResourcesListOptions struct {
	SitesOptions
	siteID string
	json   bool
}

func NewResourceNamesCmd(parentOpts *SitesOptions) *cobra.Command {
	opts := &ResourcesListOptions{}

	cmd := &cobra.Command{
		Use:     "resources <site-id>",
		Aliases: []string{"ls"},
		Short:   docs.SitesResourcesDocsList.Short,
		Long:    docs.SitesResourcesDocsList.Long,
		Example: docs.SitesResourcesDocsList.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.SitesAPI = parentOpts.SitesAPI
			opts.Out = parentOpts.Out
			opts.CiMode = parentOpts.CiMode
			opts.NoInteractive = parentOpts.NoInteractive
			if opts.SitesAPI == nil {
				return fmt.Errorf("internal error: no sites API available")
			}

			opts.siteID = args[0]

			sites, err := opts.SitesAPI.ListSites(context.Background())
			if sites == nil || err != nil{
				return fmt.Errorf("no sites available")

			}

			// if len(args) > 0 {
			// 	for _, a := range args {
			// 		if util.IsUUID(a) {
			// 			opts.siteId = append(opts.ids, a)
			// 			continue
			// 		}
			// 		opts.siteNames = append(opts.siteNames, a)
			// 	}
			// }



			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			




			
			
			//resolvers = AllowedResolverTypeEnumValues
			resolverTypes := openapi.AllowedResolverTypeEnumValues
			resourceTypes := openapi.AllowedResourceTypeEnumValues
			resource_return_list := []openapi.ResolverResources{}
			
			for resolveType := range resolverTypes{
				for resourceType := range resourceTypes{
					resources, err := opts.SitesAPI.ListResources(ctx, opts.siteID, &resolverTypes[resolveType], &resourceTypes[resourceType])
					if resources != nil {
						resource_return_list = append(resource_return_list, *resources)
					// } else {
					// 	fmt.Fprintln(opts.Out, "No resources found for" + strconv.Itoa(resourceType) + strconv.Itoa(resolveType))
					 }
					
					if err != nil {
						return err
						
					}
					 	}

			}
			
			
			if len(resource_return_list) <= 0 {
				fmt.Fprintln(opts.Out, "No resources found in the site")
				return nil
			}


			if opts.json {
				o, err := json.MarshalIndent(resource_return_list, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.Out, string(o))
			} else {
				p := util.NewPrinter(opts.Out, 4)
				p.AddHeader("Resolver", "Type", "Gateway Name", "Resource Count")
				for _, s := range resource_return_list {

					p.AddLine(
						util.StringAbbreviate(string(*s.Resolver)),
						util.StringAbbreviate(string(*s.Type)),
						util.StringAbbreviate(string(*s.GatewayName)),
						util.StringAbbreviate(string(*s.TotalCount)),
					)
					p.AddLine("Resources: ")
					for _, r := range s.Data {
						p.AddLine(r)
					}

				}
				p.Print()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.json, "json", false, "")

	return cmd
}
