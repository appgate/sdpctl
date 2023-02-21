package appliance

import (
	"context"
	"fmt"
	"io"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type resolveNameOpts struct {
	Config       *configuration.Config
	Out          io.Writer
	Client       func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance    func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug        bool
	json         bool
	applianceID  string
	resourceName string
}

func NewResolveNameCmd(f *factory.Factory) *cobra.Command {
	opts := resolveNameOpts{
		Config:    f.Config,
		Client:    f.APIClient,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:     "resolve-name [<appliance-id>] [<query>]",
		Short:   docs.ApplianceResolveNameDoc.Short,
		Long:    docs.ApplianceResolveNameDoc.Long,
		Example: docs.ApplianceResolveNameDoc.ExampleString(),
		Args: func(cmd *cobra.Command, args []string) error {
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
			}
			ctx := context.Background()
			filter := map[string]map[string]string{
				"include": {
					"function": "gateway",
				},
			}
			orderBy, err := cmd.Flags().GetStringSlice("order-by")
			if err != nil {
				return err
			}
			descending, err := cmd.Flags().GetBool("descending")
			if err != nil {
				return err
			}
			switch len(args) {
			case 0:
				applianceID, err := appliancepkg.PromptSelect(ctx, a, filter, orderBy, descending)
				if err != nil {
					return err
				}
				opts.applianceID = applianceID
			case 1:
				if util.IsUUID(args[0]) {
					opts.applianceID = args[0]
				} else {
					applianceID, err := appliancepkg.PromptSelect(ctx, a, filter, orderBy, descending)
					if err != nil {
						return err
					}
					opts.applianceID = applianceID
					opts.resourceName = args[0]
				}
			case 2:
				if !util.IsUUID(args[0]) {
					return fmt.Errorf("%s is not a valid appliance UUID", args[0])
				}
				opts.applianceID = args[0]
				opts.resourceName = args[1]
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return resolveNameRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	cmd.SetHelpFunc(cmdutil.HideIncludeExcludeFlags)

	return cmd
}

func resolveNameRun(cmd *cobra.Command, args []string, opts *resolveNameOpts) error {
	client, err := opts.Client(opts.Config)
	if err != nil {
		return err
	}
	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}

	ctx := context.Background()
	body := openapi.AppliancesIdTestResolverNamePostRequest{
		ResourceName: openapi.PtrString(opts.resourceName),
	}
	result, response, err := client.AppliancesApi.AppliancesIdTestResolverNamePost(ctx, opts.applianceID).AppliancesIdTestResolverNamePostRequest(body).Authorization(token).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if opts.json {
		return util.PrintJSON(opts.Out, result)
	}
	for _, ip := range result.GetIps() {
		fmt.Fprintln(opts.Out, ip)
	}

	fmt.Fprintln(opts.Out, result.GetError())

	return nil
}
