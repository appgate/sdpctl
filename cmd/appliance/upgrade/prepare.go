package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/files"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/network"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/queue"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/google/uuid"
	multierr "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type prepareUpgradeOptions struct {
	Config              *configuration.Config
	Out                 io.Writer
	Appliance           func(c *configuration.Config) (*appliancepkg.Appliance, error)
	HTTPClient          func() (*http.Client, error)
	SpinnerOut          func() io.Writer
	debug               bool
	NoInteractive       bool
	image               string
	DevKeyring          bool
	remoteImage         bool
	remoteBundle        bool
	filename            string
	timeout             time.Duration
	defaultFilter       map[string]map[string]string
	hostOnController    bool
	forcePrepare        bool
	ciMode              bool
	actualHostname      string
	targetVersion       *version.Version
	dockerRegistry      *url.URL
	skipBundle          bool
	logServerBundlePath string
}

// NewPrepareUpgradeCmd return a new prepare upgrade command
func NewPrepareUpgradeCmd(f *factory.Factory) *cobra.Command {
	opts := &prepareUpgradeOptions{
		Config:     f.Config,
		Appliance:  f.Appliance,
		HTTPClient: f.HTTPClient,
		debug:      f.Config.Debug,
		Out:        f.IOOutWriter,
		SpinnerOut: f.GetSpinnerOutput(),
		timeout:    DefaultTimeout,
		defaultFilter: map[string]map[string]string{
			"include": {},
			"exclude": {
				"active": "false",
			},
		},
	}
	var prepareCmd = &cobra.Command{
		Use:     "prepare",
		Short:   docs.ApplianceUpgradePrepareDoc.Short,
		Long:    docs.ApplianceUpgradePrepareDoc.Long,
		Example: docs.ApplianceUpgradePrepareDoc.ExampleString(),
		Annotations: map[string]string{
			configuration.NeedUpdateAPIConfig: "true",
		},
		Args: cobra.ExactArgs(0),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(opts.image) < 1 {
				return errors.New("--image is mandatory")
			}
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

			var errs *multierr.Error
			opts.filename = filepath.Base(opts.image)
			if err := checkImageFilename(opts.filename); err != nil {
				return err
			}

			if opts.skipBundle, err = cmd.Flags().GetBool("skip-container-bundle"); err != nil {
				return err
			}

			// Get the docker registry address
			flagRegistry, err := cmd.Flags().GetString("docker-registry")
			if err != nil {
				return err
			}
			if opts.dockerRegistry, err = f.DockerRegistry(flagRegistry); err != nil {
				return err
			}
			log.WithField("URL", opts.dockerRegistry).Debug("found docker registry address")
			// allow remote addr for image, such as aws s3 bucket
			err = util.IsValidURL(opts.image)
			if err != nil && errors.Is(err, util.ErrMalformedURL) {
				return err
			}
			if err == nil {
				opts.remoteImage = true
				// if the file is a remote image URL, derive the filename from
				// standard lib 'path' instead of 'filepath' to avoid trailing URI elements

				// we can skip error check here since we already validated that its a url
				u, _ := url.Parse(opts.image)
				// remove any query string, and leave us only with the filename
				u.RawQuery = ""
				opts.filename = path.Base(u.String())

				// guess version from filename
				if opts.targetVersion, err = appliancepkg.ParseVersionString(opts.filename); err != nil {
					log.WithField("filename", opts.filename).Debug("Failed to guess version from filename")
					return err
				}
			}
			if !opts.remoteImage {
				// Needed to avoid trouble with the '~' symbol
				opts.image = filesystem.AbsolutePath(opts.image)

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
					// get version from metadata first. guess from filename if that fails
					// fatal if both fail
					if opts.targetVersion, err = appliancepkg.ParseVersionFromZip(opts.image); err != nil {
						var e error
						if opts.targetVersion, e = appliancepkg.ParseVersionString(opts.filename); e != nil {
							errs = multierr.Append(errs, err, e)
						}
					}
				}
			}

			opts.remoteBundle = false
			if len(opts.logServerBundlePath) > 0 {
				var bundlePath string
				parsedURL, err := url.ParseRequestURI(opts.logServerBundlePath)
				if err == nil && parsedURL.Scheme == "https" {
					// Download the bundle to a temp file
					client, err := opts.HTTPClient()
					if err != nil {
						return fmt.Errorf("failed to get HTTP client: %w", err)
					}

					fmt.Fprintf(opts.Out, "Downloading LogServer bundle from %s...\n", opts.logServerBundlePath)

					resp, err := client.Get(opts.logServerBundlePath)
					if err != nil {
						return fmt.Errorf("failed to download LogServer bundle from URL: %w", err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						return fmt.Errorf("failed to download LogServer bundle: HTTP %d", resp.StatusCode)
					}

					// Create progress indicator for download
					progress := tui.New(context.Background(), opts.SpinnerOut())
					defer progress.Wait()

					// Get file size from Content-Length header if available
					fileSize := resp.ContentLength
					var reader io.Reader = resp.Body

					if fileSize > 0 {
						// Create a progress bar for the download
						reader = progress.FileDownloadProgress("LogServer bundle", "✓ Download complete", fileSize, 40, resp.Body)
					}

					id := uuid.New().String()

					tmpFile, err := os.CreateTemp("", id)
					opts.remoteBundle = true
					if err != nil {
						return fmt.Errorf("failed to create temp file for LogServer bundle: %w", err)
					}
					defer tmpFile.Close()
					_, err = io.Copy(tmpFile, reader)
					if err != nil {
						return fmt.Errorf("failed to write LogServer bundle to temp file: %w", err)
					}

					progress.Complete()
					fmt.Fprint(opts.Out, "LogServer bundle downloaded successfully\n")
					bundlePath = tmpFile.Name()
					opts.logServerBundlePath = bundlePath
				}

				if err == nil && parsedURL.Scheme == "http" {
					return fmt.Errorf("Plain HTTP URLs are not supported for LogServer bundle. Please use HTTPS URL instead")
				}

				opts.logServerBundlePath = filesystem.AbsolutePath(opts.logServerBundlePath)
				ok, err := util.FileExists(opts.logServerBundlePath)
				if err != nil {
					return err
				}
				if !ok {
					errs = multierr.Append(errs, fmt.Errorf("LogServer image bundle not found: %q", opts.logServerBundlePath))
				}
				info, err := os.Stat(opts.logServerBundlePath)
				if err != nil {
					return err
				}
				if info.IsDir() {
					errs = multierr.Append(errs, fmt.Errorf("invalid LogServer bundle: bundle is a directory"))
				}
			}

			if opts.ciMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				return err
			}
			return errs.ErrorOrNil()
		},
		RunE: func(c *cobra.Command, args []string) error {
			h, err := opts.Config.GetHost()
			if err != nil {
				return fmt.Errorf("Could not determine hostname for %s", err)
			}
			if err := network.ValidateHostnameUniqueness(h); err != nil {
				return err
			}
			return prepareRun(c, opts)
		},
	}

	flags := prepareCmd.Flags()
	flags.BoolVar(&opts.NoInteractive, "no-interactive", false, "Suppress interactive prompt with auto accept")
	flags.StringVarP(&opts.image, "image", "", "", "Upgrade image file or URL")
	flags.BoolVar(&opts.DevKeyring, "dev-keyring", false, "Use the development keyring to verify the upgrade image")
	flags.Int("throttle", 5, "Upgrade is done in batches using a throttle value. You can control the throttle using this flag")
	flags.BoolVar(&opts.hostOnController, "host-on-controller", false, "Use the primary Controller as image host when uploading from remote source")
	flags.StringVar(&opts.actualHostname, "actual-hostname", "", "If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname")
	flags.BoolVar(&opts.forcePrepare, "force", false, "Force prepare of upgrade on appliances even though the version uploaded is the same or lower than the version already running on the appliance")
	flags.BoolVar(&opts.skipBundle, "skip-container-bundle", false, "skip the bundling of the docker images for functions that need them, e.g. the LogServer")
	flags.String("docker-registry", "", "Custom docker registry for downloading function docker images. Needs to be accessible by the sdpctl host machine.")
	flags.StringVar(&opts.logServerBundlePath, "logserver-bundle", "", "URL or local file path to a LogServer image bundle file to upload and use when upgrading a LogServer appliance.")

	return prepareCmd
}

func checkImageFilename(i string) error {
	// Check if its a valid filename
	if rg := regexp.MustCompile(`\.img\.zip`); !rg.MatchString(i) {
		return errors.New("Invalid name on image file. The format is expected to be a .img.zip archive")
	}
	return nil
}

func prepareRun(cmd *cobra.Command, opts *prepareUpgradeOptions) error {
	fmt.Fprintf(opts.Out, "sdpctl_version: %s\n\n", cmd.Root().Version)
	if appliancepkg.IsOnAppliance() {
		return cmdutil.ErrExecutedOnAppliance
	}
	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	spinnerOut := opts.SpinnerOut()
	ctx, cancel := context.WithCancel(util.BaseAuthContext(a.Token))
	defer cancel()
	ctx = context.WithValue(ctx, appliancepkg.Caller, cmd.CalledAs())
	if a.UpgradeStatusWorker == nil {
		a.UpgradeStatusWorker = &appliancepkg.UpgradeStatus{
			Appliance: a,
		}
	}
	if a.ApplianceStats == nil {
		a.ApplianceStats = &appliancepkg.ApplianceStatus{
			Appliance: a,
		}
	}

	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	Allappliances, err := a.List(ctx, nil, orderBy, descending)
	if err != nil {
		return err
	}
	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}

	token, err := opts.Config.GetBearTokenHeaderValue()
	if err != nil {
		return err
	}
	ac := change.ApplianceChange{
		APIClient: a.APIClient,
		Token:     token,
	}

	if len(opts.actualHostname) > 0 {
		host = opts.actualHostname
	}

	primaryController, err := appliancepkg.FindPrimaryController(Allappliances, host, true)
	if err != nil {
		return err
	}

	initialStats, _, err := a.ApplianceStatus(ctx, nil, orderBy, descending)
	if err != nil {
		return err
	}
	skipAppliances := []appliancepkg.SkipUpgrade{}
	online, offline, _ := appliancepkg.FilterAvailable(Allappliances, initialStats.GetData())
	for _, a := range offline {
		skipAppliances = append(skipAppliances, appliancepkg.SkipUpgrade{
			Reason:    appliancepkg.ErrSkipReasonOffline,
			Appliance: a,
		})
	}
	appliances, filtered, err := appliancepkg.FilterAppliances(online, filter, orderBy, descending)
	if err != nil {
		return err
	}
	for _, f := range filtered {
		skipAppliances = append(skipAppliances, appliancepkg.SkipUpgrade{
			Appliance: f,
			Reason:    appliancepkg.ErrSkipReasonFiltered,
		})
	}

	hasLowDiskSpace := appliancepkg.HasLowDiskSpace(initialStats.GetData())
	if opts.Config.Version <= 13 {
		// Versions before v13 does not have dev-keyring functionality
		opts.DevKeyring = false
	}
	if t, gws := appliancepkg.AutoscalingGateways(appliances); len(gws) > 0 {
		if !opts.NoInteractive {
			msg, err := appliancepkg.ShowAutoscalingWarningMessage(t, gws)
			if err != nil {
				return err
			}
			fmt.Fprintf(opts.Out, "\n%s\n", msg)
			if err := prompt.AskConfirmation("WARNING: Auto-scaled gateways will be excluded from the upgrade."); err != nil {
				return err
			}
		}
		// Remove auto-scaled gateways from appliances to be upgraded
		log.Info("excluding autoscaling gateways")
		appliancesForUpgrade := make([]openapi.Appliance, 0)
		for _, appliance := range appliances {
			isAutoscaling := false
			for _, gw := range gws {
				if appliance.Id == gw.Id {
					isAutoscaling = true
				}
			}
			if !isAutoscaling {
				appliancesForUpgrade = append(appliancesForUpgrade, appliance)
			}
		}
		appliances = appliancesForUpgrade
	}
	groups := appliancepkg.GroupByFunctions(appliances)

	upgradeStatuses, err := a.UpgradeStatusMap(ctx, appliances)
	if err != nil {
		return err
	}
	if !opts.forcePrepare {
		var skip []appliancepkg.SkipUpgrade
		appliances, skip = appliancepkg.CheckVersions(ctx, *initialStats, appliances, opts.targetVersion)
		skipAppliances = append(skipAppliances, skip...)

		postPrepared := []openapi.Appliance{}
		for _, app := range appliances {
			us := upgradeStatuses[app.GetId()]
			if us.Status != appliancepkg.UpgradeStatusReady {
				postPrepared = append(postPrepared, app)
				continue
			}
			prepareVersion, err := appliancepkg.ParseVersionString(us.Details)
			if err != nil {
				postPrepared = append(postPrepared, app)
				continue
			}
			if res, err := appliancepkg.CompareVersionsAndBuildNumber(opts.targetVersion, prepareVersion); err == nil && res >= 0 {
				skipAppliances = append(skipAppliances, appliancepkg.SkipUpgrade{
					Appliance: app,
					Reason:    appliancepkg.ErrSkipReasonAlreadyPrepared,
				})
				continue
			}
			postPrepared = append(postPrepared, app)
		}
		appliances = postPrepared

		if len(appliances) <= 0 {
			var errs *multierr.Error
			errs = multierr.Append(errs, cmdutil.ErrNothingToPrepare)
			if len(skipAppliances) > 0 {
				for _, skip := range skipAppliances {
					errs = multierr.Append(errs, skip)
				}
			}
			return errs
		}
	}

	currentPrimaryControllerVersion, err := appliancepkg.GetApplianceVersion(*primaryController, *initialStats)
	if err != nil {
		return err
	}
	majorOrMinorUpgrade := appliancepkg.IsMajorUpgrade(currentPrimaryControllerVersion, opts.targetVersion) || appliancepkg.IsMinorUpgrade(currentPrimaryControllerVersion, opts.targetVersion)
	ctrlUpgradeWarning, err := appliancepkg.NeedsMultiControllerUpgrade(upgradeStatuses, initialStats.GetData(), Allappliances, appliances, majorOrMinorUpgrade)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"primary-controller": primaryController.GetName(),
		"version":            currentPrimaryControllerVersion.String(),
		"target-version":     opts.targetVersion.String(),
	}).Debug("upgrade version information")

	// Check if we need to bundle docker image and upload as well
	v62, err := version.NewVersion("6.2.0-alpha")
	if err != nil {
		return err
	}
	logserverbundleupload := opts.targetVersion.GreaterThanOrEqual(v62) && len(groups[appliancepkg.FunctionLogServer]) > 0 && !opts.skipBundle

	upgradeNames := []string{}
	skipNames := []string{}
	for _, app := range appliances {
		upgradeNames = append(upgradeNames, app.GetName())
	}
	for _, app := range skipAppliances {
		skipNames = append(upgradeNames, app.Appliance.GetName())
	}
	log.WithFields(log.Fields{
		"upgrading":       strings.Join(upgradeNames, ", "),
		"skipping":        strings.Join(skipNames, ", "),
		"target_version":  opts.targetVersion,
		"current_version": currentPrimaryControllerVersion,
	}).Info("upgrade information")

	msg, err := showPrepareUpgradeMessage(opts.filename, opts.targetVersion, appliances, skipAppliances, initialStats.GetData(), ctrlUpgradeWarning, logserverbundleupload)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", msg)
	if !opts.NoInteractive {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	fm := files.NewFileManager(a, nil)
	if !opts.ciMode {
		fm.Progress = tui.New(ctx, opts.SpinnerOut())
	}
	// Step 1
	if logserverbundleupload {
		// Download image bundle and zip it up
		client, err := opts.HTTPClient()
		if err != nil {
			return err
		}

		tagVersion, err := util.DockerTagVersion(opts.targetVersion)
		if err != nil {
			return err
		}
		logServerZipName := fmt.Sprintf("logserver-%s.zip", tagVersion)

		// check if already exists
		// we don't know the exact name, so we match with all existing files in the repository
		// will match if major and minor versions are the same in the filename
		exists := false
		files, err := a.ListFiles(ctx, nil, false)
		if err != nil {
			return err
		}
		for _, f := range files {
			lfFileName := fmt.Sprintf("logserver-%s", tagVersion)
			if strings.HasPrefix(*f.Name, lfFileName) && strings.HasSuffix(*f.Name, ".zip") {
				exists = true
			}
		}
		if !exists {
			lsImageMsg := fmt.Sprintf("Preparing image bundle for LogServer %s", tagVersion)
			var zip *os.File
			if len(opts.logServerBundlePath) <= 0 {
				logServerImages := map[string]string{
					"cz-opensearch":           tagVersion,
					"cz-opensearchdashboards": tagVersion,
				}

				var bundleProgress *tui.Progress
				if !opts.ciMode {
					bundleProgress = tui.New(ctx, spinnerOut)
					bundleProgress.WriteLine(lsImageMsg, "")
				}
				path := filepath.Join(filesystem.DownloadDir(), logServerZipName)
				zip, err = appliancepkg.DownloadDockerBundles(ctx, bundleProgress, client, path, opts.dockerRegistry, logServerImages, opts.ciMode)
				if err != nil {
					return err
				}
				defer os.Remove(zip.Name())
				opts.logServerBundlePath = zip.Name()
				if !opts.ciMode {
					bundleProgress.WriteLine("Bundle prepared successfully", "")
					bundleProgress.Wait()
				}
			} else {
				zip, err = os.Open(opts.logServerBundlePath)
				if err != nil {
					return err
				}
			}

			if err := fm.AddToQueue(zip, logServerZipName); err != nil {
				return err
			}
		} else {
			fmt.Fprint(spinnerOut, "LogServer image already exists on appliance. Skipping\n\n")
		}
	}

	// Step 2
	shouldUpload := false
	existingFile, err := a.FileStatus(ctx, opts.filename)
	if err != nil {
		// if we dont get 404, return err
		if errors.Is(err, api.ErrFileNotFound) {
			shouldUpload = true
		} else {
			return err
		}
	}
	if !shouldUpload && existingFile.GetStatus() != appliancepkg.FileReady {
		log.WithField("file", opts.filename).Infof("Remote file already exist, but is in status %s, overriding it", existingFile.GetStatus())
		shouldUpload = true
	}
	if existingFile.GetStatus() == appliancepkg.FileReady {
		log.WithField("file", existingFile.GetName()).Info("File already exists, using it as is")
	}

	if shouldUpload && !opts.remoteImage {
		i, err := os.Open(opts.image)
		if err != nil {
			return err
		}
		if err := fm.AddToQueue(i, ""); err != nil {
			return err
		}
	}
	if fm.QueueLen() > 0 {
		if len(hasLowDiskSpace) > 0 {
			appliancepkg.PrintDiskSpaceWarningMessage(opts.Out, hasLowDiskSpace, opts.Config.Version)
			if !opts.NoInteractive {
				if err := prompt.AskConfirmation(); err != nil {
					return err
				}
			}
		}

		if err := fm.WorkQueue(ctx); err != nil {
			return err
		}
	}

	if opts.remoteBundle && len(opts.logServerBundlePath) > 0 {
		if err := os.Remove(opts.logServerBundlePath); err != nil && !os.IsNotExist(err) {
			log.Warnf("Failed to delete temporary LogServer bundle file: %v", err)
		}
	}

	if opts.remoteImage && opts.hostOnController && existingFile.GetStatus() != appliancepkg.FileReady {
		fmt.Fprintf(opts.Out, "[%s] The primary Controller as host. Uploading upgrade image:\n", time.Now().Format(time.RFC3339))

		if err := a.UploadToController(ctx, opts.image, opts.filename); err != nil {
			return err
		}
		fileUploadStatus := func(ctx context.Context, controller openapi.Appliance) error {
			status := ""
			var p *tui.Progress
			var t *tui.Tracker
			if !opts.ciMode {
				p = tui.New(ctx, spinnerOut)
				defer p.Wait()
				t = p.AddTracker(controller.GetName(), "waiting", "uploaded")
				go t.Watch([]string{appliancepkg.FileReady}, []string{appliancepkg.FileFailed})
			}
			for status != appliancepkg.FileReady {
				remoteFile, err := a.FileStatus(ctx, opts.filename)
				if err != nil {
					return err
				}
				status = remoteFile.GetStatus()
				if t != nil {
					t.Update(status)
				}
				if status == appliancepkg.FileReady {
					break
				}
				if status == appliancepkg.FileFailed {
					reason := errors.New(remoteFile.GetFailureReason())
					return fmt.Errorf("Upload to the Controller failed: %w", reason)
				}
				// Arbitrary sleep for not polling file status from the API too much
				time.Sleep(time.Second * 2)
			}
			return nil
		}
		fileStatusCtx, fileStatusCancel := context.WithTimeout(ctx, opts.timeout)
		defer fileStatusCancel()
		if err := fileUploadStatus(fileStatusCtx, *primaryController); err != nil {
			return err
		}
	}

	// Step 3
	primaryControllerHostname, ok := primaryController.GetHostnameOk()
	if !ok || primaryControllerHostname == nil {
		return errors.New("failed to fetch configured hostname for primary controller")
	}
	remoteFilePath := fmt.Sprintf("controller://%s/%s", *primaryControllerHostname, opts.filename)

	if opts.remoteImage && !opts.hostOnController {
		remoteFilePath = opts.image
	}

	// prepare the image on the appliances,
	// its throttle based on nWorkers to reduce internal rate limit if we try to download from too many appliances at once.
	prepare := func(ctx context.Context, remoteFilePath string, appliances []openapi.Appliance, workers int) error {
		var errs *multierr.Error
		log.Infof("Remote file path for the Controller %s", remoteFilePath)
		var (
			count = len(appliances)
			// qw is the FIFO queue that will run Upgrade concurrently on number of workers.
			qw = queue.New(count, workers)
			// wantedStatus is the desired state for the queued jobs, we need to limit these jobs, and run them in order
			wantedStatus = []string{
				appliancepkg.UpgradeStatusVerifying,
				appliancepkg.UpgradeStatusReady,
			}
			// prepareReady is used for the status bars to mark them as ready if everything is successful.
			prepareReady = []string{appliancepkg.UpgradeStatusReady, appliancepkg.UpgradeStatusSuccess}
			// updateProgressBars is the container for the progress bars
			updateProgressBars *tui.Progress
		)

		if !opts.ciMode {
			updateProgressBars = tui.New(ctx, spinnerOut)
			updateProgressBars.WriteLine("Preparing appliances", "\n")
			defer updateProgressBars.Wait()
		}

		type queueStruct struct {
			appliance openapi.Appliance
			tracker   *tui.Tracker
			deadline  time.Time
			err       error
		}

		for _, ap := range appliances {
			appliance := ap

			// Check if same or older image is already prepared for the appliance
			status, err := a.UpgradeStatus(ctx, appliance.GetId())
			if err != nil {
				log.WithError(err).WithField("applianceID", appliance.GetId()).Debug("Failed to determine current upgrade status")
			}
			if status.GetStatus() != appliancepkg.UpgradeStatusIdle {
				log.WithFields(log.Fields{
					"appliance":      appliance.GetName(),
					"upgrade_status": status.GetStatus(),
				}).Info("another version is already prepared on the appliance. cancelling before proceeding")
				if err := a.UpgradeCancel(ctx, appliance.GetId()); err != nil {
					errs = multierr.Append(errs, err)
				}
				if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, appliance, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, nil); err != nil {
					errs = multierr.Append(errs, err)
				}
			}

			unwantedStatus := []string{appliancepkg.UpgradeStatusFailed}
			var t *tui.Tracker
			if !opts.ciMode {
				t = updateProgressBars.AddTracker(appliance.GetName(), "waiting", appliancepkg.UpgradeStatusReady)
				go t.Watch(prepareReady, unwantedStatus)
			}

			qs := queueStruct{
				appliance: appliance,
				tracker:   t,
			}
			qw.Push(qs)
		}

		// Process the initial queue and wait until the status check has passed the 'downloading' stage,
		// once it has past the 'downloading' stage, we will go to the next item in the queue.
		queueContinue := make(chan queueStruct)
		go func() {
			qw.Work(func(v interface{}) error {
				if v == nil {
					return nil
				}
				// unwantedStatus is used to determine if the upgrade prepare has failed
				unwantedStatus := []string{appliancepkg.UpgradeStatusFailed}
				qs := v.(queueStruct)
				ctx, cancel := context.WithTimeout(ctx, opts.timeout)
				defer cancel()
				deadline, ok := ctx.Deadline()
				if !ok {
					log.WithContext(ctx).Warning("no deadline in context")
				}
				// wait for appliance to be ready before preparing
				if err := a.ApplianceStats.WaitForApplianceStatus(ctx, qs.appliance, appliancepkg.StatusNotBusy, nil); err != nil {
					queueContinue <- queueStruct{err: err}
					return err
				}
				changeID, err := a.PrepareFileOn(ctx, remoteFilePath, qs.appliance.GetId(), opts.DevKeyring)
				if err != nil {
					queueContinue <- queueStruct{err: err}
					return err
				}
				if opts.Config.Version >= 15 {
					c, err := ac.RetryUntilCompleted(ctx, changeID, qs.appliance.GetId())
					if err != nil {
						queueContinue <- queueStruct{err: err}
						return err
					}
					log.WithFields(log.Fields{
						"result":    c.GetResult(),
						"status":    c.GetStatus(),
						"details":   c.GetDetails(),
						"appliance": qs.appliance.GetName(),
					}).Info("prepare image change")
				}
				if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, qs.appliance, wantedStatus, unwantedStatus, qs.tracker); err != nil {
					queueContinue <- queueStruct{err: err}
					return err
				}
				queueContinue <- queueStruct{
					appliance: qs.appliance,
					tracker:   qs.tracker,
					deadline:  deadline,
				}
				return nil
			})
			close(queueContinue)
		}()

		// continues preparing appliances until we either reach the desired state or fail
		// this is run after an appliance has reached the verifying stage and is released from the
		var wg sync.WaitGroup
		errChan := make(chan error)
		for qs := range queueContinue {
			wg.Add(1)
			go func(wg *sync.WaitGroup, qs queueStruct) {
				defer wg.Done()
				if qs.err != nil {
					errChan <- qs.err
					return
				}
				ctx, cancel := context.WithDeadline(ctx, qs.deadline)
				defer cancel()
				// unwantedStatus is used to determine if the upgrade prepare has failed
				unwantedStatus := []string{appliancepkg.UpgradeStatusFailed, appliancepkg.UpgradeStatusIdle}
				if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, qs.appliance, prepareReady, unwantedStatus, qs.tracker); err != nil {
					errChan <- err
					return
				}

			}(&wg, qs)
		}

		go func(wg *sync.WaitGroup, errChan chan error) {
			wg.Wait()
			close(errChan)
		}(&wg, errChan)

		for err := range errChan {
			errs = multierr.Append(errs, err)
		}

		return errs.ErrorOrNil()
	}

	workers, err := cmd.Flags().GetInt("throttle")
	if err != nil {
		return err
	}
	if workers <= 0 {
		workers = len(appliances)
	}
	if err := prepare(ctx, remoteFilePath, appliances, workers); err != nil {
		return err
	}

	if !opts.remoteImage || opts.hostOnController {
		// Step 3
		log.Infof("3. Delete upgrade image %s from Controller", opts.filename)
		deleteCtx, deleteCancel := context.WithTimeout(ctx, opts.timeout)
		defer deleteCancel()
		if err := a.DeleteFile(deleteCtx, opts.filename); err != nil {
			log.Warnf("Failed to delete %s from the Controller %s", opts.filename, err)
		}
		log.Infof("File %s deleted from the Controller", opts.filename)
	}
	fmt.Fprintf(opts.Out, "\n[%s] PREPARE COMPLETE\n", time.Now().Format(time.RFC3339))
	log.Info("prepare complete")
	return nil
}

const prepareUpgradeMessage = `PREPARE SUMMARY

{{ if .DockerBundleDownload }}1. Bundle and upload LogServer docker image
2. Upload upgrade image {{.Filepath}} to Controller
3. Prepare upgrade on the following appliances:{{ else -}}

1. Upload upgrade image {{.Filepath}} to Controller
2. Prepare upgrade on the following appliances:{{ end }}

{{ .ApplianceTable }}{{ if .SkipTable }}

The following appliances will be skipped:

{{ .SkipTable }}{{ end }}{{ if .MultiControllerUpgradeWarning }}

WARNING: This upgrade requires all controllers to be upgraded to the same version, but not all
controllers are being prepared for upgrade.
A partial major or minor controller upgrade is not supported. The upgrade will fail unless all
controllers are prepared for upgrade when running 'upgrade complete'.{{ end }}
`

func showPrepareUpgradeMessage(f string, prepareVersion *version.Version, appliance []openapi.Appliance, skip []appliancepkg.SkipUpgrade, stats []openapi.ApplianceWithStatus, multiControllerUpgradeWarning, dockerBundleDownload bool) (string, error) {
	type stub struct {
		Filepath                      string
		ApplianceTable                string
		SkipTable                     string
		MultiControllerUpgradeWarning bool
		DockerBundleDownload          bool
	}
	data := stub{
		Filepath:                      f,
		MultiControllerUpgradeWarning: multiControllerUpgradeWarning,
		DockerBundleDownload:          dockerBundleDownload,
	}

	abuf := &bytes.Buffer{}
	at := util.NewPrinter(abuf, 4)
	at.AddHeader("Appliance", "Online", "Current version", "Prepare version")
	for _, a := range appliance {
		for _, stat := range stats {
			if a.GetId() == stat.GetId() {
				var v string
				version, err := appliancepkg.ParseVersionString(stat.GetApplianceVersion())
				if err != nil {
					v = "N/A"
				} else if version != nil {
					v = version.String()
				}
				online := tui.No
				if appliancepkg.StatsIsOnline(stat) {
					online = tui.Yes
				}
				at.AddLine(a.GetName(), online, v, prepareVersion.String())
			}
		}
	}
	at.Print()
	data.ApplianceTable = util.PrefixStringLines(abuf.String(), " ", 2)

	if len(skip) > 0 {
		bbuf := &bytes.Buffer{}
		bt := util.NewPrinter(bbuf, 4)
		bt.AddHeader("Appliance", "Online", "Current version", "Reason")
		for _, s := range skip {
			for _, stat := range stats {
				if s.Appliance.GetId() == stat.GetId() {
					var v string
					version, err := appliancepkg.ParseVersionString(stat.GetApplianceVersion())
					if err != nil {
						v = "N/A"
					} else if version != nil {
						v = version.String()
					}
					online := tui.No
					if appliancepkg.StatsIsOnline(stat) {
						online = tui.Yes
					}
					bt.AddLine(s.Appliance.GetName(), online, v, s.Reason)
				}
			}
		}
		bt.Print()
		data.SkipTable = util.PrefixStringLines(bbuf.String(), " ", 2)
	}

	tpl := template.Must(template.New("").Parse(prepareUpgradeMessage))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
