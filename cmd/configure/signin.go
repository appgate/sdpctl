package configure

import (
	"fmt"
	"io"
	"os"

	"github.com/appgate/sdpctl/pkg/configuration"

	"github.com/appgate/sdpctl/pkg/auth"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type signinOptions struct {
	f      *factory.Factory
	StdErr io.Writer
}

// NewSigninCmd return a new signin command
func NewSigninCmd(f *factory.Factory) *cobra.Command {
	opts := signinOptions{
		f:      f,
		StdErr: f.StdErr,
	}
	var signinCmd = &cobra.Command{
		Use:        "signin",
		SuggestFor: []string{"sigin", "singin"},
		Annotations: map[string]string{
			configuration.SkipAuthCheck: "true",
		},
		Aliases: []string{"login"},
		Args:    cobra.NoArgs,
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
	if len(os.Getenv("SDPCTL_NO_KEYRING")) > 0 {
		fmt.Fprintf(opts.StdErr, "the %s command has no effect when using SDPCTL_NO_KEYRING\n", cmd.CalledAs())
		fmt.Fprintln(opts.StdErr, "When the keyring integration is disabled, you must provide credentials for each command call.")
		return nil
	}

	cfg := opts.f.Config
	// If there's an existing bearer token present, we will clear it and renew the authentication
	if err := cfg.ClearBearer(); err != nil {
		// not a fatal error
		fmt.Fprintln(opts.StdErr, err)
		fmt.Fprintln(opts.StdErr, auth.KeyringWarningMessage)
		log.WithError(err).Warn("Failed to delete auth token")
	}
	if err := auth.Signin(opts.f); err != nil {
		return err
	}
	log.WithField("config file", viper.ConfigFileUsed()).Info("Sign in event")
	fmt.Println("Successfully signed in")
	return nil
}
