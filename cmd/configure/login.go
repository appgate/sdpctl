package configure

import (
	"github.com/appgate/appgatectl/pkg/auth"
	"github.com/appgate/appgatectl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type loginOptions struct {
	f        *factory.Factory
	remember bool
}

// NewLoginCmd return a new login command
func NewLoginCmd(f *factory.Factory) *cobra.Command {
	opts := loginOptions{
		f: f,
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

	flags := loginCmd.Flags()

	flags.BoolVar(&opts.remember, "remember-me", false, "remember login credentials")

	return loginCmd
}

func loginRun(cmd *cobra.Command, args []string, opts *loginOptions) error {
	if err := auth.Login(opts.f, opts.remember); err != nil {
		return err
	}
	log.WithField("config file", viper.ConfigFileUsed()).Info("Config updated")
	return nil
}
