package backup

import (
	"context"
	"fmt"
	"io"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
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
		Short: `Controls the state of the backup API.`,
		Long: `This command controls the state of the backup API on the Appgate SDP Collective.
You will be prompted for a passphrase for the backups when enabling the backup API using this command.
The passphrase is required.`,
		Example: `# enable the backup API
$ appgate appliance backup api

# disable the backup API
$ sdpctl appliance backup api --disable`,
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
	t, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
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
		answer, err := prompt.PasswordConfirmation("The passphrase to encrypt Appliance Backups when backup API is used:")
		if err != nil {
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
