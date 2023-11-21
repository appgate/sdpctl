package functions

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	apipkg "github.com/appgate/sdpctl/pkg/api"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/terminal"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
)

var (
	logServerImages []string = []string{
		"cz-opensearch",
		"cz-opensearchdashboards",
	}
	funcImages map[string][]string = map[string][]string{
		appliancepkg.FunctionLogServer: logServerImages,
	}
)

type bundleOptions struct {
	registry *url.URL
	savePath string
	version  string
	ciMode   bool
	out      io.Writer
}

func NewApplianceFunctionsDownloadCmd(f *factory.Factory) *cobra.Command {
	opts := bundleOptions{
		out: f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "download [function...]",
		Aliases: []string{"dl", "bundle"},
		Annotations: map[string]string{
			configuration.SkipAuthCheck: "true",
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return ValidFuncs, cobra.ShellCompDirectiveNoFileComp
		},
		Short:   docs.ApplianceFunctionsDownloadDocs.Short,
		Long:    docs.ApplianceFunctionsDownloadDocs.Long,
		Example: docs.ApplianceFunctionsDownloadDocs.ExampleString(),
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), func(cmd *cobra.Command, args []string) error {
			registry, err := cmd.Flags().GetString("docker-registry")
			if err != nil {
				return err
			}
			if opts.registry, err = f.DockerRegistry(registry); err != nil {
				return err
			}
			if err := os.MkdirAll(opts.savePath, 0o700); err != nil {
				return err
			}
			if tag, err := cmd.Flags().GetString("version"); err == nil && len(tag) > 0 {
				if v, err := appliancepkg.ParseVersionString(tag); err == nil {
					opts.version, err = util.DockerTagVersion(v)
					if err != nil {
						return err
					}
				}
			}
			if opts.ciMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				return err
			}

			var errs *multierror.Error
			for _, a := range args {
				if util.InSlice(a, UnavailableFuncs) {
					errs = multierror.Append(errs, fmt.Errorf("Function not yet supported: '%s'", a))
					continue
				}
				if !util.InSlice(a, ValidFuncs) {
					errs = multierror.Append(errs, fmt.Errorf("Invalid function provided: '%s'", a))
				}
			}
			return errs.ErrorOrNil()
		}),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(opts.version) <= 0 {
				api, err := f.Appliance(f.Config)
				if err != nil {
					return err
				}
				ctx := context.Background()
				appliances, err := api.List(ctx, appliancepkg.DefaultCommandFilter, []string{"name"}, false)
				if err != nil {
					return err
				}
				configHostURL, err := url.ParseRequestURI(f.Config.URL)
				if err != nil {
					return err
				}
				primary, err := appliancepkg.FindPrimaryController(appliances, configHostURL.Hostname(), true)
				if err != nil {
					return err
				}
				logrus.WithField("primary-controller", primary.GetName()).Debug("found primary controller")
				stats, response, err := api.Stats(ctx, nil, []string{"name"}, false)
				if err != nil {
					return apipkg.HTTPErrorResponse(response, err)
				}
				primStats, err := util.Find(stats.GetData(), func(s openapi.StatsAppliancesListAllOfData) bool { return s.GetId() == primary.GetId() })
				if err != nil {
					return err
				}
				logrus.WithField("stats", primStats).Debug("found primary controller stats")
				primVersion, err := appliancepkg.ParseVersionString(primStats.GetVersion())
				if err != nil {
					return err
				}
				tag, err := util.DockerTagVersion(primVersion)
				if err != nil {
					return err
				}
				opts.version = tag
			}

			if v, err := version.NewVersion(opts.version); err == nil {
				x, _ := version.NewVersion("6.2.0")
				if x.GreaterThan(v) {
					return fmt.Errorf("unsupported version: %s, only available for version 6.2 or higher", opts.version)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			terminal.Lock()
			defer terminal.Unlock()
			client, err := f.HTTPClient()
			if err != nil {
				return err
			}

			var wg sync.WaitGroup
			errChan := make(chan error)
			ctx := context.Background()
			var p *tui.Progress
			if !opts.ciMode {
				p = tui.New(ctx, f.SpinnerOut)
			}
			for _, arg := range args {
				wg.Add(1)
				go func(ctx context.Context, wg *sync.WaitGroup, function string, opts bundleOptions, errs chan<- error, p *tui.Progress) {
					defer wg.Done()
					zipName := fmt.Sprintf("%s-%s.zip", strings.ToLower(function), opts.version)
					path := filepath.Join(opts.savePath, zipName)
					var t *tui.Tracker
					if p != nil {
						t = p.AddTracker(zipName, "downloading", "complete", mpb.BarRemoveOnComplete())
						go t.Watch([]string{"complete"}, []string{"failed"})
						defer t.Update("complete")
					}

					images := make(map[string]string, len(funcImages[function]))
					for _, f := range funcImages[function] {
						images[f] = opts.version
					}
					file, err := appliancepkg.DownloadDockerBundles(ctx, p, client, path, opts.registry, images, false)
					if err != nil {
						errs <- err
						os.Remove(file.Name())
						return
					}
					logrus.WithField("file", file.Name()).Info("bundle ready")
				}(ctx, &wg, arg, opts, errChan, p)
			}

			var errs *multierror.Error
			go func(errChan <-chan error) {
				for e := range errChan {
					errs = multierror.Append(errs, e)
				}
			}(errChan)

			wg.Wait()
			if p != nil {
				p.Wait()
			}
			if errs == nil {
				fmt.Fprintf(opts.out, "Download complete. Files saved to %s\n", opts.savePath)
			}
			return errs.ErrorOrNil()
		},
	}

	flags := cmd.Flags()
	flags.String("docker-registry", "", "docker registry for downloading image bundles")
	flags.StringVar(&opts.savePath, "save-path", filesystem.DownloadDir(), "path to where the container bundle should be saved")
	flags.StringVar(&opts.version, "version", "", "Override the LogServer version that will be downloaded. Defaults to the same version as the primary controller.")

	return cmd
}
