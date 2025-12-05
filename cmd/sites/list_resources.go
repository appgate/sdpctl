package sites

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/sirupsen/logrus"
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
			if len(args) == 0 {
				return fmt.Errorf("A site id must be specified")
			}
			opts.siteID = args[0]
			ctx := util.BaseAuthContext(opts.SitesAPI.Token)
			sites, err := opts.SitesAPI.ListSites(ctx)
			if sites == nil || err != nil {
				return fmt.Errorf("no sites available")

			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := util.BaseAuthContext(opts.SitesAPI.Token)

			log := logrus.WithFields(logrus.Fields{
				"id": "resource query",
			})

			resolverTypes := openapi.AllowedResolverTypeEnumValues
			resourceTypes := openapi.AllowedResourceTypeEnumValues
			resourceReturnList := []openapi.ResolverResources{}

			fmt.Println("Querying resource names...")
			resolverString, _ := cmd.Flags().GetString("resolver")
			resolvers := strings.Split(resolverString, "&")

			if resolverString != "" {
				filteredResolvers := []openapi.ResolverType{}
				for v := range resolvers {
					r, err := openapi.NewResolverTypeFromValue(resolvers[v])
					if err != nil {
						return err
					}
					filteredResolvers = append(filteredResolvers, *r)
				}
				if len(filteredResolvers) > 0 {
					resolverTypes = filteredResolvers
				}
			}

			resourceString, _ := cmd.Flags().GetString("resource")
			resource := strings.Split(resourceString, "&")

			if resourceString != "" {

				filteredResources := []openapi.ResourceType{}
				for v := range resolvers {
					r, err := openapi.NewResourceTypeFromValue(resource[v])
					if err != nil {
						return err
					}
					filteredResources = append(filteredResources, *r)
				}
				if len(filteredResources) > 0 {
					resourceTypes = filteredResources
				}
			}
			for resolveType := range resolverTypes {
				for resourceType := range resourceTypes {
					resources, err := opts.SitesAPI.ListResources(ctx, opts.siteID, &resolverTypes[resolveType], &resourceTypes[resourceType])
					if resources != nil {
						resourceReturnList = append(resourceReturnList, *resources)
					}
					if err != nil {
						log.Debug(err)
					}
				}
			}

			if opts.json {
				o, err := json.MarshalIndent(resourceReturnList, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.Out, string(o))
			} else {
				p := util.NewPrinter(opts.Out, 4)
				p.AddHeader("Name", "Resolver", "Type", "Gateway Name")
				for _, s := range resourceReturnList {
					for _, d := range s.Data {

						p.AddLine(
							util.StringAbbreviate(string(d)),
							util.StringAbbreviate(string(*s.Resolver)),
							util.StringAbbreviate(string(*s.Type)),
							util.StringAbbreviate(string(*s.GatewayName)),
						)

					}
				}
				if len(resourceReturnList) == 0 {
					p.AddLine("No resources found in the site")
					return nil
				}

				p.Print()
			}
			return nil
		},
	}

	pFlags := cmd.PersistentFlags()
	pFlags.String("resolver", "", "Specify resolver types. Use & to append multiple.")
	pFlags.String("resource", "", "Specify resource types. Use & to append multiple.")
	return cmd
}
