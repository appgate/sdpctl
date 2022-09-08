package maintenance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

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

var promptAskTrueFalse = func(canPrompt bool) (bool, error) {
	if !canPrompt {
		return false, errors.New("prompt not supported when --no-interactive is enabled")
	}
	options := []string{"true", "false"}
	qs := &survey.Select{
		Message: "Toggle maintenance mode to:",
		Options: options,
	}
	i := 0
	if err := prompt.SurveyAskOne(qs, &i, survey.WithValidator(survey.Required)); err != nil {
		return false, err
	}
	enabled, err := strconv.ParseBool(options[i])
	if err != nil {
		return false, errors.New("could not parse first argument, expected true|false")
	}
	return enabled, nil
}

// NewToggleCmd return a new maintenance toggle command
func NewToggleCmd(f *factory.Factory) *cobra.Command {
	opts := toggleOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}

	var cmd = &cobra.Command{
		Use:     "toggle <applianceUUID> <true|false>",
		Short:   docs.MaintenanceToggle.Short,
		Long:    docs.MaintenanceToggle.Long,
		Example: docs.MaintenanceToggle.ExampleString(),
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
					return errors.New("provide 2 arguments when using --no-interactive, sdpctl appliance maintenance-toggle controllerUUID true|false")
				}
				applianceID, err := appliancepkg.PromptSelect(ctx, a, filter(primaryControllerHostname))
				if err != nil {
					return err
				}
				opts.controllerID = applianceID
				enabled, err := promptAskTrueFalse(!opts.noInteractive)
				if err != nil {
					return err
				}
				opts.enabled = enabled
			case 1:
				if util.IsUUID(args[0]) {
					opts.controllerID = args[0]
					enabled, err := promptAskTrueFalse(!opts.noInteractive)
					if err != nil {
						return fmt.Errorf("provide second argument (true|false), %s", err)
					}
					opts.enabled = enabled
					return nil
				}
				return errors.New("expected first argument to be appliance UUID")
			case 2:
				if !util.IsUUID(args[0]) {
					return fmt.Errorf("%s is not a valid appliance UUID", args[0])
				}
				opts.controllerID = args[0]
				enabled, err := strconv.ParseBool(args[1])
				if err != nil {
					return errors.New("could not parse first argument, expected true|false")
				}
				opts.enabled = enabled
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
