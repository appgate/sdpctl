package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync/atomic"
	"text/template"
	"time"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/hashicorp/go-version"
	"github.com/mitchellh/ioprogress"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type prepareUpgradeOptions struct {
	Config     *configuration.Config
	Out        io.Writer
	Appliance  func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Token      string
	Timeout    int
	url        string
	provider   string
	debug      bool
	insecure   bool
	image      string
	DevKeyring bool
}

// NewPrepareUpgradeCmd return a new prepare upgrade command
func NewPrepareUpgradeCmd(f *factory.Factory) *cobra.Command {
	opts := &prepareUpgradeOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		Timeout:   10,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var prepareCmd = &cobra.Command{
		Use:   "prepare",
		Short: "prepare upgrade",
		Long: `Prepare an upgrade but do NOT install it.
This means the upgrade file will be downloaded/uploaded to all the appliances,
the signature verified as well as any other preconditions applicable at this point.`,
		RunE: func(c *cobra.Command, args []string) error {
			return prepareRun(c, args, opts)
		},
	}

	flags := prepareCmd.Flags()
	flags.BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	flags.StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	flags.StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	flags.StringVarP(&opts.image, "image", "", "", "image path")
	flags.BoolVar(&opts.DevKeyring, "dev-keyring", true, "Use the development keyring to verify the upgrade image")

	return prepareCmd
}

const (
	fileInProgress = "InProgress"
	fileReady      = "Ready"
	fileFailed     = "Failed"
)

func prepareRun(cmd *cobra.Command, args []string, opts *prepareUpgradeOptions) error {
	if appliancepkg.IsOnAppliance() {
		return appliancepkg.ErrExecutedOnAppliance
	}
	if opts.image == "" {
		return errors.New("Image is mandatory")
	}

	if ok, err := util.FileExists(opts.image); err != nil || !ok {
		return fmt.Errorf("Image file not found %q", opts.image)
	}

	a, err := opts.Appliance(opts.Config)
	if err != nil {
		return err
	}
	if a.UpgradeStatusWorker == nil {
		a.UpgradeStatusWorker = &appliancepkg.UpgradeStatus{
			Appliance: a,
		}
	}
	f, err := os.Open(opts.image)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Minute))
	defer cancel()

	filename := filepath.Base(f.Name())
	targetVersion, err := appliancepkg.GuessVersion(filename)
	if err != nil {
		log.Debugf("Could not guess target version based on the image file name %q", filename)
	}
	filter := util.ParseFilteringFlags(cmd.Flags())
	appliances, err := a.List(ctx, filter)
	if err != nil {
		return err
	}
	initialStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	if appliancepkg.HasLowDiskSpace(initialStats.GetData()) {
		msg, err := appliancepkg.ShowDiskSpaceWarningMessage(initialStats.GetData())
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", msg)
		if err := prompt.AskConfirmation(); err != nil {
			return err
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

	if t, gws := appliancepkg.AutoscalingGateways(appliances); autoScalingWarning && len(gws) > 0 {
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
	if len(peerAppliances) > 0 {
		msg, err := appliancepkg.ShowPeerInterfaceWarningMessage(peerAppliances)
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.Out, "\n%s\n", msg)
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}

	primaryController, err := appliancepkg.FindPrimaryController(groups[appliancepkg.FunctionController], host)
	if err != nil {
		return err
	}
	currentPrimaryControllerVersion, err := appliancepkg.GetPrimaryControllerVersion(*primaryController, initialStats)
	if err != nil {
		return err
	}
	preV, err := version.NewVersion(opts.Config.PrimaryControllerVersion)
	if err != nil {
		return err
	}
	if !preV.Equal(currentPrimaryControllerVersion) {
		return errors.New("version mismatch: run appgatectl configure login")
	}
	fmt.Fprintf(opts.Out, "\n%s\n", fmt.Sprintf(appliancepkg.BackupInstructions, primaryController.Name, appliancepkg.HelpManualURL))
	if err := prompt.AskConfirmation("Have you completed the Controller backup or snapshot?"); err != nil {
		return err
	}
	log.Infof("Primary controller is: %s and running %s", primaryController.Name, currentPrimaryControllerVersion.String())
	if targetVersion != nil {
		log.Infof("Appliances will be prepared for upgrade to version: %s", targetVersion.String())
	}
	msg, err := showPrepareUpgradeMessage(f.Name(), appliances, initialStats.GetData())
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", msg)
	if err := prompt.AskConfirmation(); err != nil {
		return err
	}

	// Step 1
	shouldUpload := false
	existingFile, err := a.FileStatus(ctx, filename)
	if err != nil {
		// if we dont get 404, return err
		if errors.Is(err, appliancepkg.ErrFileNotFound) {
			shouldUpload = true
		} else {
			return err
		}
	}
	if !shouldUpload && existingFile.GetStatus() != fileReady {
		log.Infof("Remote file %q already exist, but is in status %s, overridring it", filename, existingFile.GetStatus())
		shouldUpload = true
	}
	if existingFile.GetStatus() == fileReady {
		log.Infof("File %s already exists, using it as is", existingFile.GetName())
	}
	if shouldUpload {
		fileStat, err := f.Stat()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		fs := fileStat.Size()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", fileStat.Name())
		if err != nil {
			return err
		}
		part.Write(content)
		if err = writer.Close(); err != nil {
			return err
		}
		input := io.NopCloser(&ioprogress.Reader{
			Reader: body,
			Size:   fs,
			DrawFunc: ioprogress.DrawTerminalf(opts.Out, func(p, t int64) string {
				return fmt.Sprintf(
					"Uploading %s: %s",
					f.Name(),
					ioprogress.DrawTextFormatBytes(p, t))
			}),
		})
		headers := map[string]string{
			"Content-Type":        writer.FormDataContentType(),
			"Content-Disposition": fmt.Sprintf("attachment; filename=%q", fileStat.Name()),
		}
		if err := a.UploadFile(ctx, input, headers); err != nil {
			return err
		}
	}
	remoteFile, err := a.FileStatus(ctx, filename)
	if err != nil {
		return err
	}
	if remoteFile.GetStatus() != fileReady {
		return fmt.Errorf("remote file %q is uploaded, but is in status %s", filename, existingFile.GetStatus())
	}
	log.Infof("Remote file %s is %s", remoteFile.GetName(), remoteFile.GetStatus())

	// Step 2
	peerPort := 8443
	if v, ok := primaryController.GetPeerInterfaceOk(); ok {
		peerPort = int(v.GetHttpsPort())
	}
	// prepare the image on the appliances,
	// its throttle based on nWorkers to reduce internal rate limit if we try to download from too many appliances at once.
	prepare := func(ctx context.Context, primaryController openapi.Appliance, appliances []openapi.Appliance) ([]openapi.Appliance, error) {
		remoteFilePath := fmt.Sprintf("controller://%s:%d/%s", primaryController.GetHostname(), peerPort, filename)
		log.Infof("Remote file path for controller %s", remoteFilePath)
		g, ctx := errgroup.WithContext(ctx)

		applianceIds := make(chan openapi.Appliance)
		// Produce, send all appliance Id to the Channel so we can consume them in a fixed rate.
		g.Go(func() error {
			for _, appliance := range appliances {
				applianceIds <- appliance
			}
			close(applianceIds)
			return nil
		})

		// consume Prepare with nWorkers
		nWorkers := 2
		workers := int32(nWorkers)
		finished := make(chan openapi.Appliance)
		for i := 0; i < nWorkers; i++ {
			g.Go(func() error {
				defer func() {
					// Last one out closes the channel
					if atomic.AddInt32(&workers, -1) == 0 {
						close(finished)
					}
				}()
				for appliance := range applianceIds {
					fields := log.Fields{"appliance": appliance.GetName()}
					log.WithFields(fields).Info("Preparing upgrade")
					if err := a.PrepareFileOn(ctx, remoteFilePath, appliance.GetId(), opts.DevKeyring); err != nil {
						log.WithFields(fields).Errorf("Preparing upgrade err %s", err)
						return err
					}
					if err := a.UpgradeStatusWorker.Wait(ctx, []openapi.Appliance{appliance}, appliancepkg.UpgradeStatusReady); err != nil {
						log.WithFields(fields).Errorf("Never reached expected state %s", err)
						return err
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case finished <- appliance:
					}
				}
				return nil
			})
		}
		r := make([]openapi.Appliance, 0)
		g.Go(func() error {
			for appliance := range finished {
				r = append(r, appliance)
			}
			return nil
		})
		return r, g.Wait()
	}
	preparedAppliances, err := prepare(ctx, *primaryController, appliances)
	if err != nil {
		return fmt.Errorf("Preparation failed %s, run appgatectl appliance upgrade cancel", err)
	}
	// Blocking function that checks all appliances upgrade status to verify that
	// everyone reach desired state of ready.
	if err := a.UpgradeStatusWorker.Wait(ctx, preparedAppliances, appliancepkg.UpgradeStatusReady); err != nil {
		return err
	}

	// Step 3
	log.Infof("3. Delete upgrade image %s from Controller", filename)
	if err := a.DeleteFile(ctx, filename); err != nil {
		log.Warnf("Failed to delete %s from controller %s", filename, err)
	}
	log.Infof("File %s deleted from Controller", filename)

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
