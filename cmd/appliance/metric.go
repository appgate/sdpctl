package appliance

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/appgate/appgatectl/pkg/api"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
)

type metricOptions struct {
	Config      *configuration.Config
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
	}
	var cmd = &cobra.Command{
		Use:   "metric [<appliance-id>]",
		Short: `Get all the Prometheus metrics for the given Appliance`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(opts.applianceID) < 1 {
				return errors.New("--appliance-id is mandatory.")
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return metricRun(c, args, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.applianceID, "appliance-id", "", "appliance UUID")
	cmd.Flags().StringVar(&opts.metric, "metric-name", "", "Metric name")

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
