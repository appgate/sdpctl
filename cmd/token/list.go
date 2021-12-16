package token

import (
	"context"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/token"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/spf13/cobra"
	"io"
)

type tokenListOptions struct {
	Config *configuration.Config
	Out    io.Writer
	Token  func(c *configuration.Config) (*token.Token, error)
	debug  bool
	json   bool
}

func NewTokenListCmd(f *factory.Factory) *cobra.Command {
	opts := tokenListOptions{
		Config: f.Config,
		Out:    f.IOOutWriter,
		Token:  f.Token,
		debug:  f.Config.Debug,
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "list distinguished names of active devices",
		RunE: func(c *cobra.Command, args []string) error {
			return tokenListRun(c, args, &opts)
		},
	}
	listCmd.PersistentFlags().BoolVar(&opts.json, "json", false, "Display in JSON format")

	return listCmd
}

func tokenListRun(c *cobra.Command, args []string, opts *tokenListOptions) error {
	ctx := context.Background()
	t, err := opts.Token(opts.Config)
	if err != nil {
		return err
	}

	distinguishedNames, err := t.ListDistinguishedNames(ctx)
	if err != nil {
		return err
	}

	if opts.json {
		err = util.PrintJson(opts.Out, distinguishedNames)
		if err != nil {
			return err
		}
	}

	p := util.NewPrinter(opts.Out)
	p.AddHeader("Distinguished Name", "Device ID", "Last Token Issued At", "Provider Name", "Username")
	for _, dn := range distinguishedNames {
		p.AddLine(dn.GetDistinguishedName(), dn.GetDeviceId(), dn.GetLastTokenIssuedAt(), dn.GetProviderName(), dn.GetUsername())
	}
	p.Print()
	return nil
}
