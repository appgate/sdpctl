package configure

import (
	"fmt"

	"github.com/appgate/sdpctl/pkg/auth"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type signinOptions struct {
	f *factory.Factory
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
		Short:   docs.ConfigureSigninDocs.Short,
		Long:    docs.ConfigureSigninDocs.Long,
		Example: docs.ConfigureSigninDocs.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			return signinRun(c, args, &opts)
		},
	}

	return signinCmd
}

func signinRun(cmd *cobra.Command, args []string, opts *signinOptions) error {
	noInteractive, err := cmd.Flags().GetBool("no-interactive")
	if err != nil {
		return err
	}
	if err := auth.Signin(opts.f, true, noInteractive); err != nil {
		return err
	}
	log.WithField("config file", viper.ConfigFileUsed()).Info("Sign in event")
	fmt.Println("Successfully signed in")
	return nil
}
