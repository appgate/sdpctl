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

func NewSitesCmd(f *factory.Factory, parentOpts *configuration.Config) (*cobra.Command, error) {
	opts := SitesOptions{
		Out: f.IOOutWriter,
	}
	opts.URL = parentOpts.URL
	opts.Provider = parentOpts.Provider
	opts.Insecure = parentOpts.Insecure
	opts.Version = parentOpts.Version
	opts.BearerToken = parentOpts.BearerToken
	opts.NoInteractive = parentOpts.NoInteractive
	opts.CiMode = parentOpts.CiMode

	api, err := f.APIClient(parentOpts)
	if err != nil {
		return nil, err
	}
	token, err := f.Config.GetBearTokenHeaderValue()
	if err != nil {
		return nil, err
	}
	opts.SitesAPI = &pkgapi.SitesAPI{
		API:   api.SitesApi,
		Token: token,
	}

	cmd := &cobra.Command{
		Use:     "sites",
		Short:   docs.SitesDocRoot.Short,
		Long:    docs.SitesDocRoot.Long,
		Example: docs.SitesDocRoot.ExampleString(),
	}

	cmd.AddCommand(
		NewSitesListCmd(&opts),
		NewSitesStatusCmd(f, &opts),
	)

	return cmd, nil
}
