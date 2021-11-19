package configure

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type credentialOptions struct {
	Config      *configuration.Config
	Out         io.Writer
	debug       bool
	usernameEnv string
	passwordEnv string
	filePath    string
}

func NewCredentialsCmd(f *factory.Factory) *cobra.Command {
	opts := credentialOptions{
		Config: f.Config,
		Out:    f.IOOutWriter,
		debug:  f.Config.Debug,
	}

	cmd := &cobra.Command{
		Use: "credentials",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		TraverseChildren: true,
		Short:            "set up credentials for logging in to Appgate SDP",
		Long: `Set up credentials for re-use in the appgatectl command.
Credentials will be parsed from environment variables and stored in a file within your appgatectl configuration directory.`,
		Example: `# One line usage with default environment variables:
APPGATECTL_USERNAME=<username> APPGATECTL_PASSWORD=<password> appgatectl configure credentials

# One line usage with custom environment variables
CUSTOM_USERNAME_ENV=<username> CUSTOM_PASSWORD_ENV=<password> appgatectl configure credentials --username-env=CUSTOM_USERNAME_ENV --password-env=CUSTOM_PASSWORD_ENV

# Usage with pre-defined environment variables:
export APPGATECTL_USERNAME=<username>
export APPGATECTL_PASSWORD=<password>
appgatectl configure credentials`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return credentialsRun(&opts)
		},
	}

	defaultFilePath := filepath.Join(configuration.ConfigDir(), "credentials")
	cmd.PersistentFlags().StringVar(&opts.usernameEnv, "username-env", "APPGATECTL_USERNAME", "set env variable for username")
	cmd.PersistentFlags().StringVar(&opts.passwordEnv, "password-env", "APPGATECTL_PASSWORD", "set env variable for password")
	cmd.PersistentFlags().StringVarP(&opts.filePath, "file", "f", defaultFilePath, "filepath to credentials file")

	return cmd
}

func credentialsRun(opts *credentialOptions) error {
	// Set credentials from env vars
	// One empty field is ok since the user should be able to run the login command with missing fields in env variables
	// If both fields are empty, return an error
	username := os.Getenv(opts.usernameEnv)
	password := os.Getenv(opts.passwordEnv)
	if len(username) <= 0 && len(password) <= 0 {
		fmt.Fprintln(opts.Out, "invalid credentials")
		return errors.New("invalid credentials")
	}

	joinStrings := []string{}
	if len(username) > 0 {
		joinStrings = append(joinStrings, fmt.Sprintf("username=%s", username))
	}
	if len(password) > 0 {
		joinStrings = append(joinStrings, fmt.Sprintf("password=%s", password))
	}
	b := []byte(strings.Join(joinStrings, "\n"))

	path := filepath.FromSlash(opts.filePath)
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, b, 0600)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "Stored credentials in %s\n", opts.filePath)
	viper.Set("credentials_file", opts.filePath)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}
	fmt.Fprintln(opts.Out, "Updated configuration with credentials file path")

	return nil
}
