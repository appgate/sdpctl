package upgrade

import (
	"context"
	"fmt"
	"io"

	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/cmdappliance"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type switchPartitionOpts struct {
	cmdappliance.AppliancCmdOpts
	Out  io.Writer
	json bool
}

// NewSwitchPartitionCmd return a new appliance switch-partition command
func NewSwitchPartitionCmd(f *factory.Factory) *cobra.Command {
	aopts := cmdappliance.AppliancCmdOpts{
		Appliance: f.Appliance,
		Config:    f.Config,
		CanPrompt: f.CanPrompt(),
	}
	opts := switchPartitionOpts{
		aopts,
		f.IOOutWriter,
		false,
	}

	cmd := &cobra.Command{
		Use:     "switch-partition [<appliance-id>]",
		Short:   docs.ApplianceUpgradeSwitchPartitionDoc.Short,
		Long:    docs.ApplianceUpgradeSwitchPartitionDoc.Long,
		Example: docs.ApplianceUpgradeSwitchPartitionDoc.ExampleString(),
		Args: func(cmd *cobra.Command, args []string) error {
			return cmdappliance.ArgsSelectAppliance(cmd, args, &opts.AppliancCmdOpts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			return switchPartitionRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")

	return cmd
}

func switchPartitionRun(cmd *cobra.Command, args []string, opts *switchPartitionOpts) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	t, err := cfg.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ctx := context.Background()
	appliance, err := a.Get(ctx, opts.ApplianceID)
	if err != nil {
		return err
	}

	if !opts.NoInteractive {
		confirmation := fmt.Sprintf("Are you really sure you want to switch-partition on %s?", appliance.GetName())
		if err := prompt.AskConfirmation(confirmation); err != nil {
			return err
		}
	}

	changeID, err := a.UpgradeSwitchPartition(ctx, opts.ApplianceID)
	if err != nil {
		return err
	}
	ac := change.ApplianceChange{
		APIClient: a.APIClient,
		Token:     t,
	}
	change, err := ac.RetryUntilCompleted(ctx, changeID, opts.ApplianceID)
	if err != nil {
		return err
	}
	if opts.json {
		return util.PrintJSON(opts.Out, change)
	}
	fmt.Fprintf(opts.Out, "Change result: %s \nChange Status: %s\n", change.GetResult(), change.GetStatus())
	return nil
}
