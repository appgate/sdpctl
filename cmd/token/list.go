package token

import (
	"context"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewTokenListCmd(opts *TokenOptions) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   docs.TokenListDoc.Short,
		Long:    docs.TokenListDoc.Long,
		Example: docs.TokenListDoc.ExampleString(),
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
			return tokenListRun(opts)
		},
	}

	listCmd.Flags().StringSlice("order-by", []string{"distinguished-name"}, "Order tokens list by keyword. Available keywords are 'distinguished-name', 'hostname', 'username', 'provider-name', 'device-id' and 'username'")
	listCmd.Flags().Bool("descending", false, "Reverses the order of the token list")

	return listCmd
}

func tokenListRun(opts *TokenOptions) error {
	ctx := context.Background()
	t, err := opts.Token(opts.Config)
	if err != nil {
		return err
	}

	distinguishedNames, err := t.ListDistinguishedNames(ctx, opts.orderBy, opts.descending)
	if err != nil {
		return err
	}

	if opts.useJSON {
		return util.PrintJSON(opts.Out, distinguishedNames)
	}

	p := util.NewPrinter(opts.Out, 4)
	p.AddHeader("Distinguished Name", "Device ID", "Last Token Issued At", "Provider Name", "Username")
	for _, dn := range distinguishedNames {
		p.AddLine(dn.GetDistinguishedName(), dn.GetDeviceId(), dn.GetLastTokenIssuedAt(), dn.GetProviderName(), dn.GetUsername())
	}
	p.Print()
	return nil
}
