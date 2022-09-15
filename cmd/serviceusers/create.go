package serviceusers

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
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

			u := openapi.ServiceUser{}

			username, err := cmd.Flags().GetString("username")
			if err != nil {
				return err
			}
			password, err := cmd.Flags().GetString("password")
			if err != nil {
				return err
			}

			if len(username) <= 0 {
				qs := &survey.Input{
					Message: "username for service user",
				}
				if err := prompt.SurveyAskOne(qs, &username); err != nil {
					return err
				}
			}
			if len(password) <= 0 {
				qs := &survey.Password{
					Message: "password for service user",
				}
				if err := prompt.SurveyAskOne(qs, &password); err != nil {
					return err
				}
			}

			u.SetName(username)
			u.SetPassword(password)

			created, err := api.Create(ctx, u)
			if err != nil {
				return err
			}

			fmt.Fprint(opts.Out, "New service user created:\n")
			return util.PrintJSON(opts.Out, created)
		},
	}

	cmd.Flags().String("username", "", "username for service user")
	cmd.Flags().String("password", "", "password for service user")

	return cmd
}
