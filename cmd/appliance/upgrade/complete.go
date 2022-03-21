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
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
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
	Appliance         func(c *configuration.Config) (*appliancepkg.Appliance, error)
	debug             bool
	backup            bool
	backupDestination string
	backupAll         string
	NoInteractive     bool
	Timeout           time.Duration
}

// NewUpgradeCompleteCmd return a new upgrade status command
func NewUpgradeCompleteCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeCompleteOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
		Timeout:   DefaultTimeout,
	}
	var upgradeCompleteCmd = &cobra.Command{
		Use:   "complete",
		Short: "upgrade complete",
		Long: `Complete a prepared upgrade.
Install a prepared upgrade on the secondary partition
and perform a reboot to make the second partition the primary.`,
		Example: `# complete all pending upgrades
$ sdpctl appliance upgrade complete

# backup primary controller before completing
$ sdpctl appliance upgrade complete --backup

# backup to custom directory when completing pending upgrade
$ sdpctl appliance upgrade complete --backup --backup-destination=/path/to/custom/destination`,
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
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCompleteRun(c, args, &opts)
		},
	}

	flags := upgradeCompleteCmd.Flags()
	flags.BoolVarP(&opts.backup, "backup", "b", opts.backup, "backup primary controller before completing upgrade")
	flags.StringVar(&opts.backupDestination, "backup-destination", appliancepkg.DefaultBackupDestination, "specify path to download backup")

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
	host, err := opts.Config.GetHost()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	filter := util.ParseFilteringFlags(cmd.Flags())
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

			toBackup, err = appliancepkg.BackupPrompt(rawAppliances)
			if err != nil {
				return err
			}
		}
	}

	primaryController, err := appliancepkg.FindPrimaryController(rawAppliances, host)
	if err != nil {
		return err
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
	addtitionalControllers := groups[appliancepkg.FunctionController]
	primaryControllerUpgradeStatus, err := a.UpgradeStatus(ctx, primaryController.GetId())
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get upgrade status")
		return err
	}
	newVersion, err := appliancepkg.GetVersion(primaryControllerUpgradeStatus.GetDetails())
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to determine upgrade version")
	}

	msg, err := printCompleteSummary(opts.Out, append(appliances, *primaryController), offline, toBackup, newVersion)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Out, msg)
	if !opts.NoInteractive {
		if err = prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	if opts.backup {
		bOpts := appliancepkg.BackupOpts{
			Config:        opts.Config,
			Appliance:     opts.Appliance,
			Destination:   opts.backupDestination,
			AllFlag:       false,
			Timeout:       5 * time.Minute,
			Out:           opts.Out,
			NoInteractive: opts.NoInteractive,
			Quiet:         true,
		}
		if opts.backupAll == "all" {
			bOpts.AllFlag = true
		}
		if len(toBackup) > 0 {
			ids := []string{}
			for _, t := range toBackup {
				ids = append(ids, t.GetId())
			}
			bOpts.FilterFlag = map[string]map[string]string{
				"filter": {
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
	disableAdditionalControllers := appliancepkg.ShouldDisable(currentPrimaryControllerVersion, newVersion)
	if disableAdditionalControllers {
		fmt.Fprint(opts.Out, "\nDisabling controllers:\n")
		p := mpb.New(mpb.WithOutput(opts.Out), mpb.WithWidth(1))
		for _, controller := range addtitionalControllers {
			spinner := util.AddDefaultSpinner(p, controller.GetName(), "disabling", "disabled")
			f := log.Fields{"appliance": controller.GetName()}
			log.WithFields(f).Info("Disabling controller function")
			if err := a.DisableController(ctx, controller.GetId(), controller); err != nil {
				spinner.Abort(true)
				log.WithFields(f).Error("Unable to disable controller")
				return err
			}
			if err := a.ApplianceStats.WaitForState(ctx, controller, "appliance_ready"); err != nil {
				spinner.Abort(true)
				log.WithFields(f).Error("never reached desired state")
				return err
			}
		}
		p.Wait()
	}

	// verify the state for all controller
	p := mpb.New(mpb.WithWidth(1), mpb.WithOutput(opts.Out))
	spinner := util.AddDefaultSpinner(p, "verifying initial states", "waiting", "ready")
	state := "controller_ready"
	if cfg.Version < 15 {
		state = "single_controller_ready"
	}
	if err := a.ApplianceStats.WaitForState(ctx, *primaryController, state); err != nil {
		spinner.Abort(true)
		return fmt.Errorf("primary controller %s", err)
	}
	spinner.Increment()
	log.Info("all controllers are in correct state")
	p.Wait()

	if cfg.Version >= 15 && len(addtitionalControllers) > 0 {
		for _, controller := range addtitionalControllers {
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
		log.Errorf("appliance %s is not ready for upgrade", strings.Join(notReady, ", "))
		return fmt.Errorf("one or more appliances are not ready for upgrade.")
	}
	if primaryControllerUpgradeStatus.GetStatus() == appliancepkg.UpgradeStatusReady {
		fmt.Fprint(opts.Out, "\nUpgrading primary controller:\n")
		p := mpb.New(mpb.WithOutput(opts.Out), mpb.WithWidth(1))
		statusReport := make(chan string)
		a.UpgradeStatusWorker.Watch(ctx, p, *primaryController, appliancepkg.UpgradeStatusIdle, statusReport)
		log.WithField("appliance", primaryController.GetName()).Info("Completing upgrade and switching partition")
		if err := a.UpgradeComplete(ctx, primaryController.GetId(), true); err != nil {
			return err
		}
		log.WithField("appliance", primaryController.GetName()).Infof("Waiting for primary controller to reach state %s", state)
		if err := a.UpgradeStatusWorker.Wait(ctx, *primaryController, appliancepkg.UpgradeStatusIdle, statusReport); err != nil {
			return err
		}
		close(statusReport)
		log.WithField("appliance", primaryController.GetName()).Info("Primary controller updated")
		p.Wait()
	}

	batchUpgrade := func(ctx context.Context, appliances []openapi.Appliance, SwitchPartition bool) error {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		upgradeChan := make(chan openapi.Appliance, len(appliances))
		p := mpb.New(mpb.WithOutput(opts.Out), mpb.WithWidth(1))
		for _, appliance := range appliances {
			i := appliance
			g.Go(func() error {
				log.WithField("appliance", i.GetName()).Info("checking if ready")
				statusReport := make(chan string)
				defer close(statusReport)
				a.UpgradeStatusWorker.Watch(ctx, p, i, appliancepkg.UpgradeStatusIdle, statusReport)
				if err := a.UpgradeStatusWorker.Wait(ctx, i, appliancepkg.UpgradeStatusReady, statusReport); err != nil {
					return err
				}
				if err := a.UpgradeComplete(ctx, i.GetId(), SwitchPartition); err != nil {
					return err
				}
				if !SwitchPartition {
					if err := a.UpgradeStatusWorker.Wait(ctx, i, appliancepkg.UpgradeStatusSuccess, statusReport); err != nil {
						return err
					}
					status, err := a.UpgradeStatus(ctx, i.GetId())
					if err != nil {
						return err
					}
					regex := regexp.MustCompile(`a reboot is required for the upgrade to go into effect`)
					if regex.MatchString(status.GetDetails()) {
						if err := a.UpgradeSwitchPartition(ctx, i.GetId()); err != nil {
							return err
						}
						log.WithField("appliance", i.GetName()).Info("Switching partition")
					}
				}
				if err := a.UpgradeStatusWorker.Wait(ctx, i, appliancepkg.UpgradeStatusIdle, statusReport); err != nil {
					return err
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
			return fmt.Errorf("Error during upgrade of an appliance %w", err)
		}
		p.Wait()
		return nil
	}

	if len(addtitionalControllers) > 0 {
		fmt.Fprint(opts.Out, "\nUpgrading additional controllers:\n")
		if err := batchUpgrade(ctx, addtitionalControllers, true); err != nil {
			return fmt.Errorf("failed during upgrade of additional controllers %w", err)
		}
		log.Info("done waiting for additional controllers upgrade")
	}

	ctrlUpgradeState := "controller_ready"
	if cfg.Version < 15 {
		ctrlUpgradeState = "multi_controller_ready"
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

	// re-enable additional controllers sequentially, one at the time
	if disableAdditionalControllers {
		fmt.Fprint(opts.Out, "\nRe-enabling controllers:\n")
		p := mpb.New(mpb.WithOutput(opts.Out), mpb.WithWidth(1))
		for _, controller := range addtitionalControllers {
			spinner := util.AddDefaultSpinner(p, controller.GetName(), "enabling", "enabled")
			f := log.Fields{"controller": controller.GetName()}
			log.WithFields(f).Info("Enabling controller function")
			if err := backoffEnableController(controller); err != nil {
				spinner.Abort(true)
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
			if err := a.ApplianceStats.WaitForState(ctx, controller, ctrlUpgradeState); err != nil {
				spinner.Abort(true)
				log.WithFields(f).WithError(err).Error("Controller never reached desired state")
				return err
			}
			if cfg.Version >= 15 {
				_, err := a.DisableMaintenanceMode(ctx, controller.GetId())
				if err != nil {
					return err
				}
				log.WithFields(f).Info("Disabled maintenance mode")
			}
			spinner.Increment()
		}
		p.Wait()
	}

	readyForUpgrade, err := a.UpgradeStatusMap(ctx, appliances)
	if err != nil {
		return err
	}

	additionalAppliances := make([]openapi.Appliance, 0)
	for id, result := range readyForUpgrade {
		for _, appliance := range appliances {
			if result.Status == appliancepkg.UpgradeStatusReady {
				if id == appliance.GetId() {
					additionalAppliances = append(additionalAppliances, appliance)
				}
			}
		}

	}

	if len(additionalAppliances) > 0 {
		fmt.Fprint(opts.Out, "\nUpgrading additional appliances:\n")

		if err := batchUpgrade(ctx, additionalAppliances, false); err != nil {
			return fmt.Errorf("failed during upgrade of additional appliances %w", err)
		}
	}
	fmt.Fprint(opts.Out, "Upgrade complete!\n")

	return nil
}

func printCompleteSummary(out io.Writer, upgradeable, skipped, backup []openapi.Appliance, toVersion *version.Version) (string, error) {
	type tplStub struct {
		Upgradeable []string
		Skipped     []string
		Backup      []string
		Version     string
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

 3. The remaining appliances will be upgraded.
    Some of the additional appliances may need to be rebooted for the upgrade to take effect.

{{ if .Version -}}
The following appliances will be upgraded to version {{ .Version }}:
{{- end}}
{{- range .Upgradeable }}
 - {{ . -}}
{{ end }}
{{ with .Skipped }}
Appliances that will be skipped:
{{- range . }}
 - {{ . -}}
{{- end }}
{{ end -}}
{{ with .Backup }}
Appliances that will be backed up before completing upgrade:
{{- range . }}
 - {{ . -}}
{{- end }}
{{ end }}
`
	toUpgrade := []string{}
	for _, a := range upgradeable {
		toUpgrade = append(toUpgrade, a.GetName())
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
		Upgradeable: toUpgrade,
		Skipped:     toSkip,
		Backup:      toBackup,
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
