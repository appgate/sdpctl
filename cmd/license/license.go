package license

import (
	"io"
	"net/http"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

type licenseOpts struct {
	Out        io.Writer
	BaseURL    string
	HTTPClient func() (*http.Client, error)
}

// NewLicenseCmd return a new license subcommand
func NewLicenseCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "license",
		TraverseChildren: true,
		Short:            docs.LicenseRootDoc.Short,
		Long:             docs.LicenseRootDoc.Long,
		Hidden:           true,
	}

	opts := &licenseOpts{
		Out:        f.IOOutWriter,
		HTTPClient: f.CustomHTTPClient,
		BaseURL:    f.BaseURL(),
	}

	cmd.AddCommand(NewPruneCmd(opts))

	return cmd
}
