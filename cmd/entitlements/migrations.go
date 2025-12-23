package entitlements

import (
	"fmt"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type NamesMigrationOptions struct {
	EntitlementOptions
	dryRun		bool
}

type SitesAPI struct {
	API   *openapi.EntitlementsApiService
	Token string
}

func NewCloudMigrationsCmd(parentOpts *EntitlementOptions) *cobra.Command {
	opts := &NamesMigrationOptions{}
	cmd := &cobra.Command{
		Use:     "names-migration",
		Short:   docs.SitesResourcesDocsList.Short,
		Long:    docs.SitesResourcesDocsList.Long,
		Example: docs.SitesResourcesDocsList.ExampleString(),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.EntitlementsAPI = parentOpts.EntitlementsAPI
			opts.Out = parentOpts.Out
			opts.CiMode = parentOpts.CiMode
			opts.NoInteractive = parentOpts.NoInteractive
			if opts.EntitlementsAPI == nil {
				return fmt.Errorf("internal error: no entitlements API available")
			}
			

			ctx := util.BaseAuthContext(opts.EntitlementsAPI.Token)
			result, err := opts.EntitlementsAPI.NamesMigration(ctx)
			if err != nil {
				return fmt.Errorf("names migration failed: %w", err)
			}
			resultVal := *result
			if opts.dryRun{
				fmt.Println("Performing dry run")
			}
			
			p := util.NewPrinter(opts.Out, 4)
			p.AddHeader("Name", "ID", "Original Value", "Updated Value")
			for _, d := range resultVal.Data {

					updatedHost:= ""	

					if d.UpdatedHost != nil{
							updatedHost = *d.UpdatedHost
						}	


					p.AddLine(
						util.StringAbbreviate(*d.EntitlementName),
						util.StringAbbreviate(*d.EntitlementId),
						util.StringAbbreviate(*d.OriginalHost),
						util.StringAbbreviate(updatedHost),
					)

				}
			if len(resultVal.Data) == 0 {
				p.AddLine("Nothing to migrate")
				return nil
			}

			p.Print()
			
			// for index, value := range resultVal.Data{
			// 	fmt.Println(index)
			// 	fmt.Println(*value.EntitlementName)
			// }
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dryRun", true, "")

	return cmd
}
