package serviceusers

import (
	"fmt"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewServiceUsersGetCMD(f *factory.Factory) *cobra.Command {
	token, _ := f.Config.GetBearTokenHeaderValue()
	opts := ServiceUsersOptions{
		Token:  token,
		Config: f.Config,
		API:    f.ServiceUsers,
		Out:    f.IOOutWriter,
	}

	cmd := &cobra.Command{
		Use:     "get [id]",
		Short:   docs.ServiceUsersGet.Short,
		Long:    docs.ServiceUsersGet.Long,
		Example: docs.ServiceUsersGet.ExampleString(),
		Args:    cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !util.IsUUID(args[0]) {
				return fmt.Errorf("%s: %s", InvalidUUIDError, args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := util.BaseAuthContext(opts.Token)
			api, err := opts.API(opts.Config)
			if err != nil {
				return err
			}

			id := args[0]
			user, err := api.Read(ctx, id)
			if err != nil {
				return err
			}

			return util.PrintJSON(opts.Out, user)
		},
	}

	return cmd
}
