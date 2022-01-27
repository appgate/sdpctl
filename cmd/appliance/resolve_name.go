package appliance

import (
	"context"
	"fmt"
	"io"

	"github.com/appgate/appgatectl/pkg/api"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

const resolveNameLong = `

Test a resolver name on a Gateway.

Name resolvers are used by the Gateways on a Site resolve the IPs in the specific network or set of protected resources.

Example:

# with a specific gateway appliance id:
appgatectl appliance resolve-name d750ad44-7c6a-416d-773b-f805a2272418 --resource-name dns://google.se


# If you omit appliance id, you will be prompted with all online gateways, and you can select one to test on.
> appgatectl appliance resolve-name --resource-name dns://google.se
? select appliance: gateway-9a9b8b70-faaa-4059-a061-761ce13783ba-site1 - Default Site - []
142.251.36.3
2a00:1450:400e:80f::2003

`

// NewResolveNameCmd return a new appliance list command
func NewResolveNameCmd(f *factory.Factory) *cobra.Command {
	opts := resolveNameOpts{
		Config:    f.Config,
		Client:    f.APIClient,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:   "resolve-name [<appliance-id>] --resolve-name=query",
		Short: `Test a resolver name on a Gateway`,
		Long:  resolveNameLong,
		Args: func(cmd *cobra.Command, args []string) error {
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
			}
			ctx := context.Background()
			filter := map[string]map[string]string{
				"filter": {
					"role": "gateway",
				},
			}
			if len(args) != 1 {
				opts.applianceID, err = prompt.SelectAppliance(ctx, a, filter)
				if err != nil {
					return err
				}
				return nil
			}

			// Validate UUID if the argument is applied
			uuidArg := args[0]
			_, err = uuid.Parse(uuidArg)
			if err != nil {
				log.WithField("error", err).Info("Invalid ID. Please select appliance instead")
				uuidArg, err = prompt.SelectAppliance(ctx, a, filter)
				if err != nil {
					return err
				}
			}
			opts.applianceID = uuidArg

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return resolveNameRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	cmd.Flags().StringVar(&opts.resourceName, "resource-name", "", "The resource name to test on the Gateway. (Required)")
	cmd.MarkFlagRequired("resource-name")

	return cmd
}

func resolveNameRun(cmd *cobra.Command, args []string, opts *resolveNameOpts) error {
	client, err := opts.Client(opts.Config)
	if err != nil {
		return err
	}
	token := opts.Config.GetBearTokenHeaderValue()

	ctx := context.Background()
	body := openapi.InlineObject4{
		ResourceName: openapi.PtrString(opts.resourceName),
	}
	result, response, err := client.AppliancesApi.AppliancesIdTestResolverNamePost(ctx, opts.applianceID).InlineObject4(body).Authorization(token).Execute()
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
