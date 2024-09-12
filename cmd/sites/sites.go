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

func NewSitesCmd(f *factory.Factory) *cobra.Command {
	opts := SitesOptions{
		Out: f.IOOutWriter,
	}

	cmd := &cobra.Command{
		Use:     "sites",
		Short:   docs.SitesDocRoot.Short,
		Long:    docs.SitesDocRoot.Long,
		Example: docs.SitesDocRoot.ExampleString(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.URL = f.BaseURL()
			opts.Provider = f.Config.Provider
			opts.Insecure = f.Config.Insecure
			opts.Version = f.Config.Version
			opts.BearerToken = f.Config.BearerToken
			opts.NoInteractive = f.Config.NoInteractive
			opts.CiMode = f.Config.CiMode
			api, err := f.APIClient(f.Config)
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
