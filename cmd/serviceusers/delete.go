package serviceusers

import (
	"context"
	"fmt"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

func NewServiceUsersDeleteCMD(f *factory.Factory) *cobra.Command {
	opts := ServiceUsersOptions{
		API:    f.ServiceUsers,
		Config: f.Config,
		Out:    f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "delete [id...]",
		Short:   docs.ServiceUsersDelete.Short,
		Long:    docs.ServiceUsersDelete.Long,
		Example: docs.ServiceUsersDelete.ExampleString(),
		Aliases: []string{"remove", "rm", "del"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var errs *multierror.Error
			for _, arg := range args {
				if !util.IsUUID(arg) {
					errs = multierror.Append(errs, fmt.Errorf("%s: %s", InvalidUUIDError, args[0]))
				}
			}
			return errs.ErrorOrNil()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			api, err := opts.API(opts.Config)
			if err != nil {
				return err
			}

			ids := []string{}
			if len(args) > 0 {
				ids = args
			} else {
				noInteractive, err := cmd.Flags().GetBool("no-interactive")
				if err != nil {
					return err
				}

				if !noInteractive {
					userList, err := api.List(ctx)
					if err != nil {
						return err
					}
					userNames := []string{}
					for _, u := range userList {
						userNames = append(userNames, u.Name)
					}
					selected, err := prompt.PromptMultiSelection("Select service users to delete:", userNames, nil)
					if err != nil {
						return err
					}
					for _, sel := range selected {
						for _, u := range userList {
							if sel == u.Name {
								ids = append(ids, u.GetId())
							}
						}
					}
				}
			}

			if len(ids) <= 0 {
				return fmt.Errorf("No service users selected for deletion")
			}

			var errs *multierror.Error
			for _, id := range ids {
				if err := api.Delete(ctx, id); err != nil {
					errs = multierror.Append(errs, err)
					continue
				}
				fmt.Fprintf(opts.Out, "deleted: %s\n", id)
			}
			return errs.ErrorOrNil()
		},
	}

	return cmd
}
