package token

import (
	"io"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/token"
	"github.com/spf13/cobra"
)

type TokenOptions struct {
	Config     *configuration.Config
	Out        io.Writer
	Token      func(c *configuration.Config) (*token.Token, error)
	Debug      bool
	useJSON    bool
	orderBy    []string
	descending bool
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
		Short: "Perform actions on Admin, Claims and Entitlement tokens",
		Long:  `The token command allows you to renew or revoke tokens used in the Collective.`,
	}

	tokenCmd.PersistentFlags().BoolVar(&opts.useJSON, "json", false, "Display in JSON format")

	tokenCmd.AddCommand(NewTokenRevokeCmd(opts))
	tokenCmd.AddCommand(NewTokenListCmd(opts))

	return tokenCmd
}
