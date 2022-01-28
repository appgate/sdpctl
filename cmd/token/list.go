package token

import (
	"context"

	"github.com/appgate/appgatectl/pkg/util"
	"github.com/spf13/cobra"
)

func NewTokenListCmd(opts *TokenOptions) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "list distinguished names of active devices",
		Long:  `List distinguished names of active tokens, either in table format or JSON format using the '--json' flag`,
		Example: `$ appgatectl token list
$ appgatectl token list --json`,
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

	p := util.NewPrinter(opts.Out)
	p.AddHeader("Distinguished Name", "Device ID", "Last Token Issued At", "Provider Name", "Username")
	for _, dn := range distinguishedNames {
		p.AddLine(dn.GetDistinguishedName(), dn.GetDeviceId(), dn.GetLastTokenIssuedAt(), dn.GetProviderName(), dn.GetUsername())
	}
	p.Print()
	return nil
}
