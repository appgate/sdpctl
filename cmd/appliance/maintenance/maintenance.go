package maintenance

import (
	"context"
	"errors"
	"fmt"
	"io"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func filter(hostname string) map[string]map[string]string {
	return map[string]map[string]string{
		"include": {
			"function": "controller",
		},
		"exclude": {
			"hostname": hostname,
		},
	}
}

// NewMaintenanceCmd return a new subcommand for maintenance
func NewMaintenanceCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "maintenance",
		TraverseChildren: true,
		Short:            docs.MaintenanceRootDoc.Short,
		Long:             docs.MaintenanceRootDoc.Long,
	}

	cmd.AddCommand(NewEnableCmd(f))
	cmd.AddCommand(NewDisableCmd(f))
	cmd.AddCommand(NewStatusCmd(f))

	return cmd
}

type toggleOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug         bool
	json          bool
	controllerID  string
	enabled       bool
	noInteractive bool
}

func toggleArgs(cmd *cobra.Command, opts *toggleOptions, args []string) error {
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
			return errors.New("provide controller UUID when using --no-interactive, sdpctl appliance maintenance disable controllerUUID")
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
}

func toggleRun(cmd *cobra.Command, args []string, opts *toggleOptions) error {
	cfg := opts.Config
	if cfg.Version < 15 {
		return errors.New("maintenance mode is not supported on this version.")
	}
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	t, err := cfg.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ctx := context.Background()

	if !opts.noInteractive {
		fmt.Fprintf(opts.Out, "\n%s\n\n", `A Controller in maintenance mode will not accept any API calls besides disabling maintenance mode. Starting in version 6.0, clients will still function as usual while a Controller is in maintenance mode.
This is a superuser function and should only be used if you know what you are doing.`)
		appliance, err := a.Get(ctx, opts.controllerID)
		if err != nil {
			return err
		}

		confirmation := fmt.Sprintf("Are you really sure you want to disable maintenance mode on %s?", appliance.GetName())
		if opts.enabled {
			confirmation = fmt.Sprintf("Are you really sure you want to enable maintenance mode on %s?", appliance.GetName())
		}
		if err := prompt.AskConfirmation(confirmation); err != nil {
			return err
		}
	}

	changeID, err := a.UpdateMaintenanceMode(ctx, opts.controllerID, opts.enabled)
	if err != nil {
		return err
	}
	ac := change.ApplianceChange{
		APIClient: a.APIClient,
		Token:     t,
	}
	change, err := ac.RetryUntilCompleted(ctx, changeID, opts.controllerID)
	if err != nil {
		return err
	}
	if opts.json {
		return util.PrintJSON(opts.Out, change)
	}
	fmt.Fprintf(opts.Out, "Change result: %s \nChange Status: %s\n", change.GetResult(), change.GetStatus())
	return nil
}
