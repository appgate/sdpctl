package configure

import (
	"fmt"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
	}
	answers := struct {
		URL      string
		Insecure string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return err
	}

	if answers.Insecure == "false" {
		prompt := &survey.Input{
			Message: "Path to PEM file",
			Default: opts.Config.PemFilePath,
		}
		v := func(val interface{}) error {
			if str, ok := val.(string); ok {
				if ok, err := util.FileExists(str); err != nil || !ok {
					return fmt.Errorf("File not found %s", str)
				}
			}
			return nil
		}
		var p string
		if err := survey.AskOne(prompt, &p, survey.WithValidator(v)); err != nil {
			return err
		}

		viper.Set("pem_filepath", p)
	}

	url, err := configuration.NormalizeURL(answers.URL)
	if err != nil {
		return err
	}
	viper.Set("url", url)
	i, _ := strconv.ParseBool(answers.Insecure)
	viper.Set("insecure", i)
	viper.Set("device_id", configuration.DefaultDeviceID())

	if err := viper.WriteConfig(); err != nil {
		return err
	}
	log.Infof("Config updated %s", viper.ConfigFileUsed())
	return nil
}
