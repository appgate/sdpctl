package upgrade

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/queue"
	"github.com/appgate/sdpctl/pkg/util"
	multierr "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
)

type prepareUpgradeOptions struct {
	Config           *configuration.Config
	Out              io.Writer
	Appliance        func(c *configuration.Config) (*appliancepkg.Appliance, error)
	SpinnerOut       func() io.Writer
	debug            bool
	NoInteractive    bool
	image            string
	DevKeyring       bool
	remoteImage      bool
	filename         string
	timeout          time.Duration
	defaultFilter    map[string]map[string]string
	hostOnController bool
	forcePrepare     bool
}

// NewPrepareUpgradeCmd return a new prepare upgrade command
func NewPrepareUpgradeCmd(f *factory.Factory) *cobra.Command {
	f.Config.Timeout = 300
	opts := &prepareUpgradeOptions{
		Config:     f.Config,
		Appliance:  f.Appliance,
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
		Args: func(cmd *cobra.Command, args []string) error {
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
			var errs error
			opts.filename = filepath.Base(opts.image)
			if err := checkImageFilename(opts.filename); err != nil {
				errs = multierr.Append(errs, err)
			}

			// allow remote addr for image, such as aws s3 bucket
			if util.IsValidURL(opts.image) {
				opts.remoteImage = true
				// if the file is a remote image URL, derive the filename from
				// standard lib 'path' instead of 'filepath' to avoid trailing URI elements

				// we can skip error check here since we already validated that its a url
				u, _ := url.Parse(opts.image)
				// remove any query string, and leave us only with the filename
				u.RawQuery = ""
				opts.filename = path.Base(u.String())
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
	flags.BoolVar(&opts.NoInteractive, "no-interactive", false, "suppress interactive prompt with auto accept")
	flags.StringVarP(&opts.image, "image", "", "", "Upgrade image file or URL")
	flags.BoolVar(&opts.DevKeyring, "dev-keyring", false, "Use the development keyring to verify the upgrade image")
	flags.Int("throttle", 5, "Upgrade is done in batches using a throttle value. You can control the throttle using this flag.")
	flags.BoolVar(&opts.hostOnController, "host-on-controller", false, "Use primary controller as image host when uploading from remote source.")
	flags.BoolVar(&opts.forcePrepare, "force", false, "force prepare of upgrade on appliances even though the version uploaded is the same as the version already running on the appliance")

	return prepareCmd
}

func checkImageFilename(i string) error {
	// Check if its a valid filename
	if rg := regexp.MustCompile(`(.+)?\d\.\d\.\d(.+)?\.img\.zip`); !rg.MatchString(i) {
		return errors.New("Invalid mimetype on image file. The format is expected to be a .img.zip archive with a version number, such as 5.5.1")
	}
	return nil
}

var ErrPrimaryControllerVersionErr = errors.New("version mismatch: run sdpctl configure signin")

func prepareRun(cmd *cobra.Command, args []string, opts *prepareUpgradeOptions) error {
	if appliancepkg.IsOnAppliance() {
		return cmdutil.ErrExecutedOnAppliance
	}
	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	spinnerOut := opts.SpinnerOut()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if a.UpgradeStatusWorker == nil {
		a.UpgradeStatusWorker = &appliancepkg.UpgradeStatus{
			Appliance: a,
		}
	}

	targetVersion, err := appliancepkg.ParseVersionString(opts.filename)
	if err != nil {
		log.Debugf("Could not guess target version based on the image file name %q", opts.filename)
	}
	filter := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	Allappliances, err := a.List(ctx, nil)
	if err != nil {
		return err
	}
	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}
	filteredAppliances := appliancepkg.FilterAppliances(Allappliances, filter)

	primaryController, err := appliancepkg.FindPrimaryController(Allappliances, host)
	if err != nil {
		return err
	}

	initialStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	appliances, _, _ := appliancepkg.FilterAvailable(filteredAppliances, initialStats.GetData())
	skipAppliances := []openapi.Appliance{}
	if !opts.forcePrepare {
		appliances, skipAppliances = appliancepkg.CheckVersionsEqual(ctx, initialStats, appliances, targetVersion)
		if len(appliances) <= 0 {
			return errors.New("No appliances to prepare for upgrade. All appliances are already at the same version as the upgrade image")
		}
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
		constraints, _ := version.NewConstraint(">= 5.5.0-*")
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

	// Check if trying to prepare a lower version than current
	// build number check is to make sure even pre-release versions are included if the build number is higher than current
	targetBuild, _ := strconv.ParseInt(targetVersion.Metadata(), 10, 64)
	primaryControllerBuild, _ := strconv.ParseInt(currentPrimaryControllerVersion.Metadata(), 10, 64)
	if targetVersion.LessThan(currentPrimaryControllerVersion) && targetBuild < primaryControllerBuild {
		logEntry := log.WithFields(log.Fields{
			"currentPrimaryControllerVersion": currentPrimaryControllerVersion.String(),
			"targetVersion":                   targetVersion.String(),
		})
		if !opts.forcePrepare {
			logEntry.Error("invalid upgrade version")
			return fmt.Errorf("Downgrading is not allowed.\n\t\tCurrent version:\t%s\n\t\tPrepare version:\t%s\n\t  Please restore a backup instead.", currentPrimaryControllerVersion.String(), targetVersion.String())
		}
		fmt.Fprintf(opts.Out, "\nWARNING: forcing preperation of an older appliance version than currently running\nCurrent version: %s\nPrepare version: %s\n", currentPrimaryControllerVersion.String(), targetVersion.String())
		logEntry.Warn("preparing an older appliance version using the --force flag")
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
	msg, err := showPrepareUpgradeMessage(opts.filename, appliances, skipAppliances, initialStats.GetData())
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
	if !shouldUpload && existingFile.GetStatus() != appliancepkg.FileReady {
		log.WithField("file", opts.filename).Infof("Remote file already exist, but is in status %s, overriding it", existingFile.GetStatus())
		shouldUpload = true
	}
	if existingFile.GetStatus() == appliancepkg.FileReady {
		log.WithField("file", existingFile.GetName()).Info("File already exists, using it as is")
	}
	if shouldUpload && !opts.remoteImage {
		imageFile, err := os.Open(opts.image)
		if err != nil {
			return err
		}
		defer imageFile.Close()
		p := mpb.New(mpb.WithOutput(spinnerOut))
		log.WithField("file", imageFile.Name()).Info("Uploading file")
		if err := a.UploadFile(ctx, imageFile, p); err != nil {
			return err
		}
		log.WithField("file", imageFile.Name()).Info("Uploaded file")

		remoteFile, err := a.FileStatus(ctx, opts.filename)
		if err != nil {
			return err
		}
		if remoteFile.GetStatus() != appliancepkg.FileReady {
			return fmt.Errorf("remote file %q is uploaded, but is in status %s", opts.filename, existingFile.GetStatus())
		}
		log.WithField("file", remoteFile.GetName()).Infof("Status %s", remoteFile.GetStatus())
		p.Wait()
	}
	if opts.remoteImage && opts.hostOnController && existingFile.GetStatus() != appliancepkg.FileReady {
		fmt.Fprintf(opts.Out, "Primary controller as host. Uploading upgrade image:\n")
		token, err := opts.Config.GetBearTokenHeaderValue()
		if err != nil {
			return err
		}
		p := mpb.New(mpb.WithOutput(spinnerOut))
		if err := a.UploadToController(fileStatusCtx, *primaryController, p, token, opts.image, opts.filename); err != nil {
			return err
		}
		p.Wait()
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

	if opts.remoteImage && !opts.hostOnController {
		remoteFilePath = opts.image
	}

	// prepare the image on the appliances,
	// its throttle based on nWorkers to reduce internal rate limit if we try to download from too many appliances at once.
	prepare := func(ctx context.Context, remoteFilePath string, appliances []openapi.Appliance, workers int) error {
		var errs error
		log.Infof("Remote file path for controller %s", remoteFilePath)
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
		)

		updateProgressBars := mpb.New(mpb.WithOutput(spinnerOut))

		for _, ap := range appliances {
			appliance := ap

			// Check if same or older image is already prepared for the appliance
			status, err := a.UpgradeStatus(ctx, appliance.GetId())
			if err != nil {
				log.WithError(err).WithField("applianceID", appliance.GetId()).Debug("Failed to determine current upgrade status")
			}
			preparedVersion, err := appliancepkg.ParseVersionString(status.GetDetails())
			if err != nil {
				log.WithError(err).Warn("Failed to determine currently prepared version")
			}
			uploadVersion, _ := appliancepkg.ParseVersionString(opts.image)

			if preparedVersion != nil && uploadVersion != nil {
				// Cancel current prepared version if the one uploaded is equal or newer
				preparedBuildNr, _ := strconv.ParseInt(preparedVersion.Metadata(), 10, 64)
				uploadBuildNr, _ := strconv.ParseInt(uploadVersion.Metadata(), 10, 64)
				if preparedVersion.LessThanOrEqual(uploadVersion) && uploadBuildNr < preparedBuildNr {
					log.WithFields(log.Fields{
						"uploadVersion":   uploadVersion.String(),
						"preparedVersion": preparedVersion.String(),
						"appliance":       appliance.GetName(),
					}).Info("an older version is already prepared on the appliance. cancelling before proceeding")
					if err := a.UpgradeCancel(ctx, appliance.GetId()); err != nil {
						errs = multierr.Append(errs, err)
					}
					if err := a.UpgradeStatusWorker.Wait(ctx, appliance, []string{appliancepkg.UpgradeStatusIdle}); err != nil {
						errs = multierr.Append(errs, err)
					}
				}
			}

			qw.Push(appliance)
			statusReport := make(chan string)
			a.UpgradeStatusWorker.Watch(ctx, updateProgressBars, appliance, appliancepkg.UpgradeStatusReady, appliancepkg.UpgradeStatusFailed, statusReport)

			go func(appliance openapi.Appliance) {
				if err := a.UpgradeStatusWorker.Subscribe(ctx, appliance, prepareReady, statusReport); err != nil {
					close(statusReport)
				}
			}(appliance)
		}

		// Struct to pass when continuing queue.
		// We need to keep track of the timeout, so the context deadline is passed along
		// to the next phase
		type continueStruct struct {
			appliance openapi.Appliance
			deadline  time.Time
			err       error
		}
		queueContinue := make(chan continueStruct)

		// Process the inital queue and wait until the status check has passed the 'downloading' stage,
		// once it has past the 'downloading' stage, we will go to the next item in the queue.
		go func() {
			err := qw.Work(func(v interface{}) error {
				if v == nil {
					return nil
				}
				appliance := v.(openapi.Appliance)
				ctx, cancel := context.WithTimeout(ctx, opts.timeout)
				defer cancel()
				deadline, _ := ctx.Deadline()
				if err := a.PrepareFileOn(ctx, remoteFilePath, appliance.GetId(), opts.DevKeyring); err != nil {
					return err
				}
				if err := a.UpgradeStatusWorker.Wait(ctx, appliance, wantedStatus); err != nil {
					return err
				}
				queueContinue <- continueStruct{
					appliance: appliance,
					deadline:  deadline,
				}
				return nil
			})
			if err != nil {
				queueContinue <- continueStruct{err: err}
			}
			close(queueContinue)
		}()

		// After image is downloaded, we block until we reach the desired state or fail
		// keeping the deadline from the previous queue processing
		for qc := range queueContinue {
			if qc.err != nil {
				errs = multierr.Append(errs, qc.err)
				continue
			}
			ctx, cancel := context.WithDeadline(context.Background(), qc.deadline)
			if err := a.UpgradeStatusWorker.Wait(ctx, qc.appliance, []string{appliancepkg.UpgradeStatusReady, appliancepkg.UpgradeStatusFailed}); err != nil {
				errs = multierr.Append(errs, err)
			}
			cancel()
		}

		updateProgressBars.Wait()
		return errs
	}

	workers, err := cmd.Flags().GetInt("throttle")
	if err != nil {
		return err
	}
	if workers <= 0 {
		workers = len(appliances)
	}
	fmt.Fprint(opts.Out, "\nPreparing image on appliances:\n")
	if err := prepare(ctx, remoteFilePath, appliances, workers); err != nil {
		return err
	}

	if !opts.remoteImage || opts.hostOnController {
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

{{ if gt (len .SkipAppliances) 0 }}These appliances will be skipped:
{{- range .SkipAppliances }}
  - Current Version: {{.CurrentVersion }}{{"\t"}}{{.Online -}}{{"\t"}} {{.Name -}}
{{- end }}
{{ end }}`

func showPrepareUpgradeMessage(f string, appliance []openapi.Appliance, skip []openapi.Appliance, stats []openapi.StatsAppliancesListAllOfData) (string, error) {
	type applianceData struct {
		Name           string
		CurrentVersion string
		Online         string
	}
	type stub struct {
		Filepath       string
		Appliances     []applianceData
		SkipAppliances []applianceData
	}
	data := stub{Filepath: f}
	for _, a := range appliance {
		for _, stat := range stats {
			if a.GetId() == stat.GetId() {
				version, _ := appliancepkg.ParseVersionString(stat.GetVersion())
				i := applianceData{
					Name:           a.GetName(),
					CurrentVersion: version.String(),
					Online:         "Offline ⨯",
				}

				if appliancepkg.StatsIsOnline(stat) {
					i.Online = "Online ✓"
				}
				data.Appliances = append(data.Appliances, i)
			}
		}
	}

	for _, s := range skip {
		for _, stat := range stats {
			if s.GetId() == stat.GetId() {
				version, _ := appliancepkg.ParseVersionString(stat.GetVersion())
				i := applianceData{
					Name:           s.GetName(),
					CurrentVersion: version.String(),
					Online:         "Offline ⨯",
				}
				if appliancepkg.StatsIsOnline(stat) {
					i.Online = "Online ✓"
				}
				data.SkipAppliances = append(data.SkipAppliances, i)
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
