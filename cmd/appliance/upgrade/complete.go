package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/network"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/terminal"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
)

type upgradeCompleteOptions struct {
	Config            *configuration.Config
	Out               io.Writer
	SpinnerOut        func() io.Writer
	Appliance         func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug             bool
	backup            bool
	backupDestination string
	NoInteractive     bool
	Timeout           time.Duration
	actualHostname    string
	defaultFilter     map[string]map[string]string
	ciMode            bool
	batchSize         int
}

// NewUpgradeCompleteCmd return a new upgrade status command
func NewUpgradeCompleteCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeCompleteOptions{
		Config:     f.Config,
		Appliance:  f.Appliance,
		debug:      f.Config.Debug,
		Out:        f.IOOutWriter,
		SpinnerOut: f.GetSpinnerOutput(),
		Timeout:    DefaultTimeout,
		backup:     true,
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
		Annotations: map[string]string{
			"updateAPIConfig": "true",
		},
		Args: func(cmd *cobra.Command, args []string) error {
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

			ciModeFlag, err := cmd.Flags().GetBool("ci-mode")
			if err != nil {
				return err
			}
			opts.ciMode = ciModeFlag

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			h, err := opts.Config.GetHost()
			if err != nil {
				return fmt.Errorf("could not determine hostname for %s", err)
			}
			if err := network.ValidateHostnameUniqueness(h); err != nil {
				return err
			}
			return upgradeCompleteRun(c, args, &opts)
		},
	}

	flags := upgradeCompleteCmd.Flags()
	flags.BoolVarP(&opts.backup, "backup", "b", opts.backup, "backup primary controller before completing upgrade")
	flags.StringVar(&opts.backupDestination, "backup-destination", appliancepkg.DefaultBackupDestination, "specify path to download backup")
	flags.StringVar(&opts.actualHostname, "actual-hostname", "", "If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname.")
	flags.IntVar(&opts.batchSize, "batch-size", 2, "number of batch groups")
	return upgradeCompleteCmd
}

func upgradeCompleteRun(cmd *cobra.Command, args []string, opts *upgradeCompleteOptions) error {
	fmt.Fprintf(opts.Out, "sdpctl_version: %s\n\n", cmd.Root().Version)
	terminal.Lock()
	defer terminal.Unlock()
	var err error
	if opts.NoInteractive, err = cmd.Flags().GetBool("no-interactive"); err != nil {
		return err
	}

	cfg := opts.Config
	a, err := opts.Appliance(cfg)
	if err != nil {
		return err
	}
	spinnerOut := opts.SpinnerOut()
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
	ac := change.ApplianceChange{
		APIClient: a.APIClient,
		Token:     a.Token,
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
	if !flagIsChanged && !opts.NoInteractive {
		performBackup := &survey.Confirm{
			Message: "Do you want to backup before proceeding?",
			Default: opts.backup,
		}

		if err := survey.AskOne(performBackup, &opts.backup); err != nil {
			return err
		}

		// if answer is yes, ask where to save the backup
		if opts.backup {
			destPrompt := &survey.Input{
				Message: "Path to where backup should be saved",
				Default: filesystem.AbsolutePath(opts.backupDestination),
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
	primaryController, err := appliancepkg.FindPrimaryController(rawAppliances, controlHost, true)
	if err != nil {
		return err
	}

	bOpts := appliancepkg.BackupOpts{
		Config:        opts.Config,
		Appliance:     opts.Appliance,
		Destination:   opts.backupDestination,
		AllFlag:       false,
		PrimaryFlag:   false,
		Out:           opts.Out,
		SpinnerOut:    opts.SpinnerOut,
		NoInteractive: opts.NoInteractive,
		Quiet:         true,
	}
	if opts.backup && len(toBackup) <= 0 {
		toBackup = append(toBackup, *primaryController)
	}

	allAppliances, _, err := appliancepkg.FilterAppliances(rawAppliances, filter)
	if err != nil {
		return err
	}
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

	currentPrimaryControllerVersion, err := appliancepkg.GetApplianceVersion(*primaryController, *initialStats)
	if err != nil {
		return err
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

	// isolate additional controllers
	additionalControllers := groups[appliancepkg.FunctionController]
	additionalAppliances := appliances
	for _, ctrls := range additionalControllers {
		for i, app := range additionalAppliances {
			if ctrls.GetId() == app.GetId() {
				additionalAppliances = append(additionalAppliances[:i], additionalAppliances[i+1:]...)
			}
		}
	}

	// isolate log forwarders and log servers
	// this is only needed when upgrading to version 6.0 from 5.x, so we need to check for this particular case
	logForwardersAndServersAll := append(groups[appliancepkg.FunctionLogServer], groups[appliancepkg.FunctionLogForwarder]...)
	logForwardersAndServers := []openapi.Appliance{}
	for _, lfs := range logForwardersAndServersAll {
		upgradeStatus, err := a.UpgradeStatus(ctx, lfs.GetId())
		if err != nil {
			return err
		}
		currentVersion, err := appliancepkg.GetApplianceVersion(lfs, *initialStats)
		if err != nil {
			return err
		}
		upgradeVersion, err := appliancepkg.ParseVersionString(upgradeStatus.GetDetails())
		if err != nil {
			return err
		}

		v6, err := version.NewConstraint(">= 6.0.0-beta")
		if err != nil {
			return err
		}
		if v6.Check(upgradeVersion) && !v6.Check(currentVersion) {
			for i, app := range additionalAppliances {
				if lfs.GetId() == app.GetId() {
					additionalAppliances = append(additionalAppliances[:i], additionalAppliances[i+1:]...)
				}
			}
			isAlsoInControllers := false
			for _, ctrl := range additionalControllers {
				if lfs.GetId() == ctrl.GetId() {
					isAlsoInControllers = true
				}
			}
			if !isAlsoInControllers {
				logForwardersAndServers = append(logForwardersAndServers, lfs)
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

	if primaryControllerUpgradeStatus.GetStatus() != appliancepkg.UpgradeStatusReady && len(additionalControllers) <= 0 && len(additionalAppliances) <= 0 {
		return fmt.Errorf("No appliances are ready to upgrade. Please run 'upgrade prepare' before trying to complete an upgrade")
	}

	// chunks include slices of slices, divided in chunkSize,
	// the chunkSize represent the number of goroutines used
	// for parallel upgrades, each chunk the slice has tried to split
	// the appliances based on site and function to avoid downtime.
	//
	// users can overwrite chunkSize with chunkSize '--batch-size' flag
	chunks := appliancepkg.ChunkApplianceGroup(opts.batchSize, appliancepkg.SplitAppliancesByGroup(additionalAppliances))
	chunkLength := len(chunks)

	msg := ""
	if primaryControllerUpgradeStatus.GetStatus() == appliancepkg.UpgradeStatusReady {
		msg, err = printCompleteSummary(opts.Out, primaryController, additionalControllers, logForwardersAndServers, chunks, offline, toBackup, opts.backupDestination, newVersion)
		if err != nil {
			return err
		}
	} else {
		msg, err = printCompleteSummary(opts.Out, nil, additionalControllers, logForwardersAndServers, chunks, offline, toBackup, opts.backupDestination, newVersion)
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
		fmt.Fprintf(opts.Out, "\n[%s] Backing up:\n", time.Now().Format(time.RFC3339))
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
	fmt.Fprintf(opts.Out, "\n[%s] Initializing upgrade:\n", time.Now().Format(time.RFC3339))
	initP := mpb.NewWithContext(ctx, mpb.WithOutput(spinnerOut))
	disableAdditionalControllers := appliancepkg.ShouldDisable(currentPrimaryControllerVersion, newVersion)
	if disableAdditionalControllers {
		for _, controller := range additionalControllers {
			spinner := tui.AddDefaultSpinner(initP, controller.GetName(), "disabling", "disabled")
			f := log.Fields{"appliance": controller.GetName()}
			log.WithFields(f).Info("Disabling controller function")
			if err := a.DisableController(ctx, controller.GetId(), controller); err != nil {
				spinner.Abort(false)
				log.WithFields(f).Error("Unable to disable controller")
				return err
			}
			if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, nil); err != nil {
				spinner.Abort(false)
				log.WithFields(f).Error("never reached desired state")
				return err
			}
			spinner.Increment()
		}
	}

	// verify the state for all controller
	verifyingSpinner := tui.AddDefaultSpinner(initP, "verifying states", "verifying", "ready")
	if err := a.ApplianceStats.WaitForApplianceState(ctx, *primaryController, appliancepkg.StatReady, nil); err != nil {
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

	if primaryControllerUpgradeStatus.GetStatus() == appliancepkg.UpgradeStatusReady {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading primary controller:\n", time.Now().Format(time.RFC3339))
		upgradeReadyPrimary := func(ctx context.Context, controller openapi.Appliance) error {
			var initialVolume float32
			for _, appData := range initialStats.GetData() {
				if controller.GetId() == appData.GetId() {
					initialVolume = appData.GetVolumeNumber()
				}
			}
			ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
			var primaryControllerBars *tui.Progress
			var t *tui.Tracker
			if !opts.ciMode {
				primaryControllerBars = tui.New(ctx, spinnerOut)
				defer primaryControllerBars.Wait()
				t = primaryControllerBars.AddTracker(controller.GetName(), "upgraded")
				go t.Watch(appliancepkg.StatReady, []string{appliancepkg.UpgradeStatusFailed})
			}

			logEntry := log.WithField("appliance", controller.GetName())
			logEntry.Info("completing upgrade and switching partition")
			if err := a.UpgradeComplete(ctx, controller.GetId(), true); err != nil {
				return err
			}
			logEntry.WithField("want", appliancepkg.StatReady).Info("waiting for primary controller to reach a wanted state")
			if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, controller, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
				return err
			}
			if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, t); err != nil {
				return err
			}
			s, _, err := a.Stats(ctx)
			if err != nil {
				return err
			}

			// Check if partition has been switched
			for _, appData := range s.GetData() {
				if controller.GetId() == appData.GetId() && appData.GetVolumeNumber() == initialVolume {
					return fmt.Errorf("upgrade failed on %s: never switched partition", controller.GetName())
				}
			}

			logEntry.Info("primary controller updated")
			return nil
		}
		if err := upgradeReadyPrimary(ctx, *primaryController); err != nil {
			return err
		}
	}

	batchUpgrade := func(ctx context.Context, appliances []openapi.Appliance, SwitchPartition bool) error {
		g, ctx := errgroup.WithContext(ctx)
		regex := regexp.MustCompile(`a reboot is required for the upgrade to go into effect`)
		upgradeChan := make(chan openapi.Appliance, len(appliances))
		var p *tui.Progress
		if !opts.ciMode {
			p = tui.New(ctx, spinnerOut)
			defer p.Wait()
		}
		for _, appliance := range appliances {
			i := appliance
			var initialVolume float32
			for _, appData := range initialStats.GetData() {
				if i.GetId() == appData.GetId() {
					initialVolume = appData.GetVolumeNumber()
				}
			}
			g.Go(func() error {
				ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
				logEntry := log.WithField("appliance", i.GetName())
				logEntry.Info("checking if ready")
				var t *tui.Tracker
				if !opts.ciMode {
					t = p.AddTracker(i.GetName(), "upgraded")
					go t.Watch(appliancepkg.StatReady, []string{appliancepkg.UpgradeStatusFailed})
				}
				if err := a.UpgradeComplete(ctx, i.GetId(), SwitchPartition); err != nil {
					return err
				}
				logEntry.Info("install the downloaded upgrade image to the other partition")
				if !SwitchPartition {
					if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, i, []string{appliancepkg.UpgradeStatusSuccess}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
						return err
					}
					status, err := a.UpgradeStatus(ctx, i.GetId())
					if err != nil {
						return err
					}
					if regex.MatchString(status.GetDetails()) {
						if err := a.UpgradeSwitchPartition(ctx, i.GetId()); err != nil {
							return err
						}
						log.WithField("appliance", i.GetName()).Info("Switching partition")
					}
				}
				if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, i, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
					return err
				}
				if err := a.ApplianceStats.WaitForApplianceState(ctx, i, appliancepkg.StatReady, t); err != nil {
					return err
				}

				s, _, err := a.Stats(ctx)
				if err != nil {
					return err
				}

				// Check if partition has been switched
				for _, appData := range s.GetData() {
					if i.GetId() == appData.GetId() && appData.GetVolumeNumber() == initialVolume {
						return fmt.Errorf("upgrade failed on %s: never switched partition", i.GetName())
					}
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
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
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading additional controllers:\n", time.Now().Format(time.RFC3339))

		upgradeAdditionalController := func(ctx context.Context, controller openapi.Appliance, disable bool, p *tui.Progress) error {
			log.Infof("Upgrading controller %s", controller.GetName())
			ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
			defer cancel()

			var initialVolume float32
			for _, appData := range initialStats.GetData() {
				if controller.GetId() == appData.GetId() {
					initialVolume = appData.GetVolumeNumber()
				}
			}

			var t *tui.Tracker
			if !opts.ciMode && p != nil {
				t = p.AddTracker(controller.GetName(), "upgraded")
				go t.Watch(appliancepkg.StatReady, []string{appliancepkg.UpgradeStatusFailed})
			}
			if err := a.UpgradeComplete(ctx, controller.GetId(), true); err != nil {
				return err
			}
			if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, controller, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
				log.WithFields(f).WithError(err).Error("Controller never reached desired upgrade status")
				return err
			}
			if disable {
				log.WithField("appliance", controller.GetName()).Info("re-enabling controller")
				if err := backoffEnableController(controller); err != nil {
					log.WithFields(f).WithError(err).Error("Failed to enable controller")
					if merr, ok := err.(*multierror.Error); ok {
						var mutliErr error
						for _, e := range merr.Errors {
							mutliErr = multierror.Append(e)
						}
						mutliErr = multierror.Append(fmt.Errorf("could not enable controller on %s", controller.GetName()))
						return mutliErr
					}
					return err
				}
			}
			if cfg.Version >= 15 {
				if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, t); err != nil {
					return err
				}
				changeID, err := a.DisableMaintenanceMode(ctx, controller.GetId())
				if err != nil {
					return err
				}
				if _, err = ac.RetryUntilCompleted(ctx, changeID, controller.GetId()); err != nil {
					return err
				}
				log.WithFields(f).Info("Disabled maintenance mode")
				if err := a.ApplianceStats.WaitForApplianceStatus(ctx, controller, appliancepkg.StatusNotBusy, t); err != nil {
					return err
				}
			} else {
				if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, t); err != nil {
					log.WithFields(f).WithError(err).Error("Controller never reached desired state")
					return err
				}
			}
			s, _, err := a.Stats(ctx)
			if err != nil {
				return err
			}

			// Check if partition has been switched
			for _, appData := range s.GetData() {
				if controller.GetId() == appData.GetId() && appData.GetVolumeNumber() == initialVolume {
					return fmt.Errorf("upgrade failed on %s: never switched partition", controller.GetName())
				}
			}

			log.Infof("Upgraded controller %s", controller.GetName())
			return nil
		}
		for _, ctrl := range additionalControllers {
			var additionalControllerBars *tui.Progress
			if !opts.ciMode {
				additionalControllerBars = tui.New(ctx, spinnerOut)

			}
			if err := upgradeAdditionalController(ctx, ctrl, disableAdditionalControllers, additionalControllerBars); err != nil {
				return err
			}
			if !opts.ciMode {
				additionalControllerBars.Wait()
			}
		}
		log.Info("done waiting for additional controllers upgrade")
	}

	if len(logForwardersAndServers) > 0 {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading LogForwarder/LogServer appliances:\n", time.Now().Format(time.RFC3339))
		if err := batchUpgrade(ctx, logForwardersAndServers, false); err != nil {
			return err
		}
	}

	for index, chunk := range chunks {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading additional appliances (Batch %d / %d):\n", time.Now().Format(time.RFC3339), index+1, chunkLength)
		if err := batchUpgrade(ctx, chunk, false); err != nil {
			return fmt.Errorf("failed during upgrade of additional appliances %w", err)
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
	fmt.Fprintf(opts.Out, "\n[%s] %s\n", time.Now().Format(time.RFC3339), postSummary)

	return nil
}

func printCompleteSummary(out io.Writer, primaryController *openapi.Appliance, additionalControllers, logForwardersServers []openapi.Appliance, chunks [][]openapi.Appliance, skipped, backup []openapi.Appliance, backupDestination string, toVersion *version.Version) (string, error) {
	var (
		completeSummaryTpl = `
UPGRADE COMPLETE SUMMARY{{ if .Version }}

Appliances will be upgraded to version {{ .Version }}{{ end }}

Upgrade will be completed in steps:
{{ range $i, $s := .Steps }}
 {{ sum $i 1 }}. {{ $s.Description }}

{{ $s.TableString }}
{{ end }}{{ if .Skipped }}
Appliances that will be skipped:{{ range .Skipped }}
  - {{ . }}{{ end }}
{{ end }}`
		DescriptionIndent = "\n    "
		BackupDescription = []string{
			"Backup will be performed on the selected appliances",
			fmt.Sprintf("and downloaded to %s:", backupDestination),
		}
		PrimaryControllerDescription = []string{
			"The primary controller will be upgraded.",
			"This will result in the API being unreachable while completing the primary controller upgrade.",
		}
		AdditionalControllerDescription = []string{
			"Additional controllers will be upgraded.",
			"In some cases, the controller function on additional controllers will need to be disabled",
			"before proceeding with the upgrade. The disabled controllers will then be re-enabled once",
			"the upgrade is completed.",
			"This step will also reboot the upgraded controllers for the upgrade to take effect.",
		}
		LogForwardersAndServersDescription = []string{
			"Appliances with LogForwarder/LogServer functions are updated",
			"Other appliances need a connection to to these appliances for logging.",
		}
		AdditionalAppliancesDescription = []string{
			"Additional appliances will be upgraded. The additional appliances will be split into",
			"batches to keep the collective as available as possible during the upgrade process.",
			"Some of the additional appliances may need to be rebooted for the upgrade to take effect.",
		}
	)
	type step struct {
		Description string
		TableString string
	}

	type tplStub struct {
		Steps   []step
		Skipped []string
		Version string
	}

	tplSteps := []step{}

	if len(backup) > 0 {
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		for _, a := range backup {
			t.AddLine(fmt.Sprintf("- %s", a.GetName()))
		}
		t.Print()
		tplSteps = append(tplSteps, step{
			Description: strings.Join(BackupDescription, DescriptionIndent),
			TableString: util.PrefixStringLines(tb.String(), " ", 4),
		})
	}

	if primaryController != nil {
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		t.AddLine(fmt.Sprintf("- %s", primaryController.GetName()))
		t.Print()
		tplSteps = append(tplSteps, step{
			Description: strings.Join(PrimaryControllerDescription, DescriptionIndent),
			TableString: util.PrefixStringLines(tb.String(), " ", 4),
		})
	}

	if len(additionalControllers) > 0 {
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		for _, a := range additionalControllers {
			t.AddLine(fmt.Sprintf("- %s", a.GetName()))
		}
		t.Print()
		tplSteps = append(tplSteps, step{
			Description: strings.Join(AdditionalControllerDescription, DescriptionIndent),
			TableString: util.PrefixStringLines(tb.String(), " ", 4),
		})
	}

	if len(logForwardersServers) > 0 {
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		for _, a := range logForwardersServers {
			t.AddLine(fmt.Sprintf("- %s", a.GetName()))
		}
		t.Print()
		tplSteps = append(tplSteps, step{
			Description: strings.Join(LogForwardersAndServersDescription, DescriptionIndent),
			TableString: util.PrefixStringLines(tb.String(), " ", 4),
		})
	}

	if len(chunks) > 0 {
		tb := &bytes.Buffer{}
		for i, c := range chunks {
			fmt.Fprintf(tb, "Batch #%d:\n", i+1)
			t := util.NewPrinter(tb, 4)
			for _, a := range c {
				t.AddLine(fmt.Sprintf("- %s", a.GetName()))
			}
			t.Print()
		}
		tplSteps = append(tplSteps, step{
			Description: strings.Join(AdditionalAppliancesDescription, DescriptionIndent),
			TableString: util.PrefixStringLines(tb.String(), " ", 4),
		})
	}

	toSkip := []string{}
	for _, a := range skipped {
		toSkip = append(toSkip, a.GetName())
	}
	tplData := tplStub{
		Steps:   tplSteps,
		Skipped: toSkip,
	}
	if toVersion != nil {
		tplData.Version = toVersion.String()
	}
	t := template.Must(template.New("").Funcs(util.TPLFuncMap).Parse(completeSummaryTpl))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, tplData); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func printPostCompleteSummary(applianceVersions map[string]string, hasDiff bool) (string, error) {
	keys := make([]string, 0, len(applianceVersions))
	for k := range applianceVersions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	type tplStub struct {
		VersionTable string
		HasDiff      bool
	}

	tpl := `UPGRADE COMPLETE

{{ .VersionTable }}{{ if .HasDiff }}
WARNING: Upgrade was completed, but not all appliances are running the same version.{{ end }}
`

	tb := &bytes.Buffer{}
	tp := util.NewPrinter(tb, 4)
	tp.AddHeader("Appliance", "Upgraded to")
	for _, k := range keys {
		tp.AddLine(k, applianceVersions[k])
	}
	tp.Print()

	tplData := tplStub{
		VersionTable: tb.String(),
		HasDiff:      hasDiff,
	}
	t := template.Must(template.New("").Parse(tpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, tplData); err != nil {
		return "", err
	}
	return buf.String(), nil
}
