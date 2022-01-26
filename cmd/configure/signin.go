package configure

import (
	"github.com/appgate/appgatectl/pkg/auth"
	"github.com/appgate/appgatectl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type signinOptions struct {
	f        *factory.Factory
	remember bool
}

// NewSigninCmd return a new signin command
func NewSigninCmd(f *factory.Factory) *cobra.Command {
	opts := signinOptions{
		f: f,
	}
	var signinCmd = &cobra.Command{
		Use: "signin",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Aliases: []string{"login"},
		Short:   "Sign in and authenticate to Appgate SDP Collective",
		Long: `Sign in to the Appgate SDP Collective using the configuration file created by the 'appgatectl configure' command.
This will fetch a token on valid authentication which will be valid for 24 hours and stored in the configuration.`,
		RunE: func(c *cobra.Command, args []string) error {
			return signinRun(c, args, &opts)
		},
	}

	flags := signinCmd.Flags()

	flags.BoolVar(&opts.remember, "remember-me", false, "remember sign in credentials")

	return signinCmd
}

func signinRun(cmd *cobra.Command, args []string, opts *signinOptions) error {
	if err := auth.Signin(opts.f, opts.remember, true); err != nil {
		return err
	}
	log.WithField("config file", viper.ConfigFileUsed()).Info("Config updated")
	return nil
}
