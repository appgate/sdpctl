package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
)

type upgradeCompleteOptions struct {
	Config            *configuration.Config
	Out               io.Writer
	Appliance         func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug             bool
	backup            bool
	backupDestination string
	NoInteractive     bool
	Timeout           time.Duration
	actualHostname    string
	defaultFilter     map[string]map[string]string
}

// NewUpgradeCompleteCmd return a new upgrade status command
func NewUpgradeCompleteCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeCompleteOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		Timeout:   DefaultTimeout,
		defaultFilter: map[string]map[string]string{
			"include": {},
			"exclude": {
				"active": "false",
			},
		},
	}
	var upgradeCompleteCmd = &cobra.Command{
		Use:     "complete",
		Short:   docs.ApplianceUpgradeCompleteDoc.Short,
		Long:    docs.ApplianceUpgradeCompleteDoc.Long,
		Example: docs.ApplianceUpgradeCompleteDoc.ExampleString(),
		Args: func(cmd *cobra.Command, args []string) error {
			var err error
			if opts.NoInteractive, err = cmd.Flags().GetBool("no-interactive"); err != nil {
				return err
			}

			minTimeout := 5 * time.Minute
			flagTimeout, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				return err
			}
			if flagTimeout < minTimeout {
				fmt.Printf("WARNING: timeout is less than the allowed minimum. Using default timeout instead: %s", opts.Timeout)
			} else {
				opts.Timeout = flagTimeout
			}

			actualHostname, err := cmd.Flags().GetString("actual-hostname")
			if err != nil {
				return err
			}
			if len(actualHostname) > 0 {
				opts.actualHostname = actualHostname
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCompleteRun(c, args, &opts)
		},
	}

	flags := upgradeCompleteCmd.Flags()
	flags.BoolVarP(&opts.backup, "backup", "b", opts.backup, "backup primary controller before completing upgrade")
	flags.StringVar(&opts.backupDestination, "backup-destination", appliancepkg.DefaultBackupDestination, "specify path to download backup")
	flags.String("actual-hostname", "", "If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname.")

	return upgradeCompleteCmd
}

func upgradeCompleteRun(cmd *cobra.Command, args []string, opts *upgradeCompleteOptions) error {
	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}

	if a.ApplianceStats == nil {
		a.ApplianceStats = &appliancepkg.ApplianceStatus{
			Appliance: a,
		}
	}
	if a.UpgradeStatusWorker == nil {
		a.UpgradeStatusWorker = &appliancepkg.UpgradeStatus{
			Appliance: a,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	filter := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	rawAppliances, err := a.List(ctx, nil)
	if err != nil {
		return err
	}

	// if backup is default value (false) and user hasn't explicitly stated the flag, ask if user wants to backup
	flagIsChanged := cmd.Flags().Changed("backup")
	toBackup := []openapi.Appliance{}
	if !opts.backup && !flagIsChanged && !opts.NoInteractive {
		performBackup := &survey.Confirm{
			Message: "Do you want to backup before proceeding?",
			Default: false,
		}

		if err := survey.AskOne(performBackup, &opts.backup); err != nil {
			return err
		}

		// if answer is yes, ask where to save the backup
		if opts.backup {
			destPrompt := &survey.Input{
				Message: "Path to where backup should be saved",
				Default: os.ExpandEnv(opts.backupDestination),
			}

			if err := survey.AskOne(destPrompt, &opts.backupDestination, nil); err != nil {
				return err
			}

			toBackup, err = appliancepkg.BackupPrompt(rawAppliances, []openapi.Appliance{})
			if err != nil {
				return err
			}
		}
	}

	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}
	controlHost := host
	if len(opts.actualHostname) > 0 {
		controlHost = opts.actualHostname
	}
	primaryController, err := appliancepkg.FindPrimaryController(rawAppliances, controlHost)
	if err != nil {
		return err
	}

	bOpts := appliancepkg.BackupOpts{
		Config:        opts.Config,
		Appliance:     opts.Appliance,
		Destination:   opts.backupDestination,
		AllFlag:       false,
		PrimaryFlag:   false,
		Timeout:       5 * time.Minute,
		Out:           opts.Out,
		NoInteractive: opts.NoInteractive,
		Quiet:         true,
	}
	if opts.backup && len(toBackup) <= 0 {
		toBackup = append(toBackup, *primaryController)
	}

	allAppliances := appliancepkg.FilterAppliances(rawAppliances, filter)
	initialStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	appliances, offline, err := appliancepkg.FilterAvailable(allAppliances, initialStats.GetData())
	if err != nil {
		return fmt.Errorf("Could not complete upgrade operation %w", err)
	}
	for _, o := range offline {
		log.Warnf("%q is offline and will be excluded from upgrade.", o.GetName())
	}

	if hasLowDiskSpace := appliancepkg.HasLowDiskSpace(initialStats.GetData()); len(hasLowDiskSpace) > 0 {
		appliancepkg.PrintDiskSpaceWarningMessage(opts.Out, hasLowDiskSpace)
		if !opts.NoInteractive {
			if err := prompt.AskConfirmation(); err != nil {
				return err
			}
		}
	}

	currentPrimaryControllerVersion, err := appliancepkg.GetApplianceVersion(*primaryController, initialStats)
	if err != nil {
		return err
	}
	// if we have an existing config with the primary controller version, check if we need to re-authetnicate
	// before we continue with the upgrade to update the peer API version.
	if len(opts.Config.PrimaryControllerVersion) > 0 {
		preV, err := version.NewVersion(opts.Config.PrimaryControllerVersion)
		if err != nil {
			return err
		}
		if !preV.Equal(currentPrimaryControllerVersion) {
			return fmt.Errorf("version mismatch: run sdpctl configure signin")
		}
	}

	f := log.Fields{
		"appliance": primaryController.GetName(),
		"version":   currentPrimaryControllerVersion.String(),
	}
	log.WithFields(f).Info("Found primary controller")
	// We will exclude the primary controller from the others controllers
	// since the primary controller is a special case during the upgrade process.
	for i, appliance := range appliances {
		if appliance.GetId() == primaryController.GetId() {
			appliances = append(appliances[:i], appliances[i+1:]...)
		}
	}

	upgradeStatuses, err := a.UpgradeStatusMap(ctx, appliances)
	if err != nil {
		return err
	}
	for id, result := range upgradeStatuses {
		if !util.InSlice(result.Status, []string{appliancepkg.UpgradeStatusReady, appliancepkg.UpgradeStatusSuccess}) {
			for i, appliance := range appliances {
				if id == appliance.GetId() {
					log.WithField("appliance", appliance.GetName()).Infof("Excluding from upgrade")
					appliances = append(appliances[:i], appliances[i+1:]...)
				}
			}
		}
	}
	groups := appliancepkg.GroupByFunctions(appliances)
	additionalControllers := groups[appliancepkg.FunctionController]
	additionalAppliances := appliances
	for _, ctrls := range additionalControllers {
		for i, app := range additionalAppliances {
			if ctrls.GetId() == app.GetId() {
				additionalAppliances = append(additionalAppliances[:i], additionalAppliances[i+1:]...)
			}
		}
	}
	primaryControllerUpgradeStatus, err := a.UpgradeStatus(ctx, primaryController.GetId())
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get upgrade status")
		return err
	}
	newVersion, err := appliancepkg.ParseVersionString(primaryControllerUpgradeStatus.GetDetails())
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to determine upgrade version")
	}
	var newPeerAPIVersion int
	if newVersion != nil {
		newPeerAPIVersion = a.GetPeerAPIVersion(newVersion)
	}

	if primaryControllerUpgradeStatus.GetStatus() != appliancepkg.UpgradeStatusReady && len(additionalControllers) <= 0 && len(additionalAppliances) <= 0 {
		return fmt.Errorf("No appliances are ready to upgrade. Please run 'upgrade prepare' before trying to complete an upgrade")
	}

	// chunks include slices of slices, divided in chunkSize,
	// the chunkSize represent the number of goroutines used
	// for pararell upgrades, each chunk the slice has tried to split
	// the appliances based on site and function to avoid downtime
	// the chunkSize is determined by the number of active sites.
	chunkSize := appliancepkg.ActiveSitesInAppliances(additionalAppliances)
	chunks := appliancepkg.ChunkApplianceGroup(chunkSize, appliancepkg.SplitAppliancesByGroup(additionalAppliances))
	chunkLength := len(chunks)

	msg := ""
	if primaryControllerUpgradeStatus.GetStatus() == appliancepkg.UpgradeStatusReady {
		msg, err = printCompleteSummary(opts.Out, primaryController, additionalControllers, chunks, offline, toBackup, newVersion)
		if err != nil {
			return err
		}
	} else {
		msg, err = printCompleteSummary(opts.Out, nil, additionalControllers, chunks, offline, toBackup, newVersion)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(opts.Out, msg)

	if !opts.NoInteractive {
		if err = prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	if opts.backup {
		if len(toBackup) > 0 {
			ids := []string{}
			for _, t := range toBackup {
				ids = append(ids, t.GetId())
			}
			bOpts.FilterFlag = map[string]map[string]string{
				"include": {
					"id": strings.Join(ids, appliancepkg.FilterDelimiter),
				},
			}
		}
		fmt.Fprint(opts.Out, "\nBacking up:\n")
		if err := appliancepkg.PrepareBackup(&bOpts); err != nil {
			return err
		}
		backupMap, err := appliancepkg.PerformBackup(cmd, args, &bOpts)
		if err != nil {
			return err
		}
		if err := appliancepkg.CleanupBackup(&bOpts, backupMap); err != nil {
			return err
		}
	}

	// 1. Disable Controller function on the following appliance
	// we will run this sequencelly, since this is a sensitive operation
	// so that we can leave the collective gracefully.
	fmt.Fprint(opts.Out, "\nInitializing upgrade:\n")
	initP := mpb.New(mpb.WithOutput(opts.Out))
	disableAdditionalControllers := appliancepkg.ShouldDisable(currentPrimaryControllerVersion, newVersion)
	if disableAdditionalControllers {
		for _, controller := range additionalControllers {
			spinner := util.AddDefaultSpinner(initP, controller.GetName(), "disabling", "disabled")
			f := log.Fields{"appliance": controller.GetName()}
			log.WithFields(f).Info("Disabling controller function")
			if err := a.DisableController(ctx, controller.GetId(), controller); err != nil {
				spinner.Abort(false)
				log.WithFields(f).Error("Unable to disable controller")
				return err
			}
			if err := a.ApplianceStats.WaitForState(ctx, controller, "appliance_ready", nil); err != nil {
				spinner.Abort(false)
				log.WithFields(f).Error("never reached desired state")
				return err
			}
			spinner.Increment()
		}
	}

	// verify the state for all controller
	verifyingSpinner := util.AddDefaultSpinner(initP, "verifying states", "verifying", "ready")
	state := "controller_ready"
	if cfg.Version < 15 {
		state = "single_controller_ready"
	}
	if err := a.ApplianceStats.WaitForState(ctx, *primaryController, state, nil); err != nil {
		verifyingSpinner.Abort(false)
		return fmt.Errorf("primary controller %s", err)
	}
	log.Info("all controllers are in correct state")

	if cfg.Version >= 15 && len(additionalControllers) > 0 {
		for _, controller := range additionalControllers {
			f := log.Fields{"controller": controller.GetName()}
			log.WithFields(f).Info("enabling maintenance mode")
			id, err := a.EnableMaintenanceMode(ctx, controller.GetId())
			if err != nil {
				log.WithFields(f).Warnf("Unable to enable maintenance mode %s", err)
				return err
			}
			log.WithFields(f).Infof("id %s", id)
		}
	}
	m, err := a.UpgradeStatusMap(ctx, appliances)
	if err != nil {
		log.WithError(err).Error("Upgrade status failed")
		return err
	}
	notReady := make([]string, 0)
	for _, result := range m {
		log.WithFields(log.Fields{
			"appliance": result.Name,
		}).Infof("Upgrade status %s", result.Status)
		if !util.InSlice(result.Status, []string{appliancepkg.UpgradeStatusReady, appliancepkg.UpgradeStatusSuccess}) {
			notReady = append(notReady, result.Name)
		}
	}
	if len(notReady) > 0 {
		verifyingSpinner.Abort(false)
		log.Errorf("appliance %s is not ready for upgrade", strings.Join(notReady, ", "))
		return fmt.Errorf("one or more appliances are not ready for upgrade.")
	}
	verifyingSpinner.Increment()
	initP.Wait()

	ctrlUpgradeState := "controller_ready"
	if cfg.Version < 15 {
		ctrlUpgradeState = "multi_controller_ready"
		if disableAdditionalControllers {
			ctrlUpgradeState = "single_controller_ready"
		}
	}
	if primaryControllerUpgradeStatus.GetStatus() == appliancepkg.UpgradeStatusReady {
		pctx, pcancel := context.WithTimeout(ctx, opts.Timeout)
		fmt.Fprint(opts.Out, "\nUpgrading primary controller:\n")
		primaryP := mpb.New(mpb.WithOutput(opts.Out))
		statusReport := make(chan string)
		a.UpgradeStatusWorker.Watch(pctx, primaryP, *primaryController, ctrlUpgradeState, appliancepkg.UpgradeStatusFailed, statusReport)
		log.WithField("appliance", primaryController.GetName()).Info("Completing upgrade and switching partition")
		if err := a.UpgradeComplete(pctx, primaryController.GetId(), true); err != nil {
			close(statusReport)
			pcancel()
			return err
		}
		log.WithField("appliance", primaryController.GetName()).Infof("Waiting for primary controller to reach state %s", state)
		if err := a.UpgradeStatusWorker.Subscribe(pctx, *primaryController, []string{appliancepkg.UpgradeStatusIdle}, statusReport); err != nil {
			close(statusReport)
			pcancel()
			return err
		}
		if err := a.ApplianceStats.WaitForState(pctx, *primaryController, ctrlUpgradeState, statusReport); err != nil {
			close(statusReport)
			pcancel()
			return err
		}
		close(statusReport)
		log.WithField("appliance", primaryController.GetName()).Info("Primary controller updated")
		pcancel()
		primaryP.Wait()
	}

	batchUpgrade := func(ctx context.Context, p *mpb.Progress, appliances []openapi.Appliance, SwitchPartition bool, finalState string) error {
		g, ctx := errgroup.WithContext(ctx)
		upgradeChan := make(chan openapi.Appliance, len(appliances))
		regex := regexp.MustCompile(`a reboot is required for the upgrade to go into effect`)
		for _, appliance := range appliances {
			bctx, bcancel := context.WithTimeout(ctx, opts.Timeout)
			i := appliance
			g.Go(func() error {
				defer bcancel()
				log.WithField("appliance", i.GetName()).Info("checking if ready")
				statusReport := make(chan string)
				defer close(statusReport)
				a.UpgradeStatusWorker.Watch(bctx, p, i, finalState, appliancepkg.UpgradeStatusFailed, statusReport)
				if err := a.UpgradeComplete(bctx, i.GetId(), SwitchPartition); err != nil {
					close(statusReport)
					return err
				}
				if !SwitchPartition {
					if err := a.UpgradeStatusWorker.Subscribe(bctx, i, []string{appliancepkg.UpgradeStatusSuccess}, statusReport); err != nil {
						close(statusReport)
						return err
					}
					status, err := a.UpgradeStatus(bctx, i.GetId())
					if err != nil {
						close(statusReport)
						return err
					}
					if regex.MatchString(status.GetDetails()) {
						if err := a.UpgradeSwitchPartition(bctx, i.GetId()); err != nil {
							close(statusReport)
							return err
						}
						log.WithField("appliance", i.GetName()).Info("Switching partition")
					}
				}
				if err := a.UpgradeStatusWorker.Subscribe(bctx, i, []string{appliancepkg.UpgradeStatusIdle}, statusReport); err != nil {
					close(statusReport)
					return err
				}
				if err := a.ApplianceStats.WaitForState(bctx, i, finalState, statusReport); err != nil {
					close(statusReport)
					return err
				}
				select {
				case <-bctx.Done():
					close(statusReport)
					return bctx.Err()
				case upgradeChan <- i:
				}
				return nil
			})
		}
		go func() {
			g.Wait()
			close(upgradeChan)
		}()
		if err := g.Wait(); err != nil {
			log.WithError(err).Error(err.Error())
			return fmt.Errorf("Error during upgrade of an appliance %w", err)
		}
		return nil
	}

	backoffEnableController := func(controller openapi.Appliance) error {
		b := backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval: 10 * time.Second,
			Multiplier:      1,
			MaxInterval:     2 * time.Minute,
			MaxElapsedTime:  15 * time.Minute,
			Stop:            backoff.Stop,
			Clock:           backoff.SystemClock,
		}, ctx)

		return backoff.Retry(func() error {
			if err := a.EnableController(ctx, controller.GetId(), controller); err != nil {
				log.Infof("Failed to enabled controller function on %s, will retry", controller.GetName())
				return err
			}
			log.Infof("Enabled controller function OK on %s", controller.GetName())
			return nil
		}, b)
	}

	if len(additionalControllers) > 0 {
		fmt.Fprint(opts.Out, "\nUpgrading additional controllers:\n")
		for _, ctrl := range additionalControllers {
			ctrlCtx, ctrlCancel := context.WithTimeout(ctx, opts.Timeout)
			ctrlP := mpb.New(mpb.WithOutput(opts.Out))
			finalState := "controller_ready"
			if cfg.Version < 15 {
				finalState = "multi_controller_ready"
			}
			statusReport := make(chan string)
			a.UpgradeStatusWorker.Watch(ctx, ctrlP, ctrl, finalState, appliancepkg.UpgradeStatusFailed, statusReport)
			if err := a.UpgradeComplete(ctx, ctrl.GetId(), true); err != nil {
				close(statusReport)
				ctrlCancel()
				return err
			}
			if err := a.UpgradeStatusWorker.Subscribe(ctrlCtx, ctrl, []string{appliancepkg.UpgradeStatusIdle}, statusReport); err != nil {
				ctrlCancel()
				close(statusReport)
				return err
			}
			if disableAdditionalControllers {
				if err := backoffEnableController(ctrl); err != nil {
					log.WithFields(f).WithError(err).Error("Failed to enable controller")
					if merr, ok := err.(*multierror.Error); ok {
						var mutliErr error
						for _, e := range merr.Errors {
							mutliErr = multierror.Append(e)
						}
						mutliErr = multierror.Append(fmt.Errorf("could not enable controller on %s", ctrl.GetName()))
						ctrlCancel()
						close(statusReport)
						return mutliErr
					}
					close(statusReport)
					ctrlCancel()
					return err
				}
			}
			if err := a.ApplianceStats.WaitForState(ctx, ctrl, finalState, statusReport); err != nil {
				log.WithFields(f).WithError(err).Error("Controller never reached desired state")
				close(statusReport)
				ctrlCancel()
				return err
			}
			if cfg.Version >= 15 {
				_, err := a.DisableMaintenanceMode(ctx, ctrl.GetId())
				if err != nil {
					close(statusReport)
					ctrlCancel()
					return err
				}
				log.WithFields(f).Info("Disabled maintenance mode")
			}
			close(statusReport)
			ctrlCancel()
			ctrlP.Wait()
		}
		log.Info("done waiting for additional controllers upgrade")
	}

	for index, chunk := range chunks {
		chunkP := mpb.New(mpb.WithOutput(opts.Out))
		fmt.Fprintf(opts.Out, "\nUpgrading additional appliances (Batch %d / %d):\n", index+1, chunkLength)
		if err := batchUpgrade(ctx, chunkP, chunk, false, "appliance_ready"); err != nil {
			return fmt.Errorf("failed during upgrade of additional appliances %w", err)
		}
		chunkP.Wait()
	}

	if newVersion != nil && newVersion.GreaterThan(currentPrimaryControllerVersion) {
		cfg.PrimaryControllerVersion = newVersion.String()
		cfg.Version = newPeerAPIVersion
		viper.Set("primary_controller_version", newVersion.String())
		viper.Set("api_version", newPeerAPIVersion)
		if err := viper.WriteConfig(); err != nil {
			log.WithFields(log.Fields{
				"primary_controller_version": newVersion.String(),
				"api_version":                newPeerAPIVersion,
			}).WithError(err).Warn("failed to write config file")
			fmt.Fprintln(opts.Out, "WARNING: Failed to write to config file. Please run 'sdpctl configure signin' to reconfigure.")
		}
	}

	// Check if all appliances are running the same version after upgrade complete
	newStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	newStatsData := newStats.GetData()
	hasDiff, versionList := appliancepkg.HasDiffVersions(newStatsData)

	postSummary, err := printPostCompleteSummary(versionList, hasDiff)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Out, "\n%s\n", postSummary)

	return nil
}

func printCompleteSummary(out io.Writer, primaryController *openapi.Appliance, additionalControllers []openapi.Appliance, chunks [][]openapi.Appliance, skipped, backup []openapi.Appliance, toVersion *version.Version) (string, error) {
	type tplStub struct {
		PrimaryController     string
		AdditionalControllers []string
		Chunks                map[int][]string
		Skipped               []string
		Backup                []string
		Version               string
	}
	completeSummaryTpl := `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in a few ordered steps:

 1. The primary controller will be upgraded.
    This will result in the API being unreachable while completing the primary controller upgrade.

 2. Additional controllers will be upgraded.
    In some cases, the controller function on additional controllers will need to be disabled
    before proceeding with the upgrade. The disabled controllers will then be re-enabled once
    the upgrade is completed.
    This step will also reboot the upgraded controllers for the upgrade to take effect.

 3. The remaining appliances will be upgraded. The additional appliances will be split into
    batches to keep the collective as available as possible during the upgrade process.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

{{ if .Version -}}The following appliances will be upgraded to version {{ .Version }}:{{ else }}The following appliances will be upgraded:{{ end }}
{{- with .PrimaryController }}
  Primary Controller: {{ . }}
{{ end }}
{{- with .AdditionalControllers }}
  Additional Controllers:{{ range . }}
  - {{ . }}{{ end }}
{{ end }}
{{- if .Chunks }}
  Additional Appliances:{{ range $i, $v := .Chunks }}
    Batch #{{$i}}:{{ range $v }}
    - {{.}}{{end}}{{ end }}
{{ end }}
{{- with .Skipped }}
Appliances that will be skipped:{{ range . }}
  - {{ . }}{{ end }}
{{ end }}
{{ with .Backup -}}
Appliances that will be backed up before completing upgrade:{{ range . }}
  - {{ . }}{{ end }}
{{ end }}
`
	additionalControllerNames := []string{}
	for _, a := range additionalControllers {
		additionalControllerNames = append(additionalControllerNames, a.GetName())
	}
	upgradeChunks := map[int][]string{}
	for i, chunk := range chunks {
		chunkSlice := []string{}
		for _, a := range chunk {
			chunkSlice = append(chunkSlice, a.GetName())
		}
		index := i + 1
		upgradeChunks[index] = chunkSlice
	}
	toSkip := []string{}
	for _, a := range skipped {
		toSkip = append(toSkip, a.GetName())
	}
	toBackup := []string{}
	for _, a := range backup {
		toBackup = append(toBackup, a.GetName())
	}
	tplData := tplStub{
		AdditionalControllers: additionalControllerNames,
		Chunks:                upgradeChunks,
		Skipped:               toSkip,
		Backup:                toBackup,
	}
	if primaryController != nil {
		tplData.PrimaryController = primaryController.GetName()
	}
	if toVersion != nil {
		tplData.Version = toVersion.String()
	}
	t := template.Must(template.New("").Parse(completeSummaryTpl))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, tplData); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func printPostCompleteSummary(applianceVersions map[string]string, hasDiff bool) (string, error) {
	type tplStub struct {
		ApplianceVersions map[string]string
		HasDiff           bool
	}
	tpl := `UPGRADE COMPLETE

Appliances are now running these versions:
{{ range $appliance, $version := .ApplianceVersions }}
  {{ $appliance }}: {{ $version }}{{ end }}
{{ .HasDiff }}
WARNING: Upgrade was completed, but there are different versions running on the appliances.{{ end }}
`
	tplData := tplStub{
		ApplianceVersions: applianceVersions,
		HasDiff:           hasDiff,
	}
	t := template.Must(template.New("").Parse(tpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, tplData); err != nil {
		return "", err
	}
	return buf.String(), nil
}
