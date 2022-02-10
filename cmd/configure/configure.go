package configure

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
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
		Short: "Configure your Appgate SDP Collective",
		Long: `Setup a configuration file towards your Appgate SDP Collective to be able to interact with the collective. By default, the configuration file
will be created in a default directory in depending on your system. This can be overridden by setting the 'SDPCTL_CONFIG_DIR' environment variable.
See 'sdpctl help environment' for more information on using environment variables.`,
		Example: `sdpctl configure
sdpctl configure --pem <path/to/pem>
SDPCTL_CONFIG_DIR=<path/to/config/dir sdpctl configure`,
		RunE: func(c *cobra.Command, args []string) error {
			return configRun(c, args, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.PEM, "pem", "", "Path to PEM file to use for request certificate validation")

	cmd.AddCommand(NewSigninCmd(f))

	return cmd
}

func configRun(cmd *cobra.Command, args []string, opts *configureOptions) error {
	q := &survey.Input{
		Message: "Enter the url for the controller API (example https://appgate.controller.com/admin)",
		Default: opts.Config.URL,
	}
	var URL string
	err := prompt.SurveyAskOne(q, &URL, survey.WithValidator(survey.Required))
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
	fmt.Println("Configuration updated successfully")
	return nil
}
