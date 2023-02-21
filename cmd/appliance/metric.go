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
		Use:     "metric [<appliance-id>] [<metric-name>]",
		Short:   docs.ApplianceMetricsDoc.Short,
		Long:    docs.ApplianceMetricsDoc.Long,
		Example: docs.ApplianceMetricsDoc.ExampleString(),
		Aliases: []string{"metrics"},
		Args: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
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
				applianceID, err := appliancepkg.PromptSelect(ctx, a, nil, orderBy, descending)
				if err != nil {
					return err
				}
				opts.applianceID = applianceID
			case 1:
				if util.IsUUID(args[0]) {
					opts.applianceID = args[0]
				} else {
					applianceID, err := appliancepkg.PromptSelect(ctx, a, nil, orderBy, descending)
					if err != nil {
						return err
					}
					opts.applianceID = applianceID
					opts.metric = args[0]
				}

			case 2:
				if !util.IsUUID(args[0]) {
					return fmt.Errorf("%s is not a valid appliance UUID", args[0])
				}
				opts.applianceID = args[0]
				opts.metric = args[1]
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return metricRun(c, args, &opts)
		},
	}
	cmd.SetHelpFunc(cmdutil.HideIncludeExcludeFlags)
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
	t, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
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
