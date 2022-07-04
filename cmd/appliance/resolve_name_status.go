package appliance

import (
	"context"
	"io"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
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

type resolveNameStatusOpts struct {
	Config      *configuration.Config
	Out         io.Writer
	Client      func(c *configuration.Config) (*openapi.APIClient, error)
	Appliance   func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug       bool
	json        bool
	applianceID string
}

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
		Short:   docs.ApplianceResolveNameStatusDoc.Short,
		Long:    docs.ApplianceResolveNameStatusDoc.Long,
		Example: docs.ApplianceResolveNameStatusDoc.ExampleString(),
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
			if len(args) != 1 {
				opts.applianceID, err = appliancepkg.PromptSelect(ctx, a, filter)
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
				uuidArg, err = appliancepkg.PromptSelect(ctx, a, filter)
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
	cmd.SetHelpFunc(cmdutil.HideIncludeExcludeFlags)

	return cmd
}

func resolveNameStatusRun(cmd *cobra.Command, args []string, opts *resolveNameStatusOpts) error {
	client, err := opts.Client(opts.Config)
	if err != nil {
		return err
	}
	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ctx := context.Background()

	result, response, err := client.AppliancesApi.AppliancesIdNameResolutionStatusGet(ctx, opts.applianceID).Authorization(token).Execute()
	if err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if opts.json {
		return util.PrintJSON(opts.Out, result)
	}

	p := util.NewPrinter(opts.Out, 4)
	p.AddHeader("Partial", "Finals", "Partials", "Errors")
	for _, r := range result.GetResolutions() {
		p.AddLine(r.GetPartial(), r.GetFinals(), r.GetPartials(), r.GetErrors())
	}
	p.Print()
	return nil
}
