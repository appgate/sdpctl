package token

import (
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/token"
	"github.com/spf13/cobra"
	"io"
)

type TokenOptions struct {
	Config  *configuration.Config
	Out     io.Writer
	Token   func(c *configuration.Config) (*token.Token, error)
	Debug   bool
	useJSON bool
}

func NewTokenCmd(f *factory.Factory) *cobra.Command {
	opts := &TokenOptions{
		Config: f.Config,
		Out:    f.IOOutWriter,
		Token:  f.Token,
		Debug:  f.Config.Debug,
	}

	var tokenCmd = &cobra.Command{
		Use:   "token",
		Short: "Perform actions related to token on the Appgate SDP Collective",
	}

	tokenCmd.PersistentFlags().BoolVar(&opts.useJSON, "json", false, "Display in JSON format")

	tokenCmd.AddCommand(NewTokenRevokeCmd(opts))
	tokenCmd.AddCommand(NewTokenListCmd(opts))

	return tokenCmd
}
