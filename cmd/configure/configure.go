package configure

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
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
		opts.PEM = os.ExpandEnv(opts.PEM)
		if absPath, err := filepath.Abs(opts.PEM); err == nil {
			opts.PEM = absPath
		}
		if ok, err := util.FileExists(opts.PEM); err != nil || !ok {
			return fmt.Errorf("File not found: %s", opts.PEM)
		}
		viper.Set("pem_filepath", opts.PEM)
	} else {
		if existing := viper.GetString("pem_filepath"); len(existing) > 0 {
			viper.Set("pem_filepath", "")
		}
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
