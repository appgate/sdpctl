package configure

import (
	"fmt"
	"os"

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
	PEM    string
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

	cmd.Flags().StringVar(&opts.PEM, "pem", "", "Path to PEM file to use for request certificate validation")

	cmd.AddCommand(NewLoginCmd(f))

	return cmd
}

func configRun(cmd *cobra.Command, args []string, opts *configureOptions) error {
	prompt := &survey.Input{
		Message: "Enter the url for the controller API (example https://appgate.controller.com/admin)",
		Default: opts.Config.URL,
	}
	var URL string
	err := survey.AskOne(prompt, &URL, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	if len(opts.PEM) > 0 {
		opts.PEM = os.ExpandEnv(opts.PEM)
		if ok, err := util.FileExists(opts.PEM); err != nil || !ok {
			return fmt.Errorf("File not found: %s", opts.PEM)
		}
		viper.Set("pem_filepath", opts.PEM)
	}

	viper.Set("url", URL)
	viper.Set("device_id", configuration.DefaultDeviceID())
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	log.WithField("file", viper.ConfigFileUsed()).Info("Config updated")
	return nil
}
