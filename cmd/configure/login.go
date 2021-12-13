package configure

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/auth"
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

	flags := loginCmd.Flags()
	flags.BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	flags.StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	flags.StringVar(&opts.provider, "provider", "local", "identity provider")
	flags.BoolVar(&opts.remember, "remember-me", false, "remember login credentials")

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
	authenticator := auth.NewAuth(client)
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
	ctx := context.Background()
	acceptHeaderFormatString := "application/vnd.appgate.peer-v%d+json"
	// initial authtentication, this will fail, since we will use the login response
	// to compute the correct peerVersion used in the selected appgate sdp collective.
	_, minMax, err := authenticator.Authentication(context.WithValue(ctx, openapi.ContextAcceptHeader, fmt.Sprintf(acceptHeaderFormatString, 5)), loginOpts)
	if err != nil && minMax == nil {
		return fmt.Errorf("invalid credentials %w", err)
	}
	if minMax != nil {
		viper.Set("api_version", minMax.Max)
		cfg.Version = int(minMax.Max)

	}

	acceptValue := fmt.Sprintf(acceptHeaderFormatString, minMax.Max)
	ctxWithAccept := context.WithValue(ctx, openapi.ContextAcceptHeader, acceptValue)
	providers, err := authenticator.ProviderNames(ctxWithAccept)
	if err != nil {
		return err
	}
	prompt := &survey.Select{
		Message: "Choose a provider:",
		Options: providers,
	}
	if err := survey.AskOne(prompt, &loginOpts.ProviderName); err != nil {
		return err
	}

	loginResponse, _, err := authenticator.Authentication(ctxWithAccept, loginOpts)
	if err != nil {
		return err
	}
	authToken := fmt.Sprintf("Bearer %s", loginResponse.GetToken())
	_, err = authenticator.Authorization(ctxWithAccept, loginOpts.GetPassword(), authToken)
	if errors.Is(err, auth.ErrPreConditionFailed) {
		otp, err := authenticator.InitializeOTP(ctxWithAccept, loginOpts.GetPassword(), authToken)
		if err != nil {
			return err
		}
		switch otpType := otp.GetType(); otpType {
		case "Secret":
			fmt.Println("One-time password initialization is required!")
			fmt.Printf("\nTo initialize the timed based OTP enter the following secret: %s\n", otp.GetSecret())
			optKey := &survey.Input{
				Message: "Please enter your one-time password:",
			}
			var answer string
			if err := survey.AskOne(optKey, &answer); err != nil {
				return err
			}
			newToken, err := authenticator.PushOTP(ctxWithAccept, answer, authToken)
			if err != nil {
				return err
			}
			authToken = fmt.Sprintf("Bearer %s", newToken.GetToken())
		case "AlreadySeeded":
			optKey := &survey.Input{
				Message: "Please enter your one-time password:",
			}
			var answer string
			if err := survey.AskOne(optKey, &answer); err != nil {
				return err
			}
			newToken, err := authenticator.PushOTP(ctxWithAccept, answer, authToken)
			if err != nil {
				return err
			}
			authToken = fmt.Sprintf("Bearer %s", newToken.GetToken())
		}
	} else if err != nil {
		return err
	}
	authorizationToken, err := authenticator.Authorization(ctxWithAccept, loginOpts.GetPassword(), authToken)
	if err != nil {
		return err
	}

	cfg.BearerToken = authorizationToken.GetToken()
	cfg.ExpiresAt = authorizationToken.Expires.String()

	viper.Set("bearer", cfg.BearerToken)
	viper.Set("expires_at", cfg.ExpiresAt)
	viper.Set("url", cfg.URL)
	host, err := cfg.GetHost()
	if err != nil {
		return err
	}

	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	allAppliances, err := a.List(ctxWithAccept, nil)
	if err != nil {
		return err
	}
	primaryController, err := appliancepkg.FindPrimaryController(allAppliances, host)
	if err != nil {
		return err
	}
	stats, _, err := a.Stats(ctxWithAccept)
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
