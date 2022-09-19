package serviceusers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

func NewServiceUsersCreateCMD(f *factory.Factory) *cobra.Command {
	opts := ServiceUsersOptions{
		Config: f.Config,
		API:    f.ServiceUsers,
		Out:    f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   docs.ServiceUsersCreate.Short,
		Long:    docs.ServiceUsersCreate.Long,
		Example: docs.ServiceUsersCreate.ExampleString(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			api, err := opts.API(opts.Config)
			if err != nil {
				return err
			}

			fromFile, err := cmd.Flags().GetString("from-file")
			if err != nil {
				return err
			}

			users := []openapi.ServiceUser{}
			if len(fromFile) > 0 {
				path := filesystem.AbsolutePath(fromFile)
				ok, err := util.FileExists(path)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("file not found: %s", path)
				}
				file, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				dto := ServiceUserArrayDTO{}
				if err := json.Unmarshal(file, &dto); err != nil {
					return err
				}
				for i := 0; i < len(dto); i++ {
					users = append(users, openapi.ServiceUser{
						Name:     dto[i].Name,
						Password: dto[i].Password,
						Disabled: &dto[i].Disabled,
						Labels:   &dto[i].Labels,
						Notes:    &dto[i].Notes,
						Tags:     dto[i].Tags,
					})
				}
			} else {
				noInteractive, err := cmd.Flags().GetBool("no-interactive")
				if err != nil {
					return err
				}

				if !noInteractive {
					single := openapi.ServiceUser{}
					username, err := cmd.Flags().GetString("name")
					if err != nil {
						return err
					}
					password, err := cmd.Flags().GetString("passphrase")
					if err != nil {
						return err
					}
					if len(username) <= 0 {
						qs := &survey.Input{
							Message: "Name for service user:",
						}
						if err := prompt.SurveyAskOne(qs, &username); err != nil {
							return err
						}
					}
					if len(password) <= 0 {
						password, err = prompt.PasswordConfirmation("Passphrase for service user:")
						if err != nil {
							return err
						}
					}

					single.SetName(username)
					single.SetPassword(password)
					users = append(users, single)
				}
			}

			if len(users) <= 0 {
				return fmt.Errorf("failed to create user(s): no user data provided")
			}

			var errs *multierror.Error
			for _, u := range users {
				created, err := api.Create(ctx, u)
				if err != nil {
					errs = multierror.Append(err, errs)
					continue
				}
				fmt.Fprint(opts.Out, "New service user created:\n")
				util.PrintJSON(opts.Out, created)
			}
			return errs.ErrorOrNil()
		},
	}

	flags := cmd.Flags()
	flags.String("name", "", "name for service user")
	flags.String("passphrase", "", "passphrase for service user")
	flags.StringP("from-file", "f", "", "create a user from a valid json file")

	return cmd
}
