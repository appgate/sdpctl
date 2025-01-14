package device

import (
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewDeviceListCmd(opts *DeviceOptions) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   docs.DeviceListDoc.Short,
		Long:    docs.DeviceListDoc.Long,
		Example: docs.DeviceListDoc.ExampleString(),
		Aliases: []string{"ls"},
		Args: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.orderBy, err = cmd.Flags().GetStringSlice("order-by")
			if err != nil {
				return err
			}
			opts.descending, err = cmd.Flags().GetBool("descending")
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return deviceListRun(opts)
		},
	}

	listCmd.Flags().StringSlice("order-by", []string{"distinguished-name"}, "Order devices list by keyword. Available keywords are 'distinguished-name', 'hostname', 'username', 'provider-name', 'device-id' and 'username'")
	listCmd.Flags().Bool("descending", false, "Reverses the order of the device list")

	return listCmd
}

func deviceListRun(opts *DeviceOptions) error {
	t, err := opts.Device(opts.Config)
	if err != nil {
		return err
	}
	ctx := util.BaseAuthContext(t.Token)

	distinguishedNames, err := t.ListDistinguishedNames(ctx, opts.orderBy, opts.descending)
	if err != nil {
		return err
	}

	if opts.useJSON {
		return util.PrintJSON(opts.Out, distinguishedNames)
	}

	p := util.NewPrinter(opts.Out, 4)
	p.AddHeader(
		"Distinguished Name",
		"Device ID",
		"Username",
		"Provider Name",
		"Device Type",
		"Hostname",
		"Onboarded At",
		"Last Seen At",
	)
	for _, t := range distinguishedNames {
		p.AddLine(
			t.GetDistinguishedName(),
			t.GetDeviceId(),
			t.GetUsername(),
			t.GetProviderName(),
			t.GetDeviceType(),
			t.GetHostname(),
			t.GetOnBoardedAt(),
			t.GetLastSeenAt(),
		)
	}
	p.Print()
	return nil
}
