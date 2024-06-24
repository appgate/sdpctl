package appliance

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cheynewallace/tabby"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

type SkipUpgrade struct {
	Reason    error
	Appliance openapi.Appliance
}

func (su SkipUpgrade) Error() string {
	return fmt.Sprintf("%s: %s", su.Appliance.GetName(), su.Reason)
}

var (
	ErrSkipReasonNotPrepared            = errors.New("appliance is not prepared for upgrade")
	ErrSkipReasonOffline                = errors.New("appliance is offline")
	ErrSkipReasonFiltered               = errors.New("filtered using the '--include' and/or '--exclude' flag")
	ErrSkipReasonAlreadyPrepared        = errors.New("appliance is already prepared for upgrade with a higher or equal version")
	ErrSkipReasonUnsupportedUpgradePath = errors.New("Upgrading from version 6.0.0 to version 6.2.x is unsupported. Version 6.0.1 or later is required.")
	ErrSkipReasonAlreadySameVersion     = errors.New("appliance is already running a version higher or equal to the prepare version")
	ErrNoApplianceStats                 = errors.New("failed to find appliance stats")
	ErrVersionParse                     = errors.New("failed to parse current appliance version")
)

type UpgradePlan struct {
	PrimaryController       *openapi.Appliance
	Controllers             []openapi.Appliance
	LogForwardersAndServers []openapi.Appliance
	Batches                 [][]openapi.Appliance
	Skipping                []SkipUpgrade
	BackupIds               []string
	stats                   openapi.StatsAppliancesList
	adminHostname           string
}

func NewUpgradePlan(appliances []openapi.Appliance, stats openapi.StatsAppliancesList, adminHostname string, filter map[string]map[string]string, orderBy []string, descending bool) (*UpgradePlan, error) {
	plan := UpgradePlan{
		adminHostname: adminHostname,
		stats:         stats,
	}

	primary, err := FindPrimaryController(appliances, plan.adminHostname, false)
	if err != nil {
		return nil, err
	}

	// we check if all controllers need upgrade very early
	if _, err := CheckNeedsMultiControllerUpgrade(stats, appliances); err != nil {
		return nil, err
	}

	postOnlineInclude, offline, err := FilterAvailable(appliances, stats.GetData())
	if err != nil {
		return nil, err
	}
	for _, o := range offline {
		plan.Skipping = append(plan.Skipping, SkipUpgrade{
			Appliance: o,
			Reason:    ErrSkipReasonOffline,
		})
	}

	finalApplianceList, filtered, err := FilterAppliances(postOnlineInclude, filter, orderBy, descending)
	if err != nil {
		return nil, err
	}
	for _, f := range filtered {
		plan.Skipping = append(plan.Skipping, SkipUpgrade{
			Appliance: f,
			Reason:    ErrSkipReasonFiltered,
		})
	}

	// Sort input group first
	sort.SliceStable(finalApplianceList, func(i, j int) bool {
		return finalApplianceList[i].GetName() < finalApplianceList[j].GetName()
	})

	gatewaysGroupedBySite := map[string][]openapi.Appliance{}
	logforwardersGroupedBySite := map[string][]openapi.Appliance{}
	other := []openapi.Appliance{}

	// LogForwarders and LogServers need to in their own group
	// when upgrading from <= 5.5 to >= 6.0
	lflsConstraint, _ := version.NewConstraint(">= 6.0.0-beta")

	var errs *multierror.Error
	for _, a := range finalApplianceList {
		// Get current version and stats
		stats, err := ApplianceStats(a, plan.stats)
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.Skipping = append(plan.Skipping, SkipUpgrade{
				Appliance: a,
				Reason:    ErrNoApplianceStats,
			})
			continue
		}
		currentVersion, err := ParseVersionString(stats.GetVersion())
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.Skipping = append(plan.Skipping, SkipUpgrade{
				Appliance: a,
				Reason:    ErrVersionParse,
			})
			continue
		}

		// Get upgrade status and target version
		upgradeStatus := stats.GetUpgrade()
		if status, ok := upgradeStatus.GetStatusOk(); ok && *status != UpgradeStatusReady {
			plan.Skipping = append(plan.Skipping, SkipUpgrade{
				Appliance: a,
				Reason:    ErrSkipReasonNotPrepared,
			})
			continue
		}
		targetVersion, err := ParseVersionString(upgradeStatus.GetDetails())
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.Skipping = append(plan.Skipping, SkipUpgrade{
				Appliance: a,
				Reason:    ErrVersionParse,
			})
			continue
		}

		if ctrl, ok := a.GetControllerOk(); ok {
			if ctrl.GetEnabled() {
				if a.GetId() == primary.GetId() {
					plan.PrimaryController = &a
				} else {
					plan.Controllers = append(plan.Controllers, a)
				}
				continue
			}
		}
		if gw, ok := a.GetGatewayOk(); ok {
			if gw.GetEnabled() {
				site := a.GetSite()
				gatewaysGroupedBySite[site] = append(gatewaysGroupedBySite[site], a)
				continue
			}
		}
		if !lflsConstraint.Check(currentVersion) && lflsConstraint.Check(targetVersion) {
			if ls, ok := a.GetLogServerOk(); ok {
				if ls.GetEnabled() {
					plan.LogForwardersAndServers = append(plan.LogForwardersAndServers, a)
					continue
				}
			}
			if lf, ok := a.GetLogForwarderOk(); ok {
				if lf.GetEnabled() {
					plan.LogForwardersAndServers = append(plan.LogForwardersAndServers, a)
					continue
				}
			}
		} else {
			if lf, ok := a.GetLogForwarderOk(); ok {
				if lf.GetEnabled() {
					logforwardersGroupedBySite[a.GetSite()] = append(logforwardersGroupedBySite[a.GetSite()], a)
					continue
				}
			}
		}
		other = append(other, a)
	}

	// Determine how many batches we will do
	// using the amount of gateways and logforwarders per site
	batchSize := 0
	for _, g := range gatewaysGroupedBySite {
		if len(g) > batchSize {
			batchSize = len(g)
		}
	}
	for _, lf := range logforwardersGroupedBySite {
		if len(lf) > batchSize {
			batchSize = len(lf)
		}
	}

	// Equally distribute gateways to batches
	// Each batch should contain only one gateway per site
	plan.Batches = make([][]openapi.Appliance, batchSize)
	for _, appliances := range gatewaysGroupedBySite {
		batchIndex := 0
		for _, a := range appliances {
			batch := plan.Batches[batchIndex]
			batch = append(batch, a)
			plan.Batches[batchIndex] = batch
			if batchIndex == batchSize-1 {
				batchIndex = 0
				continue
			}
			batchIndex++
		}
	}

	// Equally distribute logforwarders to batches
	// Each batch should contain only one logforwarder per site
	for _, appliances := range logforwardersGroupedBySite {
		batchIndex := 0
		for _, a := range appliances {
			batch := plan.Batches[batchIndex]
			batch = append(batch, a)
			plan.Batches[batchIndex] = batch
			if batchIndex == batchSize-1 {
				batchIndex = 0
				continue
			}
			batchIndex++
		}
	}

	// Distribute the rest of the appliances into the batches
	// Strategy is to balance the batches so they are of roughly equal size
	batchIndex := 0
	for _, a := range other {
		// Get the index of the batch with the least amount of appliances in
		var minBatch *int
		for i, b := range plan.Batches {
			if minBatch == nil {
				minBatch = openapi.PtrInt(len(b))
				batchIndex = i
				continue
			}
			if len(b) < *minBatch {
				minBatch = openapi.PtrInt(len(b))
				batchIndex = i
			}
		}

		// Append appliance to the batch with index found above
		plan.Batches[batchIndex] = append(plan.Batches[batchIndex], a)
	}

	// Sort the output in the upgrade plan
	if len(plan.LogForwardersAndServers) > 0 {
		sort.SliceStable(plan.LogForwardersAndServers, func(i, j int) bool {
			return plan.LogForwardersAndServers[i].GetName() < plan.LogForwardersAndServers[j].GetName()
		})
	}
	for i := 0; i < len(plan.Batches); i++ {
		sort.SliceStable(plan.Batches[i], func(ix, jx int) bool {
			return plan.Batches[i][ix].GetName() < plan.Batches[i][jx].GetName()
		})
	}

	return &plan, nil
}

func (up *UpgradePlan) AddBackups(applianceIds []string) error {
	var errs *multierror.Error
	for _, id := range applianceIds {
		if !util.IsUUID(id) {
			errs = multierror.Append(errs, fmt.Errorf("%s is not a valid UUID", id))
		}
	}
	up.BackupIds = applianceIds
	return errs.ErrorOrNil()
}

func (up *UpgradePlan) HasDiffVersions(newStats []openapi.StatsAppliancesListAllOfData) (bool, map[string]string) {
	res := map[string]string{}
	versionList := []string{}
	for _, stat := range newStats {
		statVersionString := stat.GetVersion()
		if statVersionString != unknownStat {
			v, err := ParseVersionString(statVersionString)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"appliance": stat.GetName(),
					"version":   statVersionString,
				}).Warn("failed to parse version string")
			}
			versionString := statVersionString
			if v != nil {
				versionString = v.String()
			}
			res[stat.GetName()] = versionString
			versionList = append(versionList, versionString)
		}

	}
	unique := uniqueString(versionList)
	return len(unique) != 1, res
}

func (up *UpgradePlan) NothingToUpgrade() bool {
	return up.PrimaryController == nil && len(up.Controllers) <= 0 && len(up.LogForwardersAndServers) <= 0 && len(up.Batches) <= 0
}

func (up *UpgradePlan) PrintPreCompleteSummary(out io.Writer) error {
	type upgradeStep struct {
		Description, Table string
	}
	type stubStruct struct {
		Steps   []upgradeStep
		Skipped []SkipUpgrade
	}

	shouldBackup := func(id string) string {
		res := tui.No
		if util.InSlice(id, up.BackupIds) {
			res = tui.Yes
		}
		return res
	}
	stub := stubStruct{
		Steps: []upgradeStep{},
	}
	tableHeaders := func(t *tabby.Tabby) {
		t.AddHeader("Appliance", "Site", "Current version", "Prepared version", "Backup")
	}
	if up.PrimaryController != nil {
		currentVersion, targetVersion := applianceVersions(*up.PrimaryController, up.stats)
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		tableHeaders(t)
		t.AddLine(up.PrimaryController.GetName(), up.PrimaryController.GetSiteName(), currentVersion, targetVersion, shouldBackup(up.PrimaryController.GetId()))
		t.Print()
		stub.Steps = append(stub.Steps, upgradeStep{
			Description: strings.Join(primaryControllerDescription, descriptionIndent),
			Table:       util.PrefixStringLines(tb.String(), " ", 4),
		})
	}
	if len(up.Controllers) > 0 {
		step := upgradeStep{
			Description: strings.Join(additionalControllerDescription, descriptionIndent),
		}
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		tableHeaders(t)
		for _, ctrl := range up.Controllers {
			current, target := applianceVersions(ctrl, up.stats)
			t.AddLine(ctrl.GetName(), ctrl.GetSiteName(), current, target, shouldBackup(ctrl.GetId()))
		}
		t.Print()
		step.Table = util.PrefixStringLines(tb.String(), " ", 4)
		stub.Steps = append(stub.Steps, step)
	}
	if len(up.LogForwardersAndServers) > 0 {
		step := upgradeStep{
			Description: strings.Join(logForwaredersAndServersDescription, descriptionIndent),
		}
		tb := &bytes.Buffer{}
		t := util.NewPrinter(tb, 4)
		tableHeaders(t)
		for _, lfls := range up.LogForwardersAndServers {
			current, target := applianceVersions(lfls, up.stats)
			t.AddLine(lfls.GetName(), lfls.GetSiteName(), current, target, shouldBackup(lfls.GetId()))
		}
		t.Print()
		step.Table = util.PrefixStringLines(tb.String(), " ", 4)
		stub.Steps = append(stub.Steps, step)
	}
	if len(up.Batches) > 0 {
		tb := &bytes.Buffer{}
		for i, c := range up.Batches {
			fmt.Fprintf(tb, "Batch #%d:\n\n", i+1)
			t := util.NewPrinter(tb, 4)
			tableHeaders(t)
			for _, a := range c {
				current, target := applianceVersions(a, up.stats)
				// s := fmt.Sprintf("- %s: %s -> %s", a.GetName(), current, target)
				t.AddLine(a.GetName(), a.GetSiteName(), current, target, shouldBackup(a.GetId()))
			}
			t.AddLine("")
			t.Print()
		}
		stub.Steps = append(stub.Steps, upgradeStep{
			Description: strings.Join(additionalAppliancesDescription, descriptionIndent),
			Table:       util.PrefixStringLines(tb.String(), " ", 4),
		})
	}

	if len(up.Skipping) > 0 {
		stub.Skipped = up.Skipping
		sort.Slice(stub.Skipped, func(i, j int) bool {
			return stub.Skipped[i].Appliance.GetName() < stub.Skipped[j].Appliance.GetName()
		})
	}

	t := template.Must(template.New("").Funcs(util.TPLFuncMap).Parse(upgradeSummaryTpl))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, stub); err != nil {
		return err
	}

	_, err := fmt.Fprint(out, tpl.String())

	return err
}

var postCompleteTPL string = `{{ now }}

UPGRADE COMPLETE

{{ .VersionTable }}
{{ if .HasDiff }}WARNING: Upgrade was completed, but not all appliances are running the same version.{{ end }}
`

func (up *UpgradePlan) PrintPostCompleteSummary(out io.Writer, stats []openapi.StatsAppliancesListAllOfData) error {
	hasDiff, applianceVersions := up.HasDiffVersions(stats)
	keys := make([]string, 0, len(applianceVersions))
	for k := range applianceVersions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	type tplStub struct {
		VersionTable string
		HasDiff      bool
	}

	tb := &bytes.Buffer{}
	tp := util.NewPrinter(tb, 4)
	tp.AddHeader("Appliance", "Current Version")
	for _, k := range keys {
		tp.AddLine(k, applianceVersions[k])
	}
	tp.Print()

	tplData := tplStub{
		VersionTable: tb.String(),
		HasDiff:      hasDiff,
	}
	t := template.Must(template.New("").Funcs(util.TPLFuncMap).Parse(postCompleteTPL))
	return t.Execute(out, tplData)
}

func applianceVersions(a openapi.Appliance, s openapi.StatsAppliancesList) (currentVersion *version.Version, targetVersion *version.Version) {
	stats, _ := ApplianceStats(a, s)
	currentVersion, _ = ParseVersionString(stats.GetVersion())
	us := stats.GetUpgrade()
	targetVersion, _ = ParseVersionString(us.GetDetails())
	return
}

func ApplianceStats(a openapi.Appliance, stats openapi.StatsAppliancesList) (*openapi.StatsAppliancesListAllOfData, error) {
	for _, s := range stats.GetData() {
		if s.GetId() == a.GetId() {
			return &s, nil
		}
	}
	return nil, ErrNoApplianceStats
}

var upgradeSummaryTpl string = `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in steps:
{{ range $i, $s := .Steps }}
 {{ sum $i 1 }}. {{ $s.Description }}

{{ $s.Table }}
{{ end }}


{{- if .Skipped -}}
Appliances that will be skipped:
  {{ range .Skipped }}- {{ .Appliance.Name }}: {{ .Reason -}}{{- end }}
{{ end }}`

var (
	descriptionIndent            = "\n    "
	primaryControllerDescription = []string{
		"The primary Controller will be upgraded",
		"This will result in the API being unreachable while completing the primary Controller upgrade",
	}
	additionalControllerDescription = []string{
		"Additional Controllers will be upgraded in series.",
		"Additional Controllers will be put into maintenance mode before being upgraded. Maintenance",
		"mode will then be disabled once the upgrade has completed on the controller.",
		"This step will also reboot the upgraded Controllers for the upgrade to take effect.",
	}
	logForwaredersAndServersDescription = []string{
		"Appliances with LogForwarder/LogServer functions are upgraded",
		"Other appliances need a connection to to these appliances for logging",
	}
	additionalAppliancesDescription = []string{
		"Additional appliances will be upgraded in parallel batches. The additional appliances will be split into",
		"batches to keep the Collective as available as possible during the upgrade process",
		"Some of the additional appliances may need to be rebooted for the upgrade to take effect",
	}
)
