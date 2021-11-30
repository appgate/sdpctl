package configure

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type loginOptions struct {
	Config    *configuration.Config
	APIClient func(Config *configuration.Config) (*openapi.APIClient, error)
	Timeout   int
	url       string
	provider  string
	debug     bool
	insecure  bool
	remember  bool
}

// NewLoginCmd return a new login command
func NewLoginCmd(f *factory.Factory) *cobra.Command {
	opts := loginOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		Timeout:   10,
		debug:     f.Config.Debug,
	}
	var loginCmd = &cobra.Command{
		Use: "login",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Short: "login and authenticate to appgate SDP collective",
		Long:  `Setup a configuration file towards your appgate sdp collective to be able to interact with the collective.`,
		RunE: func(c *cobra.Command, args []string) error {
			return loginRun(c, args, &opts)
		},
	}

	loginCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	loginCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	loginCmd.PersistentFlags().StringVar(&opts.provider, "provider", "local", "identity provider")
	loginCmd.PersistentFlags().BoolVar(&opts.remember, "remember-me", false, "remember login credentials")

	return loginCmd
}

func loginRun(cmd *cobra.Command, args []string, opts *loginOptions) error {
	cfg := opts.Config
	if opts.url != "" {
		cfg.URL = opts.url
	}
	if opts.provider != "" {
		cfg.Provider = opts.provider
	}
	if opts.insecure {
		cfg.Insecure = true
	}
	if cfg.URL == "" {
		return fmt.Errorf("no addr set.")
	}

	client, err := opts.APIClient(cfg)
	if err != nil {
		return err
	}

	// Get credentials from credentials file
	// Overwrite credentials with values set through environment variables
	credentials, err := opts.Config.LoadCredentials()
	if err != nil {
		return err
	}
	if envUsername := viper.GetString("username"); len(envUsername) > 0 {
		credentials.Username = envUsername
	}
	if envPassword := viper.GetString("password"); len(envPassword) > 0 {
		credentials.Password = envPassword
	}

	if len(credentials.Username) <= 0 {
		err := survey.AskOne(&survey.Input{
			Message: "Username:",
		}, &credentials.Username, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if len(credentials.Password) <= 0 {
		err := survey.AskOne(&survey.Password{
			Message: "Password:",
		}, &credentials.Password, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}

	if opts.remember {
		if err := rememberCredentials(cfg, credentials); err != nil {
			return fmt.Errorf("Failed to store credentials: %+v", err)
		}
	}

	loginOpts := openapi.LoginRequest{
		ProviderName: cfg.Provider,
		Username:     openapi.PtrString(credentials.Username),
		Password:     openapi.PtrString(credentials.Password),
		DeviceId:     cfg.DeviceID,
	}
	loginResponse, _, err := client.LoginApi.LoginPost(context.Background()).LoginRequest(loginOpts).Execute()
	if err != nil {
		if err, ok := err.(openapi.GenericOpenAPIError); ok {
			if err, ok := err.Model().(openapi.InlineResponse406); ok {
				return fmt.Errorf(
					"You are using the wrong apiversion (peer api version) for you appgate sdp collective, you are using %d; min: %d max: %d",
					cfg.Version,
					err.GetMinSupportedVersion(),
					err.GetMaxSupportedVersion(),
				)
			}
		}
		return err
	}

	viper.Set("bearer", *openapi.PtrString(*loginResponse.Token))
	viper.Set("expires_at", loginResponse.Expires.String())
	viper.Set("url", cfg.URL)
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	log.WithField("config file", viper.ConfigFileUsed()).Info("Config updated")
	return nil
}

func rememberCredentials(cfg *configuration.Config, credentials *configuration.Credentials) error {
	q := []*survey.Question{
		{
			Name: "remember",
			Prompt: &survey.Select{
				Message: "What credentials should be saved?",
				Options: []string{"both", "only username", "only password"},
				Default: "both",
			},
		},
		{
			Name: "path",
			Prompt: &survey.Input{
				Message: "Path to credentials file:",
				Default: fmt.Sprintf("%s/credentials", configuration.ConfigDir()),
			},
			Validate: survey.Required,
		},
	}

	answers := struct {
		Remember string `survey:"remember"`
		Path     string
	}{}

	if err := survey.Ask(q, &answers); err != nil {
		return err
	}

	credentialsCopy := &configuration.Credentials{}
	switch answers.Remember {
	case "only username":
		credentialsCopy.Username = credentials.Username
	case "only password":
		credentialsCopy.Password = credentials.Password
	default:
		credentialsCopy.Username = credentials.Username
		credentialsCopy.Password = credentials.Password
	}

	// Allow variable expansion for path
	cfg.CredentialsFile = os.ExpandEnv(answers.Path)

	if err := cfg.StoreCredentials(credentialsCopy); err != nil {
		return err
	}

	return nil
}
