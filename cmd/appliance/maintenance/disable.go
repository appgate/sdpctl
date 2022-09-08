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

// NewDisableCmd return a new maintenance disable command
func NewDisableCmd(f *factory.Factory) *cobra.Command {
	opts := toggleOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		enabled:   false,
	}

	var cmd = &cobra.Command{
		Use:     "disable <applianceUUID>",
		Short:   docs.MaintenanceDisable.Short,
		Long:    docs.MaintenanceDisable.Long,
		Example: docs.MaintenanceDisable.ExampleString(),
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
			filter := map[string]map[string]string{
				"include": {
					"function": "controller",
				},
			}
			switch len(args) {
			case 0:
				if opts.noInteractive {
					return errors.New("provide controller UUID when using --no-interactive, sdpctl appliance maintenance disable controllerUUID")
				}
				applianceID, err := appliancepkg.PromptSelect(ctx, a, filter)
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
