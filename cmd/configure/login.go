package configure

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type loginOptions struct {
	Config     *configuration.Config
	APIClient  func(Config *configuration.Config) (*openapi.APIClient, error)
	Timeout    int
	url        string
	provider   string
	debug      bool
	insecure   bool
	apiversion int
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
	loginCmd.PersistentFlags().IntVarP(&opts.apiversion, "apiversion", "", f.Config.Version, "peer API version")
	loginCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")

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
	if opts.apiversion != 0 {
		cfg.Version = opts.apiversion
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
	var qs = []*survey.Question{
		{
			Name: "username",
			Prompt: &survey.Input{
				Message: "username",
			},
			Validate: survey.Required,
		},
		{
			Name: "password",
			Prompt: &survey.Password{
				Message: "password",
			},
			Validate: survey.Required,
		},
	}
	answers := struct {
		Username string
		Password string
	}{}

	if err := survey.Ask(qs, &answers); err != nil {
		return err
	}
	loginOpts := openapi.LoginRequest{
		ProviderName: cfg.Provider,
		Username:     openapi.PtrString(answers.Username),
		Password:     openapi.PtrString(answers.Password),
		DeviceId:     uuid.New().String(),
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
	log.Infof("Config updated %s", viper.ConfigFileUsed())
	return nil
}
