package cmdappliance

import (
	"context"
	"errors"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type AppliancCmdOpts struct {
	ApplianceID   string
	CanPrompt     bool
	NoInteractive bool
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Config        *configuration.Config
	Filter        map[string]map[string]string
}

// ArgsSelectAppliance allow a command to select appliance id interactively or use it as first Argument
func ArgsSelectAppliance(cmd *cobra.Command, args []string, opts *AppliancCmdOpts) error {
	noInteractive, err := cmd.Flags().GetBool("no-interactive")
	if err != nil {
		return err
	}
	opts.NoInteractive = noInteractive
	if !opts.CanPrompt {
		opts.NoInteractive = true
	}
	orderBy, err := cmd.Flags().GetStringSlice("order-by")
	if err != nil {
		return err
	}
	descending, err := cmd.Flags().GetBool("descending")
	if err != nil {
		return err
	}
	switch len(args) {
	case 0:
		if opts.NoInteractive {
			return errors.New("Can't prompt, applianceID argument required")
		}

		a, err := opts.Appliance(opts.Config)
		if err != nil {
			return err
		}
		applianceID, err := appliancepkg.PromptSelect(context.Background(), a, opts.Filter, orderBy, descending)
		if err != nil {
			return err
		}
		opts.ApplianceID = applianceID
	case 1:
		if !util.IsUUID(args[0]) {
			return errors.New("Expected argument to be appliance uuid")
		}
		opts.ApplianceID = args[0]
	}
	return nil
}
