package configure

import (
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
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
	log.Debugf("Answers %+v", answers)

	viper.Set("url", answers.URL)
	i, _ := strconv.ParseBool(answers.Insecure)
	viper.Set("insecure", i)
	viper.Set("device_id", defaultDeviceID())

	if err := viper.WriteConfig(); err != nil {
		return err
	}
	log.Infof("Config updated %s", viper.ConfigFileUsed())
	return nil
}

func defaultDeviceID() string {
	readAndParseUUID := func() (string, error) {
		// machine.ID() tries to read
		// /etc/machine-id on Linux
		// /etc/hostid on BSD
		// ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID on OSX
		// reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid on Windows
		// and tries to parse the value as a UUID
		// https://github.com/denisbrodbeck/machineid
		id, err := machineid.ID()
		if err != nil {
			return "", err
		}
		uid, err := uuid.Parse(id)
		if err != nil {
			return "", err
		}
		return uid.String(), nil
	}
	// if we cant get a valid UUID based on the machine ID, we will fallback to a random UUID value.
	v, err := readAndParseUUID()
	if err != nil {
		return uuid.New().String()
	}
	return v
}
