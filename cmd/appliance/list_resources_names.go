package appliance

import (
	"context"
	"io"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type ResourcesNamesOpts struct {
	Config      *configuration.Config
	Out         io.Writer
	Client      func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance   func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug       bool
	json        bool
	siteID string
}

func ResourceNamesCmd(f *factory.Factory) *cobra.Command {
	opts := ResourcesNamesOpts{
		Config:    f.Config,
		Client:    f.APIClient,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var cmd = &cobra.Command{
		Use:     "resources",
		Short:   docs.SitesResourcesDocsList.Short,
		Long:    docs.SitesResourcesDocsList.Long,
		Example: docs.SitesResourcesDocsList.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			a, err := opts.Appliance(opts.Config)
			if err != nil {
				return err
			}
			ctx := context.Background()


			orderBy, err := cmd.Flags().GetStringSlice("order-by")
			if err != nil {
				return err
			}
			descending, err := cmd.Flags().GetBool("descending")
			if err != nil {
				return err
			}
			if len(args) != 1 {
				opts.siteID, err = appliancepkg.PromptSelect(ctx, a, nil, orderBy, descending)
				if err != nil {
					return err
				}
				return nil
			}

			// Validate UUID if the argument is applied
			uuidArg := args[0]
			_, err = uuid.Parse(uuidArg)
			if err != nil {
				log.WithField("error", err).Info("Invalid ID")
				if err != nil {
					return err
				}
			}
			opts.siteID = uuidArg

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return ResourcesNamesStatusRun(&opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	cmd.SetHelpFunc(cmdutil.HideIncludeExcludeFlags)

	return cmd
}

func ResourcesNamesStatusRun(opts *ResourcesNamesOpts) error {
	client, err := opts.Client(opts.Config)
	if err != nil {
		return err
	}
	
	
	//ctx := context.Background()
	ctx := util.BaseAuthContext(*opts.Config.BearerToken)
	//cfg := opts.Config
	//token, err := cfg.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	//ctx := util.BaseAuthContext(token)

	result, response, err := client.SitesApi.SitesIdResourcesGet(ctx, opts.siteID).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if opts.json {
		return util.PrintJSON(opts.Out, result)
	}

	p := util.NewPrinter(opts.Out, 4)

	p.AddHeader("Resource Name")
	for k := range result.Data {
		p.AddLine(k)
	}
	p.Print()
	return nil
}
