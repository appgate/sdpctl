package appliance

import (
	"context"
	"fmt"
	"io"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/appgatectl/pkg/api"
	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
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
		Use:     "metric [<appliance-id>]",
		Short:   `Get all the Prometheus metrics for the given Appliance`,
		Aliases: []string{"metrics"},
		Args: func(cmd *cobra.Command, args []string) error {
			var err error
			if len(args) != 1 {
				opts.applianceID, err = promptForAppliance(opts)
				if err != nil {
					return err
				}
			} else {
				// Validate UUID if the argument is applied
				uuidArg := args[0]
				_, err := uuid.Parse(uuidArg)
				if err != nil {
					log.WithField("error", err).Info("Invalid ID. Please select appliance instead")
					uuidArg, err = promptForAppliance(opts)
					if err != nil {
						return err
					}
				}
				opts.applianceID = uuidArg
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return metricRun(c, args, &opts)
		},
	}

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

func promptForAppliance(opts metricOptions) (string, error) {
	// Command accepts only one argument
	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	appliances, err := a.List(ctx, nil)
	if err != nil {
		return "", err
	}

	names := []string{}
	for _, a := range appliances {
		names = append(names, a.GetName())
	}
	qs := &survey.Select{
		PageSize: len(appliances),
		Message:  "select appliance:",
		Options:  names,
	}
	selectedIndex := 0
	survey.AskOne(qs, &selectedIndex)
	appliance := appliances[selectedIndex]
	return appliance.GetId(), nil
}
