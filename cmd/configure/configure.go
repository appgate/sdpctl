package configure

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/network"
	"github.com/appgate/sdpctl/pkg/prompt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type configureOptions struct {
	Config    *configuration.Config
	PEM       string
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
		Args: func(cmd *cobra.Command, args []string) error {
			noInteractive, err := cmd.Flags().GetBool("no-interactive")
			if err != nil {
				return err
			}
			switch len(args) {
			case 0:
				if noInteractive || !opts.CanPrompt {
					return errors.New("Can't prompt, You need to provide all arguments, for example 'sdpctl configure appgate.controller.com'")
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

	cmd.Flags().StringVar(&opts.PEM, "pem", "", "Path to PEM file to use for request certificate validation")

	cmd.AddCommand(NewSigninCmd(f))

	return cmd
}

func readPemFile(path string) (*x509.Certificate, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Path %s does not exist", path)
		}
		return nil, fmt.Errorf("%s - %s", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path %s is a directory, not a file", path)
	}
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("not a file %s %s", path, err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("expected a pem file, could not decode %s", path)
	}

	// See if we can parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func certificateDetails(cert *x509.Certificate) string {
	var sb strings.Builder

	if len(cert.Subject.CommonName) > 0 {
		sb.WriteString(fmt.Sprintf("[Subject]\n\t%s\n", cert.Subject.CommonName))
	}
	if len(cert.Issuer.CommonName) > 0 {
		sb.WriteString(fmt.Sprintf("[Issuer]\n\t%s\n", cert.Issuer.CommonName))
	}
	if cert.SerialNumber != nil {
		sb.WriteString(fmt.Sprintf("[Serial Number]\n\t%s\n", cert.SerialNumber))
	}

	sb.WriteString(fmt.Sprintf("[Not Before]\n\t%s\n", cert.NotBefore))
	sb.WriteString(fmt.Sprintf("[Not After]\n\t%s\n", cert.NotAfter))

	var sha1buf strings.Builder
	for i, f := range sha1.Sum(cert.Raw) {
		if i > 0 {
			sha1buf.Write([]byte(":"))
		}
		sha1buf.Write([]byte(fmt.Sprintf("%02X", f)))
	}
	sb.WriteString(fmt.Sprintf("[Thumbprint SHA-1]\n\t%s\n", sha1buf.String()))

	var sha256buf strings.Builder
	for i, f := range sha256.Sum256(cert.Raw) {
		if i > 0 {
			sha256buf.Write([]byte(":"))
		}
		sha256buf.Write([]byte(fmt.Sprintf("%02X", f)))
	}
	sb.WriteString(fmt.Sprintf("[Thumbprint SHA-256]\n\t%s\n", sha256buf.String()))

	return sb.String()
}

func configRun(cmd *cobra.Command, args []string, opts *configureOptions) error {
	if len(opts.URL) < 1 {
		return errors.New("Missing URL for the Controller")
	}
	if len(opts.PEM) > 0 {
		opts.PEM = filesystem.AbsolutePath(opts.PEM)
		cert, err := readPemFile(opts.PEM)
		if err != nil {
			return err
		}
		fmt.Fprintln(opts.Out, "Added PEM as trusted source for sdpctl")
		fmt.Fprintln(opts.Out, certificateDetails(cert))
		viper.Set("pem_base64", base64.StdEncoding.EncodeToString(cert.Raw))
		viper.Set("pem_filepath", opts.PEM) // deprecated: TODO remove in future version
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
