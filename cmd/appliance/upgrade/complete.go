package upgrade

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	appliancepkg "github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/configuration"
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/appgate/appgatectl/pkg/prompt"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type upgradeCompleteOptions struct {
	Config    *configuration.Config
	Out       io.Writer
	Appliance func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Token     string
	Timeout   int
	url       string
	provider  string
	debug     bool
	insecure  bool
	cacert    string
}

// NewUpgradeCompleteCmd return a new upgrade status command
func NewUpgradeCompleteCmd(f *factory.Factory) *cobra.Command {
	opts := upgradeCompleteOptions{
		Config:    f.Config,
		Appliance: f.Appliance,
		Timeout:   10,
		debug:     f.Config.Debug,
		Out:       f.IOOutWriter,
	}
	var upgradeCompleteCmd = &cobra.Command{
		Use:   "complete",
		Short: "upgrade complete",
		Long: `Complete a prepared upgrade.
Install a prepared upgrade on the secondary partition
and perform a reboot to make the second partition the primary.`,
		RunE: func(c *cobra.Command, args []string) error {
			return upgradeCompleteRun(c, args, &opts)
		},
	}

	upgradeCompleteCmd.PersistentFlags().BoolVar(&opts.insecure, "insecure", true, "Whether server should be accessed without verifying the TLS certificate")
	upgradeCompleteCmd.PersistentFlags().StringVarP(&opts.url, "url", "u", f.Config.URL, "appgate sdp controller API URL")
	upgradeCompleteCmd.PersistentFlags().StringVarP(&opts.provider, "provider", "", "local", "identity provider")
	upgradeCompleteCmd.PersistentFlags().StringVarP(&opts.cacert, "cacert", "", "", "Path to the controller's CA cert file in PEM or DER format")

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
	u, err := url.Parse(opts.url)
	if err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Minute))
	defer cancel()
	allAppliances, err := a.GetAll(ctx)
	if err != nil {
		return err
	}
	initialStats, _, err := a.Stats(ctx)
	if err != nil {
		return err
	}
	appliances, offline, err := appliancepkg.FilterAvailable(allAppliances, initialStats.GetData())
	if err != nil {
		return err
	}
	for _, o := range offline {
		log.Warnf("%q is offline and will be excluded from upgrade.", o.GetName())
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
	primaryController, err := appliancepkg.FindPrimaryController(appliances, host)
	if err != nil {
		return err
	}
	log.Infof("Primary controller is: %q", primaryController.Name)
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

	// 1. Disable Controller function on the following appliance
	// we will run this sequencelly, since this is a sensitve operation
	// so that we can leave the collective gracefully.
	addtitionalControllers := groups[appliancepkg.FunctionController]
	for _, controller := range addtitionalControllers {
		log.WithFields(log.Fields{
			"controller": controller.GetName(),
		}).Info("Disabling controller function")
		if err := a.DisableController(ctx, controller.GetId(), controller); err != nil {
			// TODO
			return err
		}
		if err := a.ApplianceStats.WaitForState(ctx, []openapi.Appliance{controller}, "appliance_ready"); err != nil {
			// TODO
			return err
		}
	}
	log.Info("verify the state for all controller")
	// verify the state for all controller
	controllers := []openapi.Appliance{*primaryController}
	state := "controller_ready"
	if cfg.Version < 15 {
		state = "single_controller_ready"
	}

	if err := a.ApplianceStats.WaitForState(ctx, controllers, state); err != nil {
		// TODO
		return err
	}
	log.Info("all controllers are in correct state")

	if cfg.Version >= 15 && len(addtitionalControllers) > 0 {
		log.Info("Enabling maintenance mode on Controller")
		for _, controller := range addtitionalControllers {
			f := log.Fields{"controller": controller.GetName()}
			log.WithFields(f).Info("enabling maintenance mode")
			id, err := a.EnableMaintenanceMode(ctx, controller.GetId())
			if err != nil {
				log.WithFields(f).Warnf("Unable to enable maintenanc mode %s", err)
				return err
			}
			log.WithFields(f).Infof("id %s", id)
		}
	}
	m, err := a.UpgradeStatusMap(ctx, appliances)
	if err != nil {
		log.Errorf("Upgrade status failed %s", err)
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
	primaryControllerUpgradeStatus, err := a.UpgradeStatus(ctx, primaryController.GetId())
	if err != nil {
		return fmt.Errorf("Unable to retrieve primary controller upgrade status %w", err)
	}
	if primaryControllerUpgradeStatus.GetStatus() == appliancepkg.UpgradeStatusReady {
		log.Infof("Completing upgrade and switching partition on %s", primaryController.GetName())
		if err := a.UpgradeComplete(ctx, primaryController.GetId(), true); err != nil {
			return err
		}
		log.Infof("Wating for primary controller to come back online in state %s", state)
		if err := a.ApplianceStats.WaitForState(ctx, controllers, state); err != nil {
			return err
		}
		log.Infof("Primary controller updated")
	}

	batchUpgrade := func(ctx context.Context, appliances []openapi.Appliance, SwitchPartition bool) ([]openapi.Appliance, error) {
		g, ctx := errgroup.WithContext(context.Background())
		upgradeChan := make(chan openapi.Appliance, len(appliances))
		for _, appliance := range appliances {
			i := appliance
			g.Go(func() error {
				if err := a.UpgradeComplete(ctx, i.GetId(), SwitchPartition); err != nil {
					return err
				}
				log.WithField("appliance", i.GetName()).Info("Preformed UpgradeComplete")
				select {
				case upgradeChan <- i:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			})
		}
		go func() {
			g.Wait()
			close(upgradeChan)
		}()
		result := make([]openapi.Appliance, 0)
		for r := range upgradeChan {
			result = append(result, r)
		}
		if err := g.Wait(); err != nil {
			return nil, fmt.Errorf("Error during upgrade of an appliance %w", err)
		}
		return result, nil
	}
	upgradedAdditionalControllers, err := batchUpgrade(ctx, addtitionalControllers, true)
	if err != nil {
		return fmt.Errorf("failed during upgrade of additional controllers %w", err)
	}

	// blocking function; waiting for upgrade status to be idle
	if err := a.UpgradeStatusWorker.Wait(ctx, upgradedAdditionalControllers, appliancepkg.UpgradeStatusIdle); err != nil {
		return err
	}
	log.Info("done waiting for additional controllers upgrade")
	ctrlUpgradeState := "controller_ready"
	if cfg.Version < 15 {
		ctrlUpgradeState = "multi_controller_ready"
	}
	// re-enable additional controllers
	for _, controller := range addtitionalControllers {
		f := log.Fields{"controller": controller.GetName()}
		log.WithFields(f).Info("Enabling controller function")
		if err := a.EnableController(ctx, controller.GetId(), controller); err != nil {
			log.WithFields(f).Errorf("Unable to enable controller %s", err)
			return err
		}
		if err := a.ApplianceStats.WaitForState(ctx, []openapi.Appliance{controller}, ctrlUpgradeState); err != nil {
			log.WithFields(f).Errorf("Controller never got to desired state %s", err)
			return err
		}
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

	upgradedAppliances, err := batchUpgrade(ctx, additionalAppliances, false)
	if err != nil {
		return fmt.Errorf("failed during upgrade of additional appliances %w", err)
	}
	// blocking function; waiting for upgrade status to be idle
	if err := a.UpgradeStatusWorker.Wait(ctx, upgradedAppliances, appliancepkg.UpgradeStatusSuccess); err != nil {
		return err
	}
	switchBatch := func(ctx context.Context, appliances []openapi.Appliance) ([]openapi.Appliance, error) {
		g, ctx := errgroup.WithContext(context.Background())
		switchChan := make(chan openapi.Appliance, len(appliances))
		for _, appliance := range appliances {
			i := appliance
			g.Go(func() error {
				if err := a.UpgradeSwitchPartition(ctx, i.GetId()); err != nil {
					return err
				}
				log.WithField("appliance", i.GetName()).Info("Switching partition")
				select {
				case switchChan <- i:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			})
		}
		go func() {
			g.Wait()
			close(switchChan)
		}()
		result := make([]openapi.Appliance, 0)
		for r := range switchChan {
			result = append(result, r)
		}
		return result, g.Wait()
	}
	readyForSwitch, err := a.UpgradeStatusMap(ctx, appliances)
	if err != nil {
		return err
	}

	switchAppliances := make([]openapi.Appliance, 0)
	for id, result := range readyForSwitch {
		if result.Status == appliancepkg.UpgradeStatusSuccess {
			for _, appliance := range appliances {
				if id == appliance.GetId() {
					switchAppliances = append(switchAppliances, appliance)
				}
			}
		}
	}
	switchedAppliances, err := switchBatch(ctx, switchAppliances)
	if err != nil {
		return fmt.Errorf("failed during switch partition of additional appliances %w", err)
	}
	for _, a := range switchedAppliances {
		log.Infof("Upgraded %q with switched partition", a.GetName())
	}

	if err := a.UpgradeStatusWorker.Wait(ctx, switchedAppliances, appliancepkg.UpgradeStatusIdle); err != nil {
		return err
	}
	log.Info("Upgrade finished")
	return nil
}
