package appliance

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/template"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
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
	ErrSkipReasonInactive               = errors.New("appliance is inactive")
	ErrSkipReasonFiltered               = errors.New("filtered using the '--include' and/or '--exclude' flag")
	ErrSkipReasonAlreadyPrepared        = errors.New("appliance is already prepared for upgrade with a higher or equal version")
	ErrSkipReasonUnsupportedUpgradePath = errors.New("unsupported upgrade path")
	ErrSkipReasonAlreadySameVersion     = errors.New("appliance is already running a version higher or equal to the prepare version")
	ErrNoApplianceStats                 = errors.New("failed to find appliance stats")
	ErrVersionParse                     = errors.New("failed to parse current appliance version")
	ErrNothingToUpgrade                 = errors.New("No appliances are ready to upgrade. Please run 'upgrade prepare' before trying to complete an upgrade")
)

type UpgradePlan struct {
	PrimaryController       *openapi.Appliance
	Controllers             []openapi.Appliance
	LogForwardersAndServers []openapi.Appliance
	Batches                 [][]openapi.Appliance
	Skipping                []SkipUpgrade
	BackupIds               []string
	stats                   *openapi.ApplianceWithStatusList
	upgradeStatusMap        map[string]UpgradeStatusResult
	adminHostname           string
	primary                 *openapi.Appliance
	allAppliances           []openapi.Appliance
}

func NewUpgradePlan(
	appliances []openapi.Appliance, stats *openapi.ApplianceWithStatusList,
	upgradeStatusMap map[string]UpgradeStatusResult,
	adminHostname string,
	filter map[string]map[string]string,
	orderBy []string,
	descending bool,
	maxUnavailable int,
) (*UpgradePlan, error) {
	plan := UpgradePlan{
		adminHostname:    adminHostname,
		stats:            stats,
		upgradeStatusMap: upgradeStatusMap,
		allAppliances:    appliances,
	}

	primary, err := FindPrimaryController(appliances, plan.adminHostname, false)
	if err != nil {
		return nil, err
	}
	plan.primary = primary

	finalApplianceList, filtered, err := FilterAppliances(appliances, filter, orderBy, descending)
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
	slices.SortStableFunc(finalApplianceList, func(i, j openapi.Appliance) int { return cmp.Compare(i.GetName(), j.GetName()) })

	gatewaysGroupedBySite := map[string][]openapi.Appliance{}
	logforwardersGroupedBySite := map[string][]openapi.Appliance{}
	other := []openapi.Appliance{}

	// LogForwarders and LogServers need to in their own group
	// when upgrading from <= 5.5 to >= 6.0
	lflsConstraint, _ := version.NewConstraint(">= 6.0.0-beta")

	var errs *multierror.Error
	for _, a := range finalApplianceList {
		// Get current version and stats
		stats, err := ApplianceStats(&a, plan.stats)
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.addSkip(a, ErrNoApplianceStats)
			continue
		}
		currentVersion, err := ParseVersionString(stats.GetApplianceVersion())
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.addSkip(a, ErrVersionParse)
			continue
		}

		// Get upgrade status and target version
		upgradeStatus, ok := upgradeStatusMap[a.GetId()]
		if !ok {
			plan.addSkip(a, ErrNoApplianceStats)
			continue
		}
		if upgradeStatus.Status != UpgradeStatusReady {
			plan.addSkip(a, ErrSkipReasonNotPrepared)
			continue
		}
		targetVersion, err := ParseVersionString(upgradeStatus.Details)
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.addSkip(a, ErrVersionParse)
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
				site := a.GetSiteName()
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
					site := a.GetSiteName()
					logforwardersGroupedBySite[site] = append(logforwardersGroupedBySite[site], a)
					continue
				}
			}
		}
		other = append(other, a)
	}

	// Sort the output in the upgrade plan
	if len(plan.LogForwardersAndServers) > 0 {
		slices.SortStableFunc(plan.LogForwardersAndServers, func(i, j openapi.Appliance) int { return cmp.Compare(i.GetName(), j.GetName()) })
	}

	if len(gatewaysGroupedBySite) > 0 || len(logforwardersGroupedBySite) > 0 || len(other) > 0 {
		// Determine how many batches we will do
		// using the amount of gateways and logforwarders per site
		batches := calculateBatches(gatewaysGroupedBySite, logforwardersGroupedBySite, other, maxUnavailable)

		// Equally distribute gateways to batches
		// Each batch should contain only one gateway per site
		plan.Batches = createBatches(batches, maxUnavailable, gatewaysGroupedBySite, logforwardersGroupedBySite, other)
	}

	return &plan, nil
}

func createBatches(batchCount, maxUnavailable int, gateways, logforwarders map[string][]openapi.Appliance, other []openapi.Appliance) [][]openapi.Appliance {
	result := make([][]openapi.Appliance, batchCount)

	gatewayGroups := divideByMaxUnavailable(gateways, maxUnavailable)
	logForwarderGroups := divideByMaxUnavailable(logforwarders, maxUnavailable)

	// distribute gateways and logforwarders to groups
	resultIndex := 0
	result[resultIndex] = []openapi.Appliance{}
	for i := 0; i < len(gatewayGroups); i++ {
		result[resultIndex] = append(result[resultIndex], gatewayGroups[i]...)
		resultIndex++
	}
	resultIndex = 0
	for i := 0; i < len(logForwarderGroups); i++ {
		result[resultIndex] = append(result[resultIndex], logForwarderGroups[i]...)
		resultIndex++
	}

	// distribute other appliances and even out the groups
	slices.SortStableFunc(other, func(i, y openapi.Appliance) int { return cmp.Compare(i.GetName(), y.GetName()) })
	for _, o := range other {
		resultIndex = util.SmallestGroupIndex(result)
		result[resultIndex] = append(result[resultIndex], o)
	}

	// sort the resulting groups
	for i := 0; i < len(result); i++ {
		slices.SortStableFunc(result[i], func(x, y openapi.Appliance) int { return cmp.Compare(x.GetName(), y.GetName()) })
	}
	slices.SortStableFunc(result, func(i, j []openapi.Appliance) int { return cmp.Compare(i[0].GetSiteName(), j[0].GetSiteName()) })

	return result
}

func divideByMaxUnavailable(appliances map[string][]openapi.Appliance, maxUnavailable int) [][]openapi.Appliance {
	totalCount := 0
	for _, group := range appliances {
		totalCount += len(group)
	}
	if totalCount <= 0 {
		return nil
	}

	// for safety, if maxUnavailable is 0, set it to 1
	if maxUnavailable <= 0 {
		maxUnavailable = 1
	}

	keys := []string{}
	for k := range appliances {
		keys = append(keys, k)
	}
	slices.SortStableFunc(keys, func(i, j string) int { return cmp.Compare(i, j) })

	for _, group := range appliances {
		slices.SortStableFunc(group, func(i, j openapi.Appliance) int { return cmp.Compare(i.GetName(), j.GetName()) })
	}

	biggestGroupKey := biggestGroupKey(appliances)
	biggestGroupLength := len(appliances[biggestGroupKey])
	groupCount := biggestGroupLength / maxUnavailable
	if biggestGroupLength%maxUnavailable > 0 {
		groupCount++
	}
	groups := make([][]openapi.Appliance, groupCount)
	for i := 0; i < groupCount; i++ {
		groups[i] = make([]openapi.Appliance, 0, maxUnavailable)
	}

	groupIndex := 0
	for _, k := range keys {
		remaining := appliances[k]
		for len(remaining) > 0 {
			var picked []openapi.Appliance
			picked, remaining = util.SliceTake(remaining, maxUnavailable)
			groups[groupIndex] = append(groups[groupIndex], picked...)
			if groupIndex == len(groups)-1 {
				groupIndex = 0
				continue
			}
			groupIndex++
		}
	}

	// sort output
	for _, g := range groups {
		slices.SortStableFunc(g, func(i, j openapi.Appliance) int { return cmp.Compare(i.GetName(), j.GetName()) })
	}

	return groups
}

func biggestGroupKey(groups map[string][]openapi.Appliance) string {
	var res string
	var biggest int
	for i, g := range groups {
		count := len(g)
		if biggest == 0 {
			biggest = count
			res = i
		}
		if count > biggest {
			biggest = count
			res = i
		}
	}
	return res
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

func (up *UpgradePlan) addSkip(appliance openapi.Appliance, reason error) {
	up.Skipping = append(up.Skipping, SkipUpgrade{
		Appliance: appliance,
		Reason:    reason,
	})
	up.allAppliances = AppendUniqueAppliance(up.allAppliances, appliance)
}

func (up *UpgradePlan) AddOfflineAppliances(appliances []openapi.Appliance) {
	for _, a := range appliances {
		up.addSkip(a, ErrSkipReasonOffline)
	}
}

func (up *UpgradePlan) AddInactiveAppliances(appliances []openapi.Appliance) {
	for _, a := range appliances {
		up.addSkip(a, ErrSkipReasonInactive)
	}
}

func (up *UpgradePlan) GetPrimaryController() *openapi.Appliance {
	return up.primary
}

func (up *UpgradePlan) Validate() error {
	// we check if all controllers need upgrade very early
	if _, err := CheckNeedsMultiControllerUpgrade(up.stats, up.upgradeStatusMap, up.allAppliances); err != nil {
		return err
	}
	return nil
}

func (up *UpgradePlan) HasDiffVersions(newStats []openapi.ApplianceWithStatus) (bool, map[string]string) {
	res := map[string]string{}
	versionList := []string{}
	for _, stat := range newStats {
		statVersionString := stat.GetApplianceVersion()
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
		currentVersion, targetVersion := applianceVersions(*up.PrimaryController, *up.stats)
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
			current, target := applianceVersions(ctrl, *up.stats)
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
			current, target := applianceVersions(lfls, *up.stats)
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
				current, target := applianceVersions(a, *up.stats)
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
		slices.SortStableFunc(stub.Skipped, func(i, j SkipUpgrade) int { return cmp.Compare(i.Appliance.GetName(), j.Appliance.GetName()) })
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

func (up *UpgradePlan) PrintPostCompleteSummary(out io.Writer, stats []openapi.ApplianceWithStatus) error {
	hasDiff, applianceVersions := up.HasDiffVersions(stats)
	keys := make([]string, 0, len(applianceVersions))
	for k := range applianceVersions {
		keys = append(keys, k)
	}
	slices.Sort(keys)

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

func applianceVersions(a openapi.Appliance, s openapi.ApplianceWithStatusList) (currentVersion *version.Version, targetVersion *version.Version) {
	stats, _ := ApplianceStats(&a, &s)
	currentVersion, _ = ParseVersionString(stats.GetApplianceVersion())
	us := stats.GetDetails().Upgrade
	targetVersion, _ = ParseVersionString(us.GetDetails())
	return
}

func ApplianceStats(a *openapi.Appliance, stats *openapi.ApplianceWithStatusList) (*openapi.ApplianceWithStatus, error) {
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
{{ range .Skipped }}  - {{ .Appliance.Name }}: {{ .Reason }}
{{ end -}}
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

func calculateBatches(gatewaysBySite, logForwardersBySite map[string][]openapi.Appliance, other []openapi.Appliance, maxUnavailable int) int {
	batches := 0
	for _, g := range gatewaysBySite {
		if len(g) > batches {
			batches = len(g)
		}
	}
	for _, lf := range logForwardersBySite {
		if len(lf) > batches {
			batches = len(lf)
		}
	}

	// batchSize needs to min 1 if there are any other appliances prepared
	if len(other) > 0 && batches == 0 {
		batches = 1
	}

	// apply maxUnavailable option before returning
	if maxUnavailable <= 0 {
		maxUnavailable = 1
	}
	if maxUnavailable > 1 {
		if batches == maxUnavailable || batches < maxUnavailable {
			return 1
		}
		for batches%maxUnavailable != 0 {
			batches--
		}
	}

	return batches
}
