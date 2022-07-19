package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/spf13/cobra"
)

type apiOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	In        io.ReadCloser
	CanPrompt bool
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
		In:        f.Stdin,
		CanPrompt: f.CanPrompt(),
	}
	var cmd = &cobra.Command{
		Use:     "api",
		Short:   docs.ApplianceBackupAPIDoc.Short,
		Long:    docs.ApplianceBackupAPIDoc.Long,
		Example: docs.ApplianceBackupAPIDoc.ExampleString(),
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
		hasStdin := false
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			hasStdin = true
		}
		answer, err := getPassPhrase(opts.In, opts.CanPrompt, hasStdin)
		if err != nil {
			return err
		}
		settings.SetBackupApiEnabled(true)
		settings.SetBackupPassphrase(answer)
		message = "Backup API and passphrase has been updated."
	}

	response, err = client.GlobalSettingsApi.GlobalSettingsPut(ctx).GlobalSettings(*settings).Authorization(t).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	fmt.Fprintln(opts.Out, message)
	return nil
}

func getPassPhrase(stdIn io.Reader, canPrompt, hasStdin bool) (string, error) {
	if hasStdin {
		buf, err := io.ReadAll(stdIn)
		if err != nil {
			return "", fmt.Errorf("could not read input from stdin %s", err)
		}
		return strings.TrimSuffix(string(buf), "\n"), nil
	}
	if !canPrompt {
		return "", cmdutil.ErrMissingTTY
	}
	return prompt.PasswordConfirmation("The passphrase to encrypt Appliance Backups when backup API is used:")
}
