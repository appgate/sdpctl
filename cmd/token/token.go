package token

import (
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

func NewTokenCmd(f *factory.Factory) *cobra.Command {
	var tokenCmd = &cobra.Command{
		Use:   "token",
		Short: "Perform actions related to token on the Appgate SDP Collective",
	}

	tokenCmd.AddCommand(NewTokenRevokeCmd(f))
	tokenCmd.AddCommand(NewTokenListCmd(f))

	return tokenCmd
}
