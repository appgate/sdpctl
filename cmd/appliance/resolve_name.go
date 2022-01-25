package appliance

import (
	"context"
	"errors"
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

func (o resolveNameOpts) ResolveAppliance(c *configuration.Config) (*appliancepkg.Appliance, error) {
	return o.Appliance(c)
}

// NewResolveNameCmd return a new appliance list command
func NewResolveNameCmd(f *factory.Factory) *cobra.Command {
	opts := resolveNameOpts{
		Config:    f.Config,
		Client:    f.APIClient,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var listCmd = &cobra.Command{
		Use:   "resolve-name [<appliance-id>] --resolve-name=query",
		Short: `Test a resolver name on a Gateway`,
		Args: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var err error
			filter := map[string]map[string]string{
				"filter": {
					"role": "gateway",
				},
			}
			if len(args) != 1 {
				opts.applianceID, err = prompt.SelectAppliance(ctx, opts, opts.Config, filter)
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
				uuidArg, err = prompt.SelectAppliance(ctx, opts, opts.Config, filter)
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
	listCmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	listCmd.Flags().StringVar(&opts.applianceID, "appliance-id", "", "appliance UUID")
	listCmd.Flags().StringVar(&opts.resourceName, "resource-name", "", "The resource name to test on the Gateway.")

	return listCmd
}

func resolveNameRun(cmd *cobra.Command, args []string, opts *resolveNameOpts) error {
	if len(opts.resourceName) < 1 {
		return errors.New("--resource-name flag is mandatory")
	}
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
