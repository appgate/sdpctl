package serviceusers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
		In:     f.Stdin,
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
						Disabled: openapi.PtrBool(dto[i].Disabled),
						Labels:   &dto[i].Labels,
						Notes:    openapi.PtrString(dto[i].Notes),
						Tags:     dto[i].Tags,
					})
				}
			} else {
				username, err := cmd.Flags().GetString("name")
				if err != nil {
					return err
				}

				var password string
				var hasStdin bool
				stat, err := os.Stdin.Stat()
				if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
					hasStdin = true
				}
				if hasStdin {
					buf, err := io.ReadAll(opts.In)
					if err != nil {
						return fmt.Errorf("failed to read from stdin: %w", err)
					}
					password = strings.TrimSuffix(string(buf), "\n")
				} else {
					password, err = cmd.Flags().GetString("passphrase")
					if err != nil {
						return err
					}
				}

				noInteractive, err := cmd.Flags().GetBool("no-interactive")
				if err != nil {
					return err
				}
				if !noInteractive {
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

				}

				var errs *multierror.Error
				if len(username) <= 0 {
					errs = multierror.Append(errors.New("name is required"), errs)
				}
				if len(password) <= 0 {
					errs = multierror.Append(errors.New("password is required"), errs)
				}
				if errs != nil {
					return errs.ErrorOrNil()
				}

				users = append(users, openapi.ServiceUser{
					Name:     username,
					Password: password,
				})
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
