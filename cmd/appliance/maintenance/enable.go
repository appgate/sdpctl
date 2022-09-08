package maintenance

import (
	"context"
	"errors"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

// NewEnableCmd return a new maintenance enable command
func NewEnableCmd(f *factory.Factory) *cobra.Command {
	opts := toggleOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		enabled:   true,
	}

	var cmd = &cobra.Command{
		Use:     "enable <applianceUUID>",
		Short:   docs.MaintenanceEnable.Short,
		Long:    docs.MaintenanceEnable.Long,
		Example: docs.MaintenanceEnable.ExampleString(),
		Args: func(cmd *cobra.Command, args []string) error {
			noInteractive, err := cmd.Flags().GetBool("no-interactive")
			if err != nil {
				return err
			}
			opts.noInteractive = noInteractive
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
			}
			ctx := context.Background()
			primaryControllerHostname, err := opts.Config.GetHost()
			if err != nil {
				return err
			}

			switch len(args) {
			case 0:
				if opts.noInteractive {
					return errors.New("provide controller UUID when using --no-interactive, sdpctl appliance maintenance enable controllerUUID")
				}
				applianceID, err := appliancepkg.PromptSelect(ctx, a, filter(primaryControllerHostname))
				if err != nil {
					return err
				}
				opts.controllerID = applianceID
			case 1:
				if util.IsUUID(args[0]) {
					opts.controllerID = args[0]
					return nil
				}
				return errors.New("expected first argument to be appliance UUID")
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return toggleRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return cmd
}
