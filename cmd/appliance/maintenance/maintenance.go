package maintenance

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
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
	controllerIDs []string
	enabled       bool
	noInteractive bool
}

func toggleArgs(cmd *cobra.Command, opts *toggleOptions, args []string) error {
	noInteractive, err := cmd.Flags().GetBool("no-interactive")
	if err != nil {
		return err
	}
	opts.noInteractive = noInteractive
	orderBy, err := cmd.Flags().GetStringSlice("order-by")
	if err != nil {
		return err
	}
	descending, err := cmd.Flags().GetBool("descending")
	if err != nil {
		return err
	}
	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	ctx := util.BaseAuthContext(a.Token)
	primaryControllerHostname, err := opts.Config.GetHost()
	if err != nil {
		return err
	}
	switch len(args) {
	case 0:
		if opts.noInteractive {
			return errors.New("Provide the Controller UUID when using --no-interactive, sdpctl appliance maintenance disable controllerUUID")
		}
		applianceIDs, err := appliancepkg.PromptMultiSelect(ctx, a, filter(primaryControllerHostname), orderBy, descending)
		if err != nil {
			return err
		}
		opts.controllerIDs = applianceIDs
	case 1:
		if util.IsUUID(args[0]) {
			opts.controllerIDs = []string{args[0]}
			return nil
		}
		return errors.New("Expected first argument to be appliance UUID")
	}

	return nil
}

func toggleRun(cmd *cobra.Command, args []string, opts *toggleOptions) error {
	cfg := opts.Config
	if cfg.Version < 15 {
		return errors.New("Maintenance mode is not supported on this version")
	}
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	t, err := cfg.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ctx := util.BaseAuthContext(a.Token)

	if !opts.noInteractive {
		fmt.Fprintf(opts.Out, "\n%s\n\n", `A Controller in maintenance mode will not accept any API calls besides disabling maintenance mode. Starting in version 6.0, clients will still function as usual while a Controller is in maintenance mode.
This is a superuser function and should only be used if you know what you are doing.`)
		applianceNames := []string{}
		for _, controllerID := range opts.controllerIDs {
			appliance, err := a.Get(ctx, controllerID)
			if err != nil {
				return err
			}
			applianceNames = append(applianceNames, appliance.GetName())
		}
		confirmation := fmt.Sprintf("Are you sure you want to disable maintenance mode on %s?", strings.Join(applianceNames, ", "))
		if opts.enabled {
			confirmation = fmt.Sprintf("Are you sure you want to enable maintenance mode on %s?", strings.Join(applianceNames, ", "))
		}
		if err := prompt.AskConfirmation(confirmation); err != nil {
			return err
		}
	}
	var errs *multierror.Error
	changes := []openapi.AppliancesIdChangeChangeIdGet200Response{}
	for _, controllerID := range opts.controllerIDs {
		changeID, err := a.UpdateMaintenanceMode(ctx, controllerID, opts.enabled)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		ac := change.ApplianceChange{
			APIClient: a.APIClient,
			Token:     t,
		}
		change, err := ac.RetryUntilCompleted(ctx, changeID, controllerID)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		if opts.json {
			changes = append(changes, *change)
		} else {
			fmt.Fprintf(opts.Out, "Change result: %s \nChange Status: %s\n", change.GetResult(), change.GetStatus())
		}
	}
	if opts.json {
		err = util.PrintJSON(opts.Out, changes)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}
