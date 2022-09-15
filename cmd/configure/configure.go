package configure

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
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
		Short:   docs.ConfigureDocs.Short,
		Long:    docs.ConfigureDocs.Long,
		Example: docs.ConfigureDocs.ExampleString(),
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
		opts.PEM = filesystem.AbsolutePath(opts.PEM)
		if ok, err := util.FileExists(opts.PEM); err != nil || !ok {
			return fmt.Errorf("File not found: %s", opts.PEM)
		}
		viper.Set("pem_filepath", opts.PEM)
	}
	u, err := configuration.NormalizeURL(URL)
	if err != nil {
		return fmt.Errorf("could not determine URL for %s %s", URL, err)
	}
	viper.Set("url", u)
	opts.Config.URL = u
	viper.Set("device_id", configuration.DefaultDeviceID())
	if err := viper.WriteConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return errors.New("collective profile does not have a valid config directory")
		}
		return err
	}
	// Clear old credentials when configuring
	if err := opts.Config.ClearCredentials(); err != nil {
		log.Warnf("ran configure command, unable to clear credentials %s", err)
	}
	log.WithField("file", viper.ConfigFileUsed()).Info("Config updated")
	fmt.Println("Configuration updated successfully")
	return nil
}
