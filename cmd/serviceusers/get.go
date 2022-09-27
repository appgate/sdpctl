package serviceusers

import (
	"context"
	"fmt"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewServiceUsersGetCMD(f *factory.Factory) *cobra.Command {
	opts := ServiceUsersOptions{
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
			if opts.Config.Version <= 16 {
				return fmt.Errorf("The service user interface is only available from API version 17 or higher. Currently using API version %d", opts.Config.Version)
			}
			if !util.IsUUID(args[0]) {
				return fmt.Errorf("%s: %s", InvalidUUIDError, args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
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
