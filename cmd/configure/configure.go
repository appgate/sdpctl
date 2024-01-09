package configure

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/network"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type configureOptions struct {
	Config    *configuration.Config
	CertPath  string
	Out       io.Writer
	StdErr    io.Writer
	CanPrompt bool
	URL       string
}

// NewCmdConfigure return a new Configure command
func NewCmdConfigure(f *factory.Factory) *cobra.Command {
	opts := configureOptions{
		Config:    f.Config,
		Out:       f.IOOutWriter,
		StdErr:    f.StdErr,
		CanPrompt: f.CanPrompt(),
	}
	cmd := &cobra.Command{
		Use: "configure",
		Annotations: map[string]string{
			configuration.SkipAuthCheck: "true",
		},
		Short:   docs.ConfigureDocs.Short,
		Long:    docs.ConfigureDocs.Long,
		Example: docs.ConfigureDocs.ExampleString(),
		Args:    cobra.MatchAll(cobra.MaximumNArgs(1), argValidation),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			noInteractive, err := cmd.Flags().GetBool("no-interactive")
			if err != nil {
				return err
			}
			switch len(args) {
			case 0:
				if noInteractive || !opts.CanPrompt {
					return errors.New("Can't prompt, You need to provide all arguments, for example 'sdpctl configure company.controller.com'")
				}
				q := &survey.Input{
					Message: "Enter the url for the Controller API (example https://controller.company.com:8443)",
					Default: opts.Config.URL,
				}

				err := prompt.SurveyAskOne(q, &opts.URL, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			case 1:
				opts.URL = args[0]
			default:
				return fmt.Errorf("Accepts at most %d arg(s), received %d", 1, len(args))
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return configRun(c, args, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.CertPath, "pem", "", "Path to PEM file to use for request certificate validation")

	cmd.AddCommand(NewSigninCmd(f))

	return cmd
}

func configRun(cmd *cobra.Command, args []string, opts *configureOptions) error {
	if len(opts.URL) < 1 {
		return errors.New("Missing URL for the Controller")
	}
	if len(opts.CertPath) > 0 {
		opts.CertPath = filesystem.AbsolutePath(opts.CertPath)
		cert, err := configuration.ReadPemFile(opts.CertPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "Added certificate as trusted source for sdpctl from %s\n", opts.CertPath)
		fmt.Fprintln(opts.Out, configuration.CertificateDetails(cert))

		viper.Set("pem_base64", base64.StdEncoding.EncodeToString(cert.Raw))
	}
	u, err := configuration.NormalizeConfigurationURL(opts.URL)
	if err != nil {
		return fmt.Errorf("Could not determine URL for %s %s", opts.URL, err)
	}
	viper.Set("url", u)
	opts.Config.URL = u
	viper.Set("device_id", configuration.DefaultDeviceID())

	h, err := opts.Config.GetHost()
	if err != nil {
		return fmt.Errorf("Could not determine hostname for %s %s", opts.URL, err)
	}
	if err := network.ValidateHostnameUniqueness(h); err != nil {
		fmt.Fprintln(opts.StdErr, err.Error())
	}

	if err := viper.WriteConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// if its a new Collective config, and the directory is empty
			// try to write a plain config
			if err := viper.SafeWriteConfig(); err != nil {
				return err
			}
		}
	}
	// Clear old credentials when configuring
	if err := opts.Config.ClearCredentials(); err != nil {
		log.Warnf("Ran configure command, unable to clear credentials %s", err)
	}
	log.WithField("file", viper.ConfigFileUsed()).Info("Config updated")
	fmt.Fprintln(opts.Out, "Configuration updated successfully")
	return nil
}

func argValidation(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		arg := args[0]
		regex := regexp.MustCompile(`[signin]{3,}`)
		if regex.MatchString(arg) {
			return fmt.Errorf("'%s' is not a valid argument. Did you mean 'signin'?", arg)
		}
		// If arg is missing protocol prefix, temporarily add one to validate the url
		if !strings.HasPrefix(arg, "https://") || !strings.HasPrefix(arg, "http://") {
			arg = "https://" + arg
		}
		if !util.IsValidURL(arg) {
			return fmt.Errorf("'%s' is not a valid URL", arg)
		}
	}
	return nil
}
