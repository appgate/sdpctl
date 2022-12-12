package backup

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/spf13/cobra"
)

type apiOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	In            io.ReadCloser
	APIClient     func(c *configuration.Config) (*openapi.APIClient, error)
	debug         bool
	disable       bool
	NoInteractive bool
}

// NewBackupAPICmd return a new Backup API command
func NewBackupAPICmd(f *factory.Factory) *cobra.Command {
	opts := apiOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		In:        f.Stdin,
	}
	var cmd = &cobra.Command{
		Use:     "api",
		Short:   docs.ApplianceBackupAPIDoc.Short,
		Long:    docs.ApplianceBackupAPIDoc.Long,
		Example: docs.ApplianceBackupAPIDoc.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			var err error
			if opts.NoInteractive, err = c.Flags().GetBool("no-interactive"); err != nil {
				return err
			}
			if !f.CanPrompt() {
				opts.NoInteractive = true
			}
			return backupAPIrun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.disable, "disable", false, "Disable the Backup API")

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
		fmt.Fprintln(opts.Out, "The Backup API is already enabled")
		return nil
	}
	var message string
	if opts.disable {
		settings.SetBackupApiEnabled(false)
		message = "The Backup API has been disabled"
	} else {
		hasStdin := false
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			hasStdin = true
		}
		answer, err := prompt.GetPassphrase(opts.In, !opts.NoInteractive, hasStdin, "The passphrase to encrypt the appliance backups when the Backup API is used:")
		if err != nil {
			return err
		}
		settings.SetBackupApiEnabled(true)
		settings.SetBackupPassphrase(answer)
		message = "The Backup API and the passphrase have been updated"
	}

	response, err = client.GlobalSettingsApi.GlobalSettingsPut(ctx).GlobalSettings(*settings).Authorization(t).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	fmt.Fprintln(opts.Out, message)
	return nil
}
