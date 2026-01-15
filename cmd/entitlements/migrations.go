package entitlements

import (
	"fmt"
	"io"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
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
			
			ctx := util.BaseAuthContext(opts.EntitlementsAPI.Token)
			result, err := opts.EntitlementsAPI.NamesMigration(ctx, opts.dryRun)
			
			a, _ := opts.Appliance(opts.Config)
			
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


// func versionMin(opts *EntitlementOptions, cmd *cobra.Command, args []string, minVersion string) error {
// 	cfg := opts.Config
// 	a, err := opts.Appliance(cfg)
// 	if err != nil {
// 		return err
// 	}
// 	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), appliancepkg.DefaultCommandFilter)
// 	ctx := util.BaseAuthContext(a.Token)
// 	stats, _, err := a.ApplianceStatus(ctx, filter, orderBy, descending)
// 	if err != nil {
// 		return err
// 	}
// 	if opts.json {
// 		j, err := json.MarshalIndent(&stats, "", "  ")
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Fprintf(opts.Out, "\n%s\n", string(j))
// 		return nil
// 	}
// 	w := util.NewPrinter(opts.Out, 4)
// 	diskHeader := "Disk"
// 	if cfg.Version >= 18 {
// 		diskHeader += " (used / total)"
// 	}
// 	w.AddHeader("Name", "Status", "Function", "CPU", "Memory", "Network out/in", diskHeader, "Version", "Sessions")
// 	for _, s := range stats.GetData() {
// 		version := s.GetApplianceVersion()
// 		if v, err := appliancepkg.ParseVersionString(version); err == nil {
// 			version = v.String()
// 		}
// 		w.AddLine(
// 			s.GetName(),
// 			s.GetStatus(),
// 			appliancepkg.ApplianceActiveFunctions(s),
// 			fmt.Sprintf("%g%%", s.GetCpu()),
// 			fmt.Sprintf("%g%%", s.GetMemory()),
// 			statsNetworkPrettyPrint(s.GetDetails().Network),
// 			statsDiskUsage(s),
// 			version,
// 			s.GetNumberOfSessions(),
// 		)
// 	}
// 	w.Print()
// 	return nil
// }
