package appliance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/AlecAivazis/survey/v2"
	"github.com/appgate/sdp-api-client-go/api/v19/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/appliance/change"
	"github.com/appgate/sdpctl/pkg/cmdutil"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type cmdOpts struct {
	Config         *configuration.Config
	Appliance      func(c *configuration.Config) (*appliancepkg.Appliance, error)
	Out            io.Writer
	SpinnerOut     func() io.Writer
	NoInteractive  bool
	CiMode         bool
	ActualHostname string
}

func NewForceDisableControllerCmd(f *factory.Factory) *cobra.Command {
	opts := cmdOpts{
		Appliance:  f.Appliance,
		Config:     f.Config,
		Out:        f.IOOutWriter,
		SpinnerOut: f.GetSpinnerOutput(),
	}
	cmd := &cobra.Command{
		Use:         "force-disable-controller [hostname|ID...]",
		Short:       docs.ApplianceForceDisableControllerDocs.Short,
		Long:        docs.ApplianceForceDisableControllerDocs.Long,
		Example:     docs.ApplianceForceDisableControllerDocs.ExampleString(),
		Annotations: map[string]string{"MinAPIVersion": "18"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var errs *multierror.Error
			var err error

			if opts.NoInteractive, err = cmd.Flags().GetBool("no-interactive"); err != nil {
				errs = multierror.Append(errs, err)
			}

			if !f.CanPrompt() {
				opts.NoInteractive = true
			}

			if len(args) <= 0 && opts.NoInteractive {
				errs = multierror.Append(errs, errors.New("No arguments provided while running in no-interactive mode"))
			}

			if opts.CiMode, err = cmd.Flags().GetBool("ci-mode"); err != nil {
				errs = multierror.Append(errs, err)
			}

			if opts.ActualHostname, err = cmd.Flags().GetString("actual-hostname"); err != nil {
				errs = multierror.Append(errs, err)
			}

			return errs.ErrorOrNil()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return forceDisableControllerRunE(opts, args)
		},
	}
	cmd.Flags().StringVar(&opts.ActualHostname, "actual-hostname", "", "If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname")
	cmd.SetHelpFunc(cmdutil.HideIncludeExcludeFlags)

	return cmd
}

func forceDisableControllerRunE(opts cmdOpts, args []string) error {
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
	changeAPI := change.ApplianceChange{
		APIClient: a.APIClient,
		Token:     a.Token,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appliances, err := a.List(ctx, nil, []string{"name"}, false)
	if err != nil {
		return err
	}

	primaryHost, err := cfg.GetHost()
	if err != nil {
		return err
	}
	if len(opts.ActualHostname) > 0 {
		primaryHost = opts.ActualHostname
	}
	primaryController, err := appliancepkg.FindPrimaryController(appliances, primaryHost, true)
	if err != nil {
		return err
	}
	log.WithField("controller", primaryController.GetName()).Info("Found primary Controller")

	if util.InSlice(primaryController.GetHostname(), args) {
		return errors.New("Illegal operation. Disabling the primary Controller is not allowed")
	}

	rawControllers := appliancepkg.GroupByFunctions(appliances)[appliancepkg.FunctionController]
	// Remove primary Controller from list
	controllers := []openapi.Appliance{}
	for _, ctrl := range rawControllers {
		if ctrl.GetId() == primaryController.GetId() {
			continue
		}
		controllers = append(controllers, ctrl)
	}

	if len(controllers) <= 0 {
		return fmt.Errorf("No controllers to disable")
	}

	stats, _, err := a.Stats(ctx, nil, nil, false)
	if err != nil {
		return err
	}
	statData := stats.GetData()

	// Error is ignored here since the function returns an error when either a controller or logserver is offline, which is fine at this point
	controllers, offline, _ := appliancepkg.FilterAvailable(controllers, statData)

	// Sort slices by appliance name
	sort.SliceStable(controllers, func(i, j int) bool { return controllers[i].GetName() < controllers[j].GetName() })
	sort.SliceStable(offline, func(i, j int) bool { return offline[i].GetName() < offline[j].GetName() })
	sort.SliceStable(statData, func(i, j int) bool { return statData[i].GetName() < statData[j].GetName() })

	unselectedOffline := []string{}
	if len(args) <= 0 {
		selectable := []string{}
		preSelected := []string{}
		for _, ctrl := range controllers {
			selectable = append(selectable, fmt.Sprintf("%s (%s)", ctrl.GetName(), ctrl.GetHostname()))
		}
		for _, ctrl := range offline {
			selectableString := fmt.Sprintf("%s (%s) [OFFLINE]", ctrl.GetName(), ctrl.GetHostname())
			selectable = append(selectable, selectableString)
			preSelected = append(preSelected, selectableString)
		}
		sort.SliceStable(selectable, func(i, j int) bool { return selectable[i] < selectable[j] })
		qs := &survey.MultiSelect{
			PageSize: len(selectable),
			Message:  "Select Controllers to force disable",
			Options:  selectable,
			Default:  preSelected,
		}
		selected := []string{}
		if err := prompt.SurveyAskOne(qs, &selected); err != nil {
			return err
		}
		if len(selected) <= 0 {
			return errors.New("No Controllers selected to disable")
		}
		for _, s := range selectable {
			if !util.InSlice(s, selected) && strings.Contains(s, "[OFFLINE]") {
				unselectedOffline = append(unselectedOffline, s)
			}
		}
		for _, s := range selected {
			for _, ctrl := range controllers {
				if strings.Contains(s, ctrl.GetName()) {
					args = append(args, ctrl.GetHostname())
				}
			}
			for _, ctrl := range offline {
				if strings.Contains(s, ctrl.GetName()) {
					args = append(args, ctrl.GetHostname())
				}
			}
		}
	} else {
		hostnameArgs := []string{}
	ARG_LOOP:
		for _, arg := range args {
			if util.IsUUID(arg) {
				for _, ctrl := range controllers {
					if arg == ctrl.GetId() {
						hostnameArgs = append(hostnameArgs, ctrl.GetHostname())
						continue ARG_LOOP
					}
				}
				for _, ctrl := range offline {
					if arg == ctrl.GetId() {
						hostnameArgs = append(hostnameArgs, ctrl.GetHostname())
						continue ARG_LOOP
					}
				}
				log.WithField("id", arg).Info("No Controller found with provided id")
				continue
			}
			for _, ctrl := range controllers {
				if arg == ctrl.GetHostname() {
					hostnameArgs = append(hostnameArgs, ctrl.GetHostname())
					continue ARG_LOOP
				}
			}
			for _, ctrl := range offline {
				if arg == ctrl.GetHostname() {
					hostnameArgs = append(hostnameArgs, ctrl.GetHostname())
					continue ARG_LOOP
				}
			}
			log.WithField("arg", arg).Info("No Controller found with provided hostname")
		}
		// automatically add offline controllers to disable list, but only if the user has actively selected a valid controller first
		if len(hostnameArgs) > 0 {
			for _, ctrl := range offline {
				hostnameArgs = util.AppendIfMissing(hostnameArgs, ctrl.GetHostname())
			}
		}
		args = hostnameArgs
	}
	if len(args) <= 0 {
		return errors.New("No Controllers selected to disable")
	}
	if len(unselectedOffline) > 0 {
		return errors.New("Illegal operation: all OFFLINE Controllers must be disabled when disabling any other Controller.")
	}
	log.WithField("controllers", args).Debug("selected")

	disableList := []openapi.Appliance{}
	announceList := []openapi.Appliance{}
	for _, ctrl := range controllers {
		if util.InSlice(ctrl.GetHostname(), args) {
			disableList = append(disableList, ctrl)
		} else {
			announceList = append(announceList, ctrl)
		}
	}
	for _, ctrl := range offline {
		if util.InSlice(ctrl.GetHostname(), args) {
			disableList = append(disableList, ctrl)
		}
	}

	// Summary
	summary, err := printSummary(statData, disableList)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Out, summary)
	if !opts.NoInteractive {
		if err := prompt.AskConfirmation(); err != nil {
			return err
		}
	}

	var p *tui.Progress
	var t *tui.Tracker
	if !opts.CiMode {
		p = tui.New(ctx, opts.SpinnerOut())
		defer p.Wait()

		msg := "disabling controller"
		if len(disableList) > 1 {
			msg += "s"
		}
		t = p.AddTracker(msg, "waiting", "done")
		go func(t *tui.Tracker) {
			t.Watch([]string{"done"}, []string{})
		}(t)
		defer t.Update("done")
	}

	if t != nil {
		t.Update("disabling")
	}
	res, changeID, err := a.ForceDisableControllers(ctx, disableList)
	if err != nil {
		if t != nil {
			t.Fail(err.Error())
		}
		return err
	}

	// Re-add primary controller and sort the list
	announceList = append(announceList, *primaryController)
	sort.SliceStable(announceList, func(i, j int) bool {
		return announceList[i].GetName() > announceList[j].GetName()
	})

	var errs *multierror.Error
	if o, ok := res.GetOfflineControllersOk(); ok {
		for _, a := range announceList {
			if util.InSlice(a.GetId(), o) {
				errs = multierror.Append(errs, fmt.Errorf("Failed to send disable command to Controller %s", a.GetName()))
			}
		}
		if errs != nil {
			return errs.ErrorOrNil()
		}
	}

	var wg1 sync.WaitGroup
	for _, a := range announceList {
		wg1.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, a openapi.Appliance) {
			defer wg.Done()
			if _, err := changeAPI.RetryUntilCompleted(ctx, changeID, a.GetId()); err != nil {
				errs = multierror.Append(errs, err)
			}
		}(ctx, &wg1, a)
	}
	wg1.Wait()

	if t != nil {
		t.Update("re-partitioning IP allocations")
	}
	changeID, err = a.RepartitionIPAllocations(ctx)
	if err != nil {
		return err
	}
	var wg2 sync.WaitGroup
	for _, ctrl := range announceList {
		wg2.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ctrl openapi.Appliance) {
			defer wg.Done()
			if _, err := changeAPI.RetryUntilCompleted(ctx, changeID, ctrl.GetId()); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("IP re-partition failed for %s: %s", ctrl.GetName(), err.Error()))
			}
			if err := a.ApplianceStats.WaitForApplianceStatus(ctx, ctrl, appliancepkg.StatusNotBusy, nil); err != nil {
				errs = multierror.Append(errs, err)
			}
		}(ctx, &wg2, ctrl)
	}
	for _, ctrl := range disableList {
		wg2.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ctrl openapi.Appliance) {
			defer wg.Done()
			if err := a.ApplianceStats.WaitForApplianceStatus(ctx, ctrl, appliancepkg.StatusNotBusy, nil); err != nil {
				errs = multierror.Append(errs, err)
			}
		}(ctx, &wg2, ctrl)
	}
	wg2.Wait()

	return errs.ErrorOrNil()
}

const summaryTPLString string = `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

{{ .DisableTable }}
`

func printSummary(stats []openapi.StatsAppliancesListAllOfData, disable []openapi.Appliance) (string, error) {
	type stub struct {
		DisableTable string
	}
	disableBuffer := &bytes.Buffer{}
	dt := util.NewPrinter(disableBuffer, 4)
	dt.AddHeader("Name", "Hostname", "Status", "Version")

	data := stub{}
	for _, s := range stats {
		for _, a := range disable {
			if s.GetId() == a.GetId() {
				dt.AddLine(a.GetName(), a.GetHostname(), s.GetStatus(), s.GetVersion())
			}
		}
	}
	dt.Print()

	data.DisableTable = disableBuffer.String()

	tpl := template.Must(template.New("").Parse(summaryTPLString))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
