package serviceusers

import (
	"context"
	"fmt"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewServiceUsersDeleteCMD(f *factory.Factory) *cobra.Command {
	opts := ServiceUsersOptions{
		API:    f.ServiceUsers,
		Config: f.Config,
		Out:    f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "delete [id]",
		Short:   docs.ServiceUsersDelete.Short,
		Long:    docs.ServiceUsersDelete.Long,
		Example: docs.ServiceUsersDelete.ExampleString(),
		Aliases: []string{"remove", "rm", "del"},
		Args:    cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
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

			if err := api.Delete(ctx, id); err != nil {
				return err
			}

			fmt.Fprint(opts.Out, "user successfully deleted\n")
			return nil
		},
	}

	return cmd
}
