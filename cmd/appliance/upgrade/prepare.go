package upgrade

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"text/template"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	multierr "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
)

type prepareUpgradeOptions struct {
	Config        *configuration.Config
	Out           io.Writer
	Appliance     func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug         bool
	insecure      bool
	NoInteractive bool
	image         string
	DevKeyring    bool
	remoteImage   bool
	filename      string
	workers       int
	timeout       time.Duration
}

// NewPrepareUpgradeCmd return a new prepare upgrade command
func NewPrepareUpgradeCmd(f *factory.Factory) *cobra.Command {
	f.Config.Timeout = 300
	opts := &prepareUpgradeOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		timeout:   DefaultTimeout,
	}
	var prepareCmd = &cobra.Command{
		Use:   "prepare",
		Short: "prepare upgrade",
		Long: `Prepare an upgrade but do NOT install it.
This means the upgrade file will be downloaded/uploaded to all the appliances,
the signature verified as well as any other preconditions applicable at this point.

There are initial checks on the filename before attempting to upload it to the Appliances.
A valid filename ends with '.img.zip' and also needs to have a semver included somewhere
in the name, eg. 'upgrade.img.zip' will not not be valid, but 'upgrade5.5.3.img.zip' is
considered valid.

Note that the '--image' flag also accepts URL:s. The Appliances will then attempt to download
the upgrade image using the provided URL. It will fail if the Appliances cannot access the URL.`,
		Example: `$ sdpctl appliance upgrade prepare --image=/path/to/upgrade-5.5.3.img.zip`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(opts.image) < 1 {
				return errors.New("--image is mandatory")
			}
			workers, err := cmd.Flags().GetInt("throttle")
			if err != nil {
				errMsg := "Failed to parse throttle flag."
				log.WithError(err).Error(errMsg)
				return fmt.Errorf(errMsg)
			}
			if workers < 1 {
				errMsg := "Prepare failed: throttle too small"
				log.Error(errMsg)
				return fmt.Errorf("%s", errMsg)
			}
			opts.workers = workers

			minTimeout := 15 * time.Minute
			flagTimeout, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				return err
			}
			if flagTimeout < minTimeout {
				fmt.Printf("WARNING: timeout is less than the allowed minimum. Using default timeout instead: %s", opts.timeout)
			} else {
				opts.timeout = flagTimeout
			}
			var errs error
			opts.filename = path.Base(opts.image)
			if err := checkImageFilename(opts.filename); err != nil {
				errs = multierr.Append(errs, err)
			}

			// allow remote addr for image, such as aws s3 bucket
			if util.IsValidURL(opts.image) {
				opts.remoteImage = true
			}
			if !opts.remoteImage {
				// if the image is a local file, make sure its readable
				// make early return if not
				ok, err := util.FileExists(opts.image)
				if err != nil {
					return err
				}
				if !ok {
					errs = multierr.Append(errs, fmt.Errorf("Image file not found %q", opts.image))
				}
				if ok {
					_, err := zip.OpenReader(opts.image)
					if err != nil {
						errs = multierr.Append(errs, err)
					}
				}
			}

			return errs
		},
		RunE: func(c *cobra.Command, args []string) error {
			return prepareRun(c, args, opts)
		},
	}

	flags := prepareCmd.Flags()
	flags.BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	flags.BoolVar(&opts.NoInteractive, "no-interactive", false, "suppress interactive prompt with auto accept")
	flags.StringVarP(&opts.image, "image", "", "", "Upgrade image file or URL")
	flags.BoolVar(&opts.DevKeyring, "dev-keyring", true, "Use the development keyring to verify the upgrade image")
	flags.Int("throttle", 5, "Upgrade is done in batches using a throttle value. You can control the throttle using this flag.")

	return prepareCmd
}

func checkImageFilename(i string) error {
	// Check if its a valid filename
	if rg := regexp.MustCompile(`(.+)?\d\.\d\.\d(.+)?\.img\.zip`); !rg.MatchString(i) {
		return errors.New("Invalid mimetype on image file. The format is expected to be a .img.zip archive with a version number, such as 5.5.1")
	}
	return nil
}

const (
	fileInProgress = "InProgress"
	fileReady      = "Ready"
	fileFailed     = "Failed"
)

var ErrPrimaryControllerVersionErr = errors.New("version mismatch: run sdpctl configure signin")

func prepareRun(cmd *cobra.Command, args []string, opts *prepareUpgradeOptions) error {
	if appliancepkg.IsOnAppliance() {
		return cmdutil.ErrExecutedOnAppliance
	}

	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if a.UpgradeStatusWorker == nil {
		a.UpgradeStatusWorker = &appliancepkg.UpgradeStatus{
			Appliance: a,
		}
	}

	targetVersion, err := appliancepkg.GuessVersion(opts.filename)
	if err != nil {
		log.Debugf("Could not guess target version based on the image file name %q", opts.filename)
	}
	filter := util.ParseFilteringFlags(cmd.Flags())
	Allappliances, err := a.List(ctx, nil)
	if err != nil {
		return err
	}
	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}
	appliances := appliancepkg.FilterAppliances(Allappliances, filter)

	primaryController, err := appliancepkg.FindPrimaryController(Allappliances, host)
	if err != nil {
		return err
	}

	initialStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	if hasLowDiskSpace := appliancepkg.HasLowDiskSpace(initialStats.GetData()); len(hasLowDiskSpace) > 0 {
		appliancepkg.PrintDiskSpaceWarningMessage(opts.Out, hasLowDiskSpace)
		if !opts.NoInteractive {
			if err := prompt.AskConfirmation(); err != nil {
				return err
			}
		}
	}
	autoScalingWarning := false
	if targetVersion != nil {
		constraints, _ := version.NewConstraint(">= 5.5.0")
		if constraints.Check(targetVersion) {
			autoScalingWarning = true
		}
	} else if opts.Config.Version == 15 {
		autoScalingWarning = true
	}
	if t, gws := appliancepkg.AutoscalingGateways(appliances); autoScalingWarning && len(gws) > 0 && !opts.NoInteractive {
		msg, err := appliancepkg.ShowAutoscalingWarningMessage(t, gws)
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", msg)
		if err := prompt.AskConfirmation("Have you disabled the health check on those auto-scaled gateways"); err != nil {
			return err
		}
	}
	groups := appliancepkg.GroupByFunctions(appliances)
	targetPeers := append(groups[appliancepkg.FunctionController], groups[appliancepkg.FunctionLogServer]...)
	peerAppliances := appliancepkg.WithAdminOnPeerInterface(targetPeers)
	if len(peerAppliances) > 0 && !opts.NoInteractive {
		msg, err := appliancepkg.ShowPeerInterfaceWarningMessage(peerAppliances)
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", msg)
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	currentPrimaryControllerVersion, err := appliancepkg.GetApplianceVersion(*primaryController, initialStats)
	if err != nil {
		return err
	}
	// if we have an existing config with the primary controller version, check if we need to re-authenticate
	// before we continue with the upgrade to update the peer API version.
	if len(opts.Config.PrimaryControllerVersion) > 0 {
		preV, err := version.NewVersion(opts.Config.PrimaryControllerVersion)
		if err != nil {
			return fmt.Errorf("%s %w", ErrPrimaryControllerVersionErr, err)
		}
		if !preV.Equal(currentPrimaryControllerVersion) {
			return ErrPrimaryControllerVersionErr
		}
	}

	log.Infof("Primary controller is: %s and running %s", primaryController.Name, currentPrimaryControllerVersion.String())
	if targetVersion != nil {
		log.Infof("Appliances will be prepared for upgrade to version: %s", targetVersion.String())
	}
	msg, err := showPrepareUpgradeMessage(opts.filename, appliances, initialStats.GetData())
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", msg)
	if !opts.NoInteractive {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	// Step 1
	shouldUpload := false
	fileStatusCtx, fileStatusCancel := context.WithTimeout(ctx, opts.timeout)
	defer fileStatusCancel()
	existingFile, err := a.FileStatus(fileStatusCtx, opts.filename)
	if err != nil {
		// if we dont get 404, return err
		if errors.Is(err, appliancepkg.ErrFileNotFound) {
			shouldUpload = true
		} else {
			return err
		}
	}
	if !shouldUpload && existingFile.GetStatus() != fileReady {
		log.WithField("file", opts.filename).Infof("Remote file already exist, but is in status %s, overriding it", existingFile.GetStatus())
		shouldUpload = true
	}
	if existingFile.GetStatus() == fileReady {
		log.WithField("file", existingFile.GetName()).Info("File already exists, using it as is")
	}
	if shouldUpload && !opts.remoteImage {
		imageFile, err := os.Open(opts.image)
		if err != nil {
			return err
		}
		defer imageFile.Close()
		if err := a.UploadFile(ctx, imageFile); err != nil {
			return err
		}

		remoteFile, err := a.FileStatus(ctx, opts.filename)
		if err != nil {
			return err
		}
		if remoteFile.GetStatus() != fileReady {
			return fmt.Errorf("remote file %q is uploaded, but is in status %s", opts.filename, existingFile.GetStatus())
		}
		log.WithField("file", remoteFile.GetName()).Infof("Status %s", remoteFile.GetStatus())
	}

	// Step 2
	remoteFilePath := fmt.Sprintf("controller://%s/%s", primaryController.GetHostname(), opts.filename)
	// NOTE: Backwards compatibility with appliances older than API version 13.
	// Appliances before API version require that the peer port be passed explicitly as part of the download URL.
	// Insert the peer port into the URL if necessary.
	if opts.Config.Version < 13 {
		if v, ok := primaryController.GetPeerInterfaceOk(); ok {
			remoteFilePath = fmt.Sprintf("controller://%s:%d/%s", primaryController.GetHostname(), int(v.GetHttpsPort()), opts.filename)
		}
	}

	if opts.remoteImage {
		remoteFilePath = opts.image
	}
	// prepare the image on the appliances,
	// its throttle based on nWorkers to reduce internal rate limit if we try to download from too many appliances at once.
	prepare := func(ctx context.Context, remoteFilePath string, appliances []openapi.Appliance) ([]openapi.Appliance, error) {
		log.Infof("Remote file path for controller %s", remoteFilePath)
		g, _ := errgroup.WithContext(ctx)

		applianceIds := make(chan openapi.Appliance, len(appliances))
		// Produce, send all appliance Id to the Channel so we can consume them in a fixed rate.
		g.Go(func() error {
			for _, appliance := range appliances {
				applianceIds <- appliance
			}
			close(applianceIds)
			return nil
		})

		p := mpb.New(mpb.WithOutput(opts.Out))
		// consume Prepare with nWorkers
		finished := make(chan openapi.Appliance)
		for i := 0; i < opts.workers; i++ {
			g.Go(func() error {
				for appliance := range applianceIds {
					appCtx, appCancel := context.WithTimeout(context.Background(), opts.timeout)
					fields := log.Fields{"appliance": appliance.GetName()}
					log.WithFields(fields).Info("Preparing upgrade")
					statusReport := make(chan string)
					a.UpgradeStatusWorker.Watch(ctx, p, appliance, appliancepkg.UpgradeStatusReady, statusReport)
					if err := a.PrepareFileOn(appCtx, remoteFilePath, appliance.GetId(), opts.DevKeyring); err != nil {
						appCancel()
						log.WithFields(fields).WithError(err).WithContext(appCtx).Error(err)
						return err
					}
					if err := a.UpgradeStatusWorker.Wait(ctx, appliance, appliancepkg.UpgradeStatusDownloading, statusReport); err != nil {
						appCancel()
						return err
					}
					if err := a.UpgradeStatusWorker.Wait(ctx, appliance, appliancepkg.UpgradeStatusVerifying, statusReport); err != nil {
						appCancel()
						return err
					}
					if err := a.UpgradeStatusWorker.Wait(ctx, appliance, appliancepkg.UpgradeStatusReady, statusReport); err != nil {
						appCancel()
						return err
					}
					select {
					case finished <- appliance:
					case <-appCtx.Done():
						appCancel()
						return appCtx.Err()
					}
					close(statusReport)
					appCancel()
				}

				return nil
			})
		}

		go func() {
			g.Wait()
			close(finished)
		}()

		r := make([]openapi.Appliance, 0)
		for appliance := range finished {
			r = append(r, appliance)
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
		p.Wait()
		return r, nil
	}

	fmt.Fprint(opts.Out, "\nPreparing image on appliances:\n")
	if _, err := prepare(ctx, remoteFilePath, appliances); err != nil {
		return err
	}

	if !opts.remoteImage {
		// Step 3
		log.Infof("3. Delete upgrade image %s from Controller", opts.filename)
		deleteCtx, deleteCancel := context.WithTimeout(ctx, opts.timeout)
		defer deleteCancel()
		if err := a.DeleteFile(deleteCtx, opts.filename); err != nil {
			log.Warnf("Failed to delete %s from controller %s", opts.filename, err)
		}
		log.Infof("File %s deleted from Controller", opts.filename)
	}
	return nil
}

const prepareUpgradeMessage = `
1. Upload upgrade image {{.Filepath}} to Controller
2. Prepare upgrade on the following appliances:
{{range .Appliances }}
  - Current Version: {{.CurrentVersion }}{{"\t"}}{{.Online -}}{{"\t"}} {{.Name -}}
{{end}}

3. Delete upgrade image from Controller
`

func showPrepareUpgradeMessage(f string, appliance []openapi.Appliance, stats []openapi.StatsAppliancesListAllOfData) (string, error) {
	type applianceData struct {
		Name           string
		CurrentVersion string
		Online         string
	}
	type stub struct {
		Filepath   string
		Appliances []applianceData
	}
	data := stub{Filepath: f}
	for _, a := range appliance {
		for _, stat := range stats {
			if a.GetId() == stat.GetId() {
				i := applianceData{
					Name:           a.GetName(),
					CurrentVersion: stat.GetVersion(),
					Online:         "Offline ⨯",
				}

				if stat.GetOnline() {
					i.Online = "Online ✓"
				}
				data.Appliances = append(data.Appliances, i)
			}
		}
	}

	t := template.Must(template.New("").Parse(prepareUpgradeMessage))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
