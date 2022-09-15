package serviceusers

import (
	"context"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewServiceUsersListCMD(f *factory.Factory) *cobra.Command {
	opts := ServiceUsersOptions{
		Config: f.Config,
		API:    f.ServiceUsers,
		Out:    f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   docs.ServiceUsersList.Short,
		Long:    docs.ServiceUsersList.Long,
		Example: docs.ServiceUsersList.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if opts.JSON, err = cmd.Flags().GetBool("json"); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := opts.API(opts.Config)
			if err != nil {
				return err
			}
			ctx := context.Background()

			users, err := api.List(ctx)
			if err != nil {
				return err
			}

			if opts.JSON {
				return util.PrintJSON(opts.Out, users)
			}

			p := util.NewPrinter(opts.Out, 4)
			p.AddHeader("Name", "ID", "Disabled")
			for _, u := range users {
				p.AddLine(u.GetName(), u.GetId(), u.GetDisabled())
			}
			p.Print()

			return nil
		},
	}

	return cmd
}
