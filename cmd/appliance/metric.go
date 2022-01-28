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
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type metricOptions struct {
	Config      *configuration.Config
	Appliance   func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Out         io.Writer
	APIClient   func(c *configuration.Config) (*openapi.APIClient, error)
	debug       bool
	applianceID string
	metric      string
}

// NewMetricCmd return a new appliance metric command
func NewMetricCmd(f *factory.Factory) *cobra.Command {
	opts := metricOptions{
		Config:    f.Config,
		APIClient: f.APIClient,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		Appliance: f.Appliance,
	}
	var cmd = &cobra.Command{
		Use:   "metric [<appliance-id>]",
		Short: `Get all the Prometheus metrics for the given Appgate SDP Appliance`,
		Long: `The 'metric' command will return a list of all the available metrics provided by an Appgate SDP Appliance for use in Prometheus.
If no Appliance ID is given as an argument, the command will prompt for which Appliance you want metrics for. The '--metric-name' flag can be used
to get a specific metric name. This needs to be an exact match.

NOTE: Although the '--filter' and '--exclude' flags are provided as options here, they don't have any actual effect on the command.`,
		Example: `appgatectl appliance metric
appgatectl appliance metric <appliance-id>
appgatectl appliance metric <appliance-id> --metric-name=<some_metric_name>`,
		Aliases: []string{"metrics"},
		Args: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
			}
			if len(args) != 1 {
				opts.applianceID, err = prompt.SelectAppliance(ctx, a, nil)
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
				uuidArg, err = prompt.SelectAppliance(ctx, a, nil)
				if err != nil {
					return err
				}
			}
			opts.applianceID = uuidArg

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return metricRun(c, args, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.metric, "metric-name", "", "Query for a specific metric by name (exact match)")

	return cmd
}

func metricRun(cmd *cobra.Command, args []string, opts *metricOptions) error {
	client, err := opts.APIClient(opts.Config)
	if err != nil {
		return err
	}
	ctx := context.WithValue(
		context.Background(),
		openapi.ContextAcceptHeader,
		fmt.Sprintf("application/vnd.appgate.peer-v%d+text", opts.Config.Version),
	)
	t := opts.Config.GetBearTokenHeaderValue()
	if len(opts.metric) > 0 {
		data, response, err := client.ApplianceMetricsApi.AppliancesIdMetricsNameGet(ctx, opts.applianceID, opts.metric).Authorization(t).Execute()
		if err != nil {
			return api.HTTPErrorResponse(response, err)
		}
		fmt.Fprintln(opts.Out, data)
		return nil
	}
	data, response, err := client.ApplianceMetricsApi.AppliancesIdMetricsGet(ctx, opts.applianceID).Authorization(t).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}

	fmt.Fprintln(opts.Out, data)
	return nil
}
