package license

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/spf13/cobra"
)

// NewPruneCmd return a new license prune subcommand
func NewPruneCmd(opts *licenseOpts) *cobra.Command {
	return &cobra.Command{
		Use:     "prune",
		Short:   docs.LicensePruneDoc.Short,
		Long:    docs.LicensePruneDoc.Long,
		Example: docs.LicensePruneDoc.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			return pruneRun(c, args, opts)
		},
	}
}

func pruneRun(cmd *cobra.Command, args []string, opts *licenseOpts) error {
	client, err := opts.HTTPClient()
	if err != nil {
		return fmt.Errorf("could not resolve a HTTP client based on your current config %s", err)
	}
	requestURL := fmt.Sprintf("%s/license/users/prune", opts.BaseURL)
	request, err := http.NewRequest(http.MethodDelete, requestURL, nil)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if response.StatusCode == http.StatusNotFound {
		return errors.New("could not do license prune, not supported on your appliance version")
	}
	if response.StatusCode != http.StatusNoContent {
		fmt.Fprintf(opts.Out, "Could not prune license got HTTP %d\n", response.StatusCode)
		return nil
	}
	fmt.Fprintln(opts.Out, "users license pruned")
	return nil
}
