package license

import (
	"fmt"
	"net/http"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/spf13/cobra"
)

// NewPruneCmd return a new license prune subcommand
func NewPruneCmd(opts *licenseOpts) *cobra.Command {
	return &cobra.Command{
		Use: "prune",
		Annotations: map[string]string{
			"MinAPIVersion": "18",
			"ErrorMessage":  "sdpctl license prune requires appliance version higher or equal to 6.1 with API Version 18",
		},
		Short:   docs.LicensePruneDoc.Short,
		Long:    docs.LicensePruneDoc.Long,
		Example: docs.LicensePruneDoc.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			return pruneRun(c, args, opts)
		},
		Hidden: true,
	}
}

func pruneRun(cmd *cobra.Command, args []string, opts *licenseOpts) error {
	client, err := opts.HTTPClient()
	if err != nil {
		return cmdutil.GenericErrorWrap("failed to resolve HTTP client based on your configuration", err)
	}
	requestURL := fmt.Sprintf("%s/license/users/prune", opts.BaseURL)
	request, err := http.NewRequest(http.MethodDelete, requestURL, nil)
	if err != nil {
		return cmdutil.GenericErrorWrap("failed to prune license", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if response.StatusCode == http.StatusNotFound {
		return cmdutil.GenericErrorWrap("failed to prune license", cmdutil.ErrUnsupportedOperation)
	}
	if response.StatusCode != http.StatusNoContent {
		return cmdutil.GenericErrorWrap("failed to prune license", cmdutil.ErrUnexpectedResponseStatus(http.StatusNoContent, response.StatusCode))
	}
	fmt.Fprintln(opts.Out, "User licenses pruned")
	return nil
}
