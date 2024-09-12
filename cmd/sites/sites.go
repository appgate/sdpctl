package sites

import (
	"io"

	pkgapi "github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type SitesOptions struct {
	configuration.Config
	SitesAPI      *pkgapi.SitesAPI
	Out           io.Writer
	NoInteractive bool
	CiMode        bool
}

func NewSitesCmd(f *factory.Factory, configuration *configuration.Config) *cobra.Command {
	opts := SitesOptions{
		Out: f.IOOutWriter,
	}

	cmd := &cobra.Command{
		Use:     "sites",
		Short:   docs.SitesDocRoot.Short,
		Long:    docs.SitesDocRoot.Long,
		Example: docs.SitesDocRoot.ExampleString(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.URL = configuration.URL
			opts.Provider = configuration.Provider
			opts.Insecure = configuration.Insecure
			opts.Version = configuration.Version
			opts.BearerToken = configuration.BearerToken
			opts.NoInteractive = configuration.NoInteractive
			opts.CiMode = configuration.CiMode
			api, err := f.APIClient(configuration)
			if err != nil {
				return err
			}
			token, err := f.Config.GetBearTokenHeaderValue()
			if err != nil {
				return err
			}
			opts.SitesAPI = &pkgapi.SitesAPI{
				API:   api.SitesApi,
				Token: token,
			}
			return nil
		},
	}

	cmd.AddCommand(
		NewSitesListCmd(&opts),
	)

	return cmd
}
