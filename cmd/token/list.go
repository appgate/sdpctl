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
		RunE: func(c *cobra.Command, args []string) error {
			return tokenListRun(opts)
		},
	}

	return listCmd
}

func tokenListRun(opts *TokenOptions) error {
	ctx := context.Background()
	t, err := opts.Token(opts.Config)
	if err != nil {
		return err
	}

	distinguishedNames, err := t.ListDistinguishedNames(ctx)
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
