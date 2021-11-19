package configure

import (
	"fmt"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefaultAPIVersion is based on the latest peer api version.
const DefaultAPIVersion = 16

type configureOptions struct {
	Config *configuration.Config
}

// NewCmdConfigure return a new Configure command
func NewCmdConfigure(f *factory.Factory) *cobra.Command {
	opts := configureOptions{
		Config: f.Config,
	}
	cmd := &cobra.Command{
		Use: "configure",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Short: "Configure your appgate SDP collective",
		Long:  `Setup a configuration file towards your appgate sdp collective to be able to interact with the collective.`,
		RunE: func(c *cobra.Command, args []string) error {
			return configRun(c, args, &opts)
		},
	}

	cmd.AddCommand(NewCredentialsCmd(f))
	cmd.AddCommand(NewLoginCmd(f))

	return cmd
}

func configRun(cmd *cobra.Command, args []string, opts *configureOptions) error {
	var qs = []*survey.Question{
		{
			Name: "url",
			Prompt: &survey.Input{
				Message: "Enter the url for the controller API (example https://appgate.controller.com/admin)",
				Default: opts.Config.URL,
			},
			Validate: survey.Required,
		},
		{
			Name: "insecure",
			Prompt: &survey.Select{
				Message: "Whether server should be accessed without verifying the TLS certificate",
				Options: []string{"true", "false"},
				Default: strconv.FormatBool(opts.Config.Insecure),
			},
		},
		{
			Name: "version",
			Prompt: &survey.Select{
				Message: "API Version",
				Options: []string{"14", "15", "16"},
				Default: fmt.Sprintf("%x", DefaultAPIVersion),
			},
		},
	}
	answers := struct {
		URL      string
		Insecure string
		Version  string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return err
	}
	log.Debugf("Answers %+v", answers)

	viper.Set("url", answers.URL)
	i, _ := strconv.ParseBool(answers.Insecure)
	viper.Set("insecure", i)
	v, _ := strconv.Atoi(answers.Version)
	viper.Set("api_version", v)

	if err := viper.WriteConfig(); err != nil {
		return err
	}
	log.Infof("Config updated %s", viper.ConfigFileUsed())
	return nil
}
