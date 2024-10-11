package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
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
	CustomClient  func() (*http.Client, error)
	debug         bool
	disable       bool
	NoInteractive bool
}

// NewBackupAPICmd return a new Backup API command
func NewBackupAPICmd(f *factory.Factory) *cobra.Command {
	opts := apiOptions{
		Config:       f.Config,
		APIClient:    f.APIClient,
		CustomClient: f.HTTPClient,
		debug:        f.Config.Debug,
		Out:          f.IOOutWriter,
		In:           f.Stdin,
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
	apiClient, err := opts.APIClient(opts.Config)
	if err != nil {
		return err
	}
	ctx := context.Background()
	t, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	settings, response, err := apiClient.GlobalSettingsApi.GlobalSettingsGet(ctx).Authorization(t).Execute()
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

	cfg := apiClient.GetConfig()
	url, err := cfg.ServerURLWithContext(ctx, "GlobalSettingsApiService.GlobalSettingsPut")
	if err != nil {
		return err
	}
	customGlobalSettings := GlobalSettings(*settings)
	b, err := JSONEncode(customGlobalSettings)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(b)
	ctx = context.WithValue(ctx, api.ContextAcceptValue, fmt.Sprintf("application/vnd.appgate.peer-v%d+gpg", opts.Config.Version))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url+"/global-settings", body)
	if err != nil {
		return err
	}
	customClient, err := opts.CustomClient()
	if err != nil {
		return err
	}

	response, err = customClient.Do(req)
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	fmt.Fprintln(opts.Out, message)
	return nil
}

type GlobalSettings openapi.GlobalSettings

// JSONEncode is needed so that '<', '>' and '&' does not get replaced by escape characters
// in passwords, which the default json.Marshal will do
func JSONEncode(v interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	result := strings.TrimSpace(buffer.String())
	return []byte(result), err
}
