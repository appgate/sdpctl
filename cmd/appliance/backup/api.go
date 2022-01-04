package backup

import (
	"context"
	"fmt"
	"io"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/api"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
)

type apiOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	APIClient func(c *configuration.Config) (*openapi.APIClient, error)
	debug     bool
	disable   bool
}

// NewBackupAPICmd return a new backup API command
func NewBackupAPICmd(f *factory.Factory) *cobra.Command {
	opts := apiOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:   "api",
		Short: `Whether the backup API is enabled or not.`,
		RunE: func(c *cobra.Command, args []string) error {
			return backupAPIrun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.disable, "disable", false, "Disable the backup API")

	return cmd
}

func backupAPIrun(cmd *cobra.Command, args []string, opts *apiOptions) error {
	client, err := opts.APIClient(opts.Config)
	if err != nil {
		return err
	}
	ctx := context.Background()
	t := opts.Config.GetBearTokenHeaderValue()
	settings, response, err := client.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(t).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if v, ok := settings.GetBackupApiEnabledOk(); ok && *v && !opts.disable {
		fmt.Fprintln(opts.Out, "Backup API is already enabled.")
		return nil
	}
	var message string
	if opts.disable {
		settings.SetBackupApiEnabled(false)
		message = "backup API has been disabled."
	} else {
		settings.SetBackupApiEnabled(true)
		var answer string
		passwordPrompt := &survey.Password{
			Message: "The passphrase to encrypt Appliance Backups when backup API is used:",
		}
		if err := prompt.SurveyAskOne(passwordPrompt, &answer, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
		settings.SetBackupPassphrase(answer)
		message = "Backup API and phassphrase has been updated."
	}

	response, err = client.GlobalSettingsApi.GlobalSettingsPut(ctx).GlobalSettings(settings).Authorization(t).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	fmt.Fprintln(opts.Out, message)
	return nil
}
