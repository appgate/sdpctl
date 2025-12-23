package entitlements

import (
	"io"

	pkgapi "github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type EntitlementOptions struct {
	configuration.Config
	EntitlementsAPI      *pkgapi.EntitlementsAPI
	Out           io.Writer
	NoInteractive bool
	CiMode        bool
}

func NewEntitlementsMigrationCmd(f *factory.Factory) *cobra.Command {
	opts := EntitlementOptions{
		Out: f.IOOutWriter,
	}

	cmd := &cobra.Command{
		Use:     "entitlements",
		Short:   docs.SitesDocRoot.Short,
		Long:    docs.SitesDocRoot.Long,
		Example: docs.SitesDocRoot.ExampleString(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			api, err := f.APIClient(f.Config)
			if err != nil {
				return err
			}
			token, err := f.Config.GetBearTokenHeaderValue()
			if err != nil {
				return err
			}
			opts.EntitlementsAPI = &pkgapi.EntitlementsAPI{
				API:   api.EntitlementsApi,
				Token: token,
			}
			
			return nil
		},
	}

	cmd.AddCommand(
		NewCloudMigrationsCmd(&opts),
	)

	return cmd
}
