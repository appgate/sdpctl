package appliance

import (
	"context"
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

type resolveNameStatusOpts struct {
	Config      *configuration.Config
	Out         io.Writer
	Client      func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance   func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug       bool
	json        bool
	applianceID string
}

const (
	resolveNameStatusLong = `
    Get the status of name resolution on a Gateway. It lists all the subscribed resource names from all the connected Clients and shows the resolution results.
    `
	resolveNameStatusExample = `
    # with a specific gateway appliance id:
    appliance resolve-name-status 7f340572-0cd3-416b-7755-9f5c4e546391 --json
    {
        "resolutions": {
          "aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev": {
            "partial": false,
            "finals": [
              "3.120.51.78",
              "35.156.237.184"
            ],
            "partials": [
              "dns://all.GW-ELB-2001535196.eu-central-1.elb.amazonaws.com",
              "dns://all.purple-lb-1785267452.eu-central-1.elb.amazonaws.com"
            ],
            "errors": []
          }
        }
    }
    `
)

func NewResolveNameStatusCmd(f *factory.Factory) *cobra.Command {
	opts := resolveNameStatusOpts{
		Config:    f.Config,
		Client:    f.APIClient,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:     "resolve-name-status [<appliance-id>]",
		Short:   `Get the status of name resolution on a Gateway.`,
		Long:    resolveNameStatusLong,
		Example: resolveNameStatusExample,
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
			return resolveNameStatusRun(c, args, &opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")

	return cmd
}

func resolveNameStatusRun(cmd *cobra.Command, args []string, opts *resolveNameStatusOpts) error {
	client, err := opts.Client(opts.Config)
	if err != nil {
		return err
	}
	token := opts.Config.GetBearTokenHeaderValue()
	ctx := context.Background()

	result, response, err := client.AppliancesApi.AppliancesIdNameResolutionStatusGet(ctx, opts.applianceID).Authorization(token).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if opts.json {
		return util.PrintJSON(opts.Out, result)
	}

	p := util.NewPrinter(opts.Out)
	p.AddHeader("Partial", "Finals", "Partials", "Errors")
	for _, r := range result.GetResolutions() {
		p.AddLine(r.GetPartial(), r.GetFinals(), r.GetPartials(), r.GetErrors())
	}
	p.Print()
	return nil
}
