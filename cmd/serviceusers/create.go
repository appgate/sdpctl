package serviceusers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

func NewServiceUsersCreateCMD(f *factory.Factory) *cobra.Command {
	token, _ := f.Config.GetBearTokenHeaderValue()
	opts := ServiceUsersOptions{
		Token:  token,
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
			return serviceUserCreateRun(cmd, args, opts)
		},
	}

	flags := cmd.Flags()
	flags.String("name", "", "name for service user")
	flags.StringSlice("tags", []string{}, "tags for service user")
	flags.StringP("from-file", "f", "", "create a user from a valid json file")

	return cmd
}

func serviceUserCreateRun(cmd *cobra.Command, args []string, opts ServiceUsersOptions) error {
	api, err := opts.API(opts.Config)
	if err != nil {
		return err
	}

	ctx := util.BaseAuthContext(api.Token)

	fromFile, err := cmd.Flags().GetString("from-file")
	if err != nil {
		return err
	}

	users := []openapi.ServiceUsersGetRequest{}
	if len(fromFile) > 0 {
		path := filesystem.AbsolutePath(fromFile)
		ok, err := util.FileExists(path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("File not found: %s", path)
		}
		file, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Try unmarshal json array first
		dtoArray := ServiceUserArrayDTO{}
		if err := json.Unmarshal(file, &dtoArray); err != nil {
			// if the file is a single user object, unmarshal that instead
			singleDTO := ServiceUserDTO{}
			if err := json.Unmarshal(file, &singleDTO); err == nil {
				dtoArray = append(dtoArray, singleDTO)
			}
		}
		for i := 0; i < len(dtoArray); i++ {
			u := openapi.ServiceUsersGetRequest{
				Name:     dtoArray[i].Name,
				Password: dtoArray[i].Password,
				Disabled: openapi.PtrBool(dtoArray[i].Disabled),
				Labels:   &dtoArray[i].Labels,
				Notes:    openapi.PtrString(dtoArray[i].Notes),
				Tags:     dtoArray[i].Tags,
			}
			users = append(users, u)
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
				return fmt.Errorf("Failed to read from stdin: %w", err)
			}
			password = strings.TrimSuffix(string(buf), "\n")
		}

		noInteractive, err := cmd.Flags().GetBool("no-interactive")
		if err != nil {
			return err
		}
		if !noInteractive {
			if len(username) <= 0 {
				username, err = prompt.PromptInput("Name for service user:")
				if err != nil {
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
			errs = multierror.Append(errs, errors.New("name is required"))
		}
		if len(password) <= 0 {
			errs = multierror.Append(errs, errors.New("passphrase is required"))
		}
		if errs != nil {
			errs = multierror.Append(errs, errors.New("failed to create user: missing data"))
			return errs.ErrorOrNil()
		}

		users = append(users, openapi.ServiceUsersGetRequest{
			Name:     username,
			Password: password,
		})
	}

	if len(users) <= 0 {
		return fmt.Errorf("Failed to create user(s): no user data provided")
	}

	var errs *multierror.Error
	for _, u := range users {
		created, err := api.Create(ctx, u)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		util.PrintJSON(opts.Out, created)
	}
	return errs.ErrorOrNil()
}
