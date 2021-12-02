package configure

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
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
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
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
		Appliance: f.Appliance,
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

	loginResponse, mm, err := login(client, loginOpts)
	if err != nil {
		return err
	}
	if mm != nil {
		viper.Set("api_version", mm.max)
	}

	viper.Set("bearer", *openapi.PtrString(*loginResponse.Token))
	viper.Set("expires_at", loginResponse.Expires.String())
	viper.Set("url", cfg.URL)

	host, err := cfg.GetHost()
	if err != nil {
		return err
	}
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	allAppliances, err := a.GetAll(ctx)
	if err != nil {
		return err
	}
	primaryController, err := appliancepkg.FindPrimaryController(allAppliances, host)
	if err != nil {
		return err
	}
	stats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	v, err := appliancepkg.GetPrimaryControllerVersion(*primaryController, stats)
	if err != nil {
		return err
	}
	viper.Set("primary_controller_version", v.String())
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	log.WithField("config file", viper.ConfigFileUsed()).Info("Config updated")
	return nil
}

type minMax struct {
	min, max int32
}

func login(client *openapi.APIClient, loginOpts openapi.LoginRequest) (*openapi.LoginResponse, *minMax, error) {
	c := client
	// we will use a invalid accept header to trigger an error, and in the response body we can see that min-max values we can use
	// for the current sdp collective.
	c.GetConfig().AddDefaultHeader("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+json", 5))

	login := func() (*openapi.LoginResponse, *minMax, error) {
		loginResponse, _, err := c.LoginApi.LoginPost(context.Background()).LoginRequest(loginOpts).Execute()
		if err != nil {
			if err, ok := err.(openapi.GenericOpenAPIError); ok {
				if model, ok := err.Model().(openapi.InlineResponse406); ok {
					mm := &minMax{
						min: model.GetMinSupportedVersion(),
						max: model.GetMaxSupportedVersion(),
					}
					return &loginResponse, mm, err
				}
			}
			return nil, nil, err
		}
		return &loginResponse, nil, err
	}
	// login first with invalid accept header
	invalid, mm, err := login()
	if err != nil {
		if mm != nil {
			// login with the highest available accept header
			c.GetConfig().AddDefaultHeader("Accept", fmt.Sprintf("application/vnd.appgate.peer-v%d+json", mm.max))
			login, _, err := login()
			if err != nil {
				return nil, mm, err
			}
			return login, mm, err
		}
		return nil, mm, err
	}
	return invalid, mm, err
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
