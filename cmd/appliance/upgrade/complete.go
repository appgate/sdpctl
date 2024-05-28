package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/api"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/cmdutil"
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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
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
			configuration.NeedUpdateAPIConfig: "true",
		},
		Args: cobra.ExactArgs(0),
		PreRunE: func(cmd *cobra.Command, args []string) error {
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
	flags.BoolVarP(&opts.backup, "backup", "b", opts.backup, "Backup primary Controller before completing the upgrade")
	flags.StringVar(&opts.backupDestination, "backup-destination", "$HOME/Downloads/appgate/backup", "Specify path to download backup")
	flags.StringVar(&opts.actualHostname, "actual-hostname", "", "If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname")
	flags.IntVar(&opts.batchSize, "batch-size", 2, "Number of batch groups")
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
	ctx = context.WithValue(ctx, appliancepkg.Caller, cmd.CalledAs())
	filter, orderBy, descending := util.ParseFilteringFlags(cmd.Flags(), opts.defaultFilter)
	rawAppliances, err := a.List(ctx, nil, orderBy, descending)
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
	initialStats, _, err := a.Stats(ctx, nil, orderBy, descending)
	if err != nil {
		return err
	}
	plan, err := appliancepkg.NewUpgradePlan(rawAppliances, *initialStats, controlHost, filter, orderBy, descending)
	if err != nil {
		return err
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
	backupIds := make([]string, 0, len(toBackup))
	for _, a := range toBackup {
		backupIds = append(backupIds, a.GetId())
	}
	if err := plan.AddBackups(backupIds); err != nil {
		return err
	}

	if plan.NothingToUpgrade() {
		var errs *multierror.Error
		errs = multierror.Append(errs, fmt.Errorf("No appliances are ready to upgrade. Please run 'upgrade prepare' before trying to complete an upgrade"))
		for _, s := range plan.Skipping {
			errs = multierror.Append(errs, s)
		}
		return errs
	}

	if err := plan.PrintPreCompleteSummary(opts.Out); err != nil {
		return err
	}
	if !opts.NoInteractive {
		if err = prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	if opts.backup {
		if len(plan.BackupIds) > 0 {
			bOpts.FilterFlag = map[string]map[string]string{
				"include": {
					"id": strings.Join(plan.BackupIds, appliancepkg.FilterDelimiter),
				},
			}
		}
		fmt.Fprintf(opts.Out, "\n[%s] Backing up:\n", time.Now().Format(time.RFC3339))
		if err := appliancepkg.PrepareBackup(&bOpts); err != nil {
			log.WithError(err).Error("backup failed")
			return err
		}
		backupMap, err := appliancepkg.PerformBackup(cmd, args, &bOpts)
		if err != nil {
			log.WithError(err).Error("backup failed")
			return err
		}
		if err := appliancepkg.CleanupBackup(&bOpts, backupMap); err != nil {
			log.WithError(err).Error("backup cleanup failed")
			return err
		}
	}

	// 1. Disable Controller function on the following appliance
	// we will run this sequencelly, since this is a sensitive operation
	// so that we can leave the Collective gracefully.
	fmt.Fprintf(opts.Out, "\n[%s] Initializing upgrade:\n", time.Now().Format(time.RFC3339))
	initP := mpb.NewWithContext(ctx, mpb.WithOutput(spinnerOut))
	for _, controller := range plan.Controllers {
		spinner := tui.AddDefaultSpinner(initP, controller.GetName(), "disabling", "disabled")
		f := log.Fields{"appliance": controller.GetName()}
		log.WithFields(f).Info("Disabling controller function")
		if err := a.DisableController(ctx, controller.GetId(), controller); err != nil {
			spinner.Abort(false)
			log.WithFields(f).Error("Unable to disable the Controller")
			return err
		}
		if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, nil); err != nil {
			spinner.Abort(false)
			log.WithFields(f).Error("Never reached desired state")
			return err
		}
		spinner.Increment()
	}

	// verify the state for all Controllers
	verifyingSpinner := tui.AddDefaultSpinner(initP, "verifying states", "verifying", "ready")
	if err := a.ApplianceStats.WaitForApplianceState(ctx, *primaryController, appliancepkg.StatReady, nil); err != nil {
		verifyingSpinner.Abort(false)
		return fmt.Errorf("the primary Controller %s", err)
	}
	log.Info("All Controllers are in the correct state")

	var wg sync.WaitGroup
	errChan := make(chan error)
	for _, controller := range plan.Controllers {
		wg.Add(1)
		go func(wg *sync.WaitGroup, errChan chan error) {
			defer wg.Done()
			f := log.Fields{"controller": controller.GetName()}
			log.WithFields(f).Info("Enabling the maintenance mode")
			id, err := a.EnableMaintenanceMode(ctx, controller.GetId())
			if err != nil {
				log.WithFields(f).Warnf("Unable to enable the maintenance mode %s", err)
				errChan <- err
				return
			}
			log.WithFields(f).Infof("id %s", id)
			if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, controller, []string{appliancepkg.UpgradeStatusReady, appliancepkg.UpgradeStatusSuccess}, []string{appliancepkg.UpgradeStatusIdle}, nil); err != nil {
				log.WithFields(f).Warnf("Controller not ready %s", err)
				errChan <- err
				return
			}
		}(&wg, errChan)
	}

	go func(wg *sync.WaitGroup, errChan chan error) {
		wg.Wait()
		close(errChan)
	}(&wg, errChan)

	var errs *multierror.Error
	for e := range errChan {
		if e != nil {
			errs = multierror.Append(errs, e)
		}
	}
	if errs != nil {
		return errs.ErrorOrNil()
	}

	verifyingSpinner.Increment()
	initP.Wait()

	if plan.PrimaryController != nil {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading the primary Controller:\n", time.Now().Format(time.RFC3339))
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
				t = primaryControllerBars.AddTracker(controller.GetName(), "waiting", "upgraded")
				go t.Watch(appliancepkg.StatReady, []string{appliancepkg.UpgradeStatusFailed})
			}

			logEntry := log.WithFields(log.Fields{
				"appliance": controller.GetName(),
				"url":       cfg.URL,
			})
			ips, err := network.ResolveHostnameIPs(cfg.URL)
			if err != nil {
				logEntry.WithError(err).Error("failed to lookup hostname ips")
			}
			logEntry.Info("Completing upgrade and switching partition")
			if err := a.UpgradeComplete(ctx, controller.GetId(), true); err != nil {
				return err
			}
			msg := "Upgrading primary Controller, installing and rebooting..."
			logEntry.WithField("want", appliancepkg.StatReady).Info(msg)
			if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(context.WithValue(ctx, appliancepkg.PrimaryUpgrade, true), controller, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
				if errors.Is(err, cmdutil.ErrControllerMaintenanceMode) {
					if ips != nil {
						postIPs, _ := network.ResolveHostnameIPs(cfg.URL)
						cmpResult := slices.Equal(ips, postIPs)
						if !cmpResult {
							logEntry.WithError(fmt.Errorf("hostname resolves to different ip")).WithFields(log.Fields{
								"original_resolution": ips,
								"current_resolution":  postIPs,
							}).Error("changed hostname resolution detected")
						}
					}
					return fmt.Errorf("possible primary controller redirection detected: %w", err)
				}
				return err
			}
			if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, t); err != nil {
				return err
			}
			s, _, err := a.Stats(ctx, nil, orderBy, descending)
			if err != nil {
				return err
			}

			// Check if partition has been switched
			for _, appData := range s.GetData() {
				if controller.GetId() == appData.GetId() && appData.GetVolumeNumber() == initialVolume {
					return fmt.Errorf("Upgrade failed on %s: never switched partition", controller.GetName())
				}
			}

			logEntry.Info("The primary Controller updated")
			return nil
		}
		if err := upgradeReadyPrimary(ctx, *plan.PrimaryController); err != nil {
			return err
		}
	}

	batchUpgrade := func(ctx context.Context, appliances []openapi.Appliance, SwitchPartition bool) error {
		g, ctx := errgroup.WithContext(ctx)
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
					t = p.AddTracker(i.GetName(), "waiting", "upgraded")
					go t.Watch(appliancepkg.StatReady, []string{appliancepkg.UpgradeStatusFailed})
				}
				if !SwitchPartition {
					err := backoff.Retry(func() error {
						err := a.UpgradeComplete(ctx, i.GetId(), SwitchPartition)
						if err != nil {
							logEntry.Warnf("Attempting to run upgrade complete %s", err)
							return err
						}
						return nil
					}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
					if err != nil {
						return fmt.Errorf("Could not complete upgrade on %s %w", i.GetName(), err)
					}
				}
				logEntry.Info("Install the downloaded upgrade image to the other partition")
				if !SwitchPartition {
					if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, i, []string{appliancepkg.UpgradeStatusSuccess}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
						return fmt.Errorf("%s %w", i.GetName(), err)
					}
					status, err := a.UpgradeStatusRetry(ctx, i.GetId())
					if err != nil {
						return fmt.Errorf("%s %w", i.GetName(), err)
					}
					if status.GetStatus() == appliancepkg.UpgradeStatusSuccess {
						if err := a.UpgradeSwitchPartition(ctx, i.GetId()); err != nil {
							return fmt.Errorf("%s %w", i.GetName(), err)
						}
						log.WithField("appliance", i.GetName()).Info("Switching partition")
					}
				}
				if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(ctx, i, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
					return fmt.Errorf("%s %w", i.GetName(), err)
				}
				if err := a.ApplianceStats.WaitForApplianceState(ctx, i, appliancepkg.StatReady, t); err != nil {
					return fmt.Errorf("%s %w", i.GetName(), err)
				}

				s, _, err := a.Stats(ctx, nil, orderBy, descending)
				if err != nil {
					return err
				}

				// Check if partition has been switched
				for _, appData := range s.GetData() {
					if i.GetId() == appData.GetId() && appData.GetVolumeNumber() == initialVolume {
						return fmt.Errorf("Upgrade failed on %s: never switched partition", i.GetName())
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
			if ae, ok := err.(*api.Error); ok {
				for _, e := range ae.Errors {
					log.Error(e)
				}
			} else {
				log.Error(err)
			}

			return err
		}
		return nil
	}

	if len(plan.Controllers) > 0 {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading additional Controllers:\n", time.Now().Format(time.RFC3339))

		upgradeAdditionalController := func(ctx context.Context, controller openapi.Appliance, p *tui.Progress) error {
			log.Infof("Upgrading the Controller %s", controller.GetName())
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
				t = p.AddTracker(controller.GetName(), "waiting", "upgraded")
				go t.Watch(appliancepkg.StatReady, []string{appliancepkg.UpgradeStatusFailed})
			}
			err := backoff.Retry(func() error {
				return a.UpgradeComplete(ctx, controller.GetId(), true)
			}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
			if err != nil {
				return err
			}
			if err := a.UpgradeStatusWorker.WaitForUpgradeStatus(context.WithValue(ctx, appliancepkg.PrimaryUpgrade, "applying upgrade"), controller, []string{appliancepkg.UpgradeStatusIdle}, []string{appliancepkg.UpgradeStatusFailed}, t); err != nil {
				log.WithField("appliance", controller.GetName()).WithError(err).Error("The Controller never reached desired upgrade status")
				return err
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
				log.WithField("appliance", controller.GetName()).Info("Disabled the maintenance mode")
				if err := a.ApplianceStats.WaitForApplianceStatus(ctx, controller, appliancepkg.StatusNotBusy, t); err != nil {
					return err
				}
			} else {
				if err := a.ApplianceStats.WaitForApplianceState(ctx, controller, appliancepkg.StatReady, t); err != nil {
					log.WithField("appliance", controller.GetName()).WithError(err).Error("The Controller never reached desired state")
					return err
				}
			}
			s, _, err := a.Stats(ctx, nil, orderBy, descending)
			if err != nil {
				return err
			}

			// Check if partition has been switched
			for _, appData := range s.GetData() {
				if controller.GetId() == appData.GetId() && appData.GetVolumeNumber() == initialVolume {
					return fmt.Errorf("Upgrade failed on %s: never switched partition", controller.GetName())
				}
			}

			log.Infof("Upgraded the Controller %s", controller.GetName())
			return nil
		}
		for _, ctrl := range plan.Controllers {
			var additionalControllerBars *tui.Progress
			if !opts.ciMode {
				additionalControllerBars = tui.New(ctx, spinnerOut)

			}
			if err := upgradeAdditionalController(ctx, ctrl, additionalControllerBars); err != nil {
				return err
			}
			if !opts.ciMode {
				additionalControllerBars.Wait()
			}
		}
		log.Info("Done waiting for the additional Controllers upgrade")
	}

	if len(plan.LogForwardersAndServers) > 0 {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading LogForwarder/LogServer appliances:\n", time.Now().Format(time.RFC3339))
		if err := batchUpgrade(ctx, plan.LogForwardersAndServers, false); err != nil {
			return err
		}
	}

	for index, chunk := range plan.Batches {
		fmt.Fprintf(opts.Out, "\n[%s] Upgrading additional appliances (Batch %d / %d):\n", time.Now().Format(time.RFC3339), index+1, len(plan.Batches))
		if err := batchUpgrade(ctx, chunk, false); err != nil {
			return err
		}
	}

	// Trigger ZTP version update if needed
	// From v18 and up
	// This step is not fatal, so we only log errors here
	if opts.Config.Version >= 18 {
		ztpStatus, err := a.ZTPStatus(ctx)
		if err != nil {
			log.WithError(err).Warn("failed to get ZTP registered status")
		}
		if ztpRegistered, ok := ztpStatus.GetRegisteredOk(); err == nil && ok {
			if isRegistered := *ztpRegistered; isRegistered {
				if _, err := a.ZTPUpdateNotify(ctx); err != nil {
					log.WithError(err).Warn("failed to trigger ZTP update")
				}
			}
		}
	}

	// Clean out logserver bundle if it exists in file-repository
	if files, err := a.ListFiles(ctx, []string{}, false); err == nil {
		regex := regexp.MustCompile(`^logserver-\d+\.\d+\.zip$`)
		for _, f := range files {
			match := regex.MatchString(f.GetName())
			if !match {
				continue
			}
			if err := a.DeleteFile(ctx, f.GetName()); err != nil {
				log.WithError(err).Warn("failed to remove logserver bundle file from controller file repository")
			}
		}
	} else {
		log.WithError(err).Warn("failed to list files in file repository")
	}

	// Get new stats for post complete summary
	newStats, _, err := a.Stats(ctx, nil, orderBy, descending)
	if err != nil {
		return err
	}
	log.Info("upgrade complete")
	return plan.PrintPostCompleteSummary(opts.Out, newStats.GetData())
}
