package entitlements

import (
	"fmt"
	"io"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

type NamesMigrationOptions struct {
	EntitlementOptions
	factory *factory.Factory
	dryRun bool
	json      bool
	Config            *configuration.Config
	Out               io.Writer
	Appliance         func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug             bool
	defaultFilter     map[string]map[string]string
	ciMode            bool
}

func NewCloudMigrationsCmd(parentOpts *EntitlementOptions, f *factory.Factory) *cobra.Command {
	opts := &NamesMigrationOptions{		
		Config:     f.Config,
		Appliance:  f.Appliance,
		debug:      f.Config.Debug,
		Out:        f.IOOutWriter,
		defaultFilter: map[string]map[string]string{
			"include": {},
			"exclude": {
				"active": "false",
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "names-migration",
		Short:   docs.NamesMigrationsDocsList.Short,
		Long:    docs.NamesMigrationsDocsList.Long,
		Example: docs.NamesMigrationsDocsList.ExampleString(),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.EntitlementsAPI = parentOpts.EntitlementsAPI
			opts.Out = parentOpts.Out
			opts.CiMode = parentOpts.CiMode
			opts.NoInteractive = parentOpts.NoInteractive
			if opts.EntitlementsAPI == nil {
				return fmt.Errorf("internal error: no entitlements API available")
			}

			a, _ := opts.Appliance(opts.Config)
			if versionMin(cmd, a, "6.6.1") == false{
				return fmt.Errorf("All appliances must be version 6.6.1 or greater to run the names migration");
			} 
			
			ctx := util.BaseAuthContext(opts.EntitlementsAPI.Token)
			result, err := opts.EntitlementsAPI.NamesMigration(ctx, opts.dryRun)
			
			filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), appliancepkg.DefaultCommandFilter)
			fmt.Println(a.List(ctx, filter, orderBy, descending))
			
			if err != nil {
				return fmt.Errorf("names migration failed: %w", err)
			}

			if opts.dryRun {
				fmt.Println("Performing dry run")
			}

			if result == nil {
				fmt.Println("Nothing to migrate")
				return nil
			}



			resultVal := *result

			p := util.NewPrinter(opts.Out, 4)
			p.AddHeader("Name", "ID", "Original Value", "Updated Value")
			for _, d := range resultVal.Data {

				updatedHost := ""

				if d.UpdatedHost != nil {
					updatedHost = *d.UpdatedHost
				}

				p.AddLine(
					util.StringAbbreviate(*d.EntitlementName),
					util.StringAbbreviate(*d.EntitlementId),
					util.StringAbbreviate(*d.OriginalHost),
					util.StringAbbreviate(updatedHost),
				)

			}

			p.Print()

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dryRun", true, "")

	return cmd
}


func versionMin(cmd *cobra.Command, a *appliancepkg.Appliance, minVersion string, ) bool {
	ctx := util.BaseAuthContext(a.Token)
	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), appliancepkg.DefaultCommandFilter)
	stats, _, _ := a.ApplianceStatus(ctx, filter, orderBy, descending)
	minVersionCompare, _ := version.NewVersion(minVersion)
	for _, s := range stats.GetData() {
		version := s.GetApplianceVersion()
		if v, err := appliancepkg.ParseVersionString(version); err == nil {
			if v.LessThan(minVersionCompare){
				return false;
			}
		}
	}
	return true
}
