package appliance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

func PrintDiskSpaceWarningMessage(out io.Writer, stats []openapi.ApplianceWithStatus, apiVersion int) {
	p := util.NewPrinter(out, 4)
	diskHeader := "Disk Usage"
	if apiVersion >= 18 {
		diskHeader += " (used / total)"
	}
	p.AddHeader("Name", diskHeader)
	for _, a := range stats {
		diskUsage := fmt.Sprintf("%v%%", a.GetDisk())
		if v, ok := a.GetDetailsOk(); ok {
			diskInfo := *v.Disk
			used, total := diskInfo.GetUsed(), diskInfo.GetTotal()
			percentUsed := (float32(used) / float32(total)) * 100
			diskUsage = fmt.Sprintf("%.2f%% (%s / %s)", percentUsed, PrettyBytes(float64(used)), PrettyBytes(float64(total)))
		}
		p.AddLine(a.GetName(), diskUsage)
	}

	fmt.Fprint(out, "\nWARNING: Some appliances have very little space available\n\n")
	p.Print()
	fmt.Fprintln(out, `
Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.`)
}

func HasLowDiskSpace(stats []openapi.ApplianceWithStatus) []openapi.ApplianceWithStatus {
	result := []openapi.ApplianceWithStatus{}
	for _, s := range stats {
		if s.GetDisk() >= 75 {
			result = append(result, s)
		}
	}
	return result
}

func applianceGroupDescription(appliances []openapi.Appliance) string {
	functions := ActiveFunctions(appliances)
	var funcs []string
	for k, value := range functions {
		if _, ok := functions[k]; ok && value {
			funcs = append(funcs, k)
		}
	}
	sort.Strings(funcs)
	return strings.Join(funcs, ", ")
}

func unique(slice []openapi.Appliance) []openapi.Appliance {
	keys := make(map[string]bool)
	list := []openapi.Appliance{}
	for _, entry := range slice {
		if _, value := keys[entry.GetId()]; !value {
			keys[entry.GetId()] = true
			list = append(list, entry)
		}
	}
	return list
}

func uniqueString(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

const autoScalingWarning = `
{{ if .Template }}
There is an auto-scale template configured: {{ .Template.Name }}
{{end}}

Found {{ .Count }} auto-scaled gateway running version < 16:
{{range .Appliances}}
  - {{.Name -}}
{{end}}

Make sure that the health check for those auto-scaled gateways is disabled.
Not disabling the health checks in those auto-scaled gateways could cause them to be deleted, breaking all the connections established with them.
`

func ShowAutoscalingWarningMessage(templateAppliance *openapi.Appliance, gateways []openapi.Appliance) (string, error) {
	type stub struct {
		Template   *openapi.Appliance
		Appliances []openapi.Appliance
		Count      int
	}

	data := stub{
		Template:   templateAppliance,
		Appliances: unique(gateways),
		Count:      len(gateways),
	}
	t := template.Must(template.New("").Parse(autoScalingWarning))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

// CheckVersions will check if appliance versions are equal to the version being uploaded on all appliances
// Returns a slice of appliances that are not equal, a slice of appliances that have the same version and an error
func CheckVersions(ctx context.Context, stats openapi.ApplianceWithStatusList, appliances []openapi.Appliance, v *version.Version) ([]openapi.Appliance, []SkipUpgrade) {
	skip := []SkipUpgrade{}
	keep := []openapi.Appliance{}

	for _, appliance := range appliances {
		for _, stat := range stats.GetData() {
			if stat.GetId() == appliance.GetId() {
				statV, err := ParseVersionString(stat.GetApplianceVersion())
				if err != nil {
					log.Warn("failed to parse version from stats")
					skip = append(skip, SkipUpgrade{
						Appliance: appliance,
						Reason:    ErrVersionParse,
					})
					continue
				}
				res, err := CompareVersionsAndBuildNumber(statV, v)
				if err != nil {
					log.Warn("failed to compare versions")
					skip = append(skip, SkipUpgrade{
						Appliance: appliance,
						Reason:    errors.New("failed to compare versions"),
					})
					continue
				}
				if res < 1 {
					us := stat.GetDetails().Upgrade
					reason := ErrSkipReasonAlreadyPrepared
					if us.GetStatus() != UpgradeStatusReady {
						reason = ErrSkipReasonAlreadySameVersion
					}
					skip = append(skip, SkipUpgrade{
						Appliance: appliance,
						Reason:    reason,
					})
					continue
				}

				// Check specific version constraints on upgrades we know will break
				if err := CheckApplianceVersionsDisallowed(statV, v); err != nil {
					skip = append(skip, SkipUpgrade{
						Appliance: appliance,
						Reason:    fmt.Errorf("%s: %w", ErrSkipReasonUnsupportedUpgradePath, err),
					})
					continue
				}

				keep = append(keep, appliance)
			}
		}
	}

	return keep, skip
}

const (
	IsLower   = -1
	IsEqual   = 0
	IsGreater = 1
)

// CompareVersionsAndBuildNumber compares two versions and returns the result with an integer representation
// -1 if y is lower than x
// 0 if versions match
// 1 if y is greater than x
func CompareVersionsAndBuildNumber(x, y *version.Version) (int, error) {
	if x == nil || y == nil {
		return 0, fmt.Errorf("Failed to compare versions, got nil version - x=%v, y=%v", x, y)
	}
	var err error
	if len(x.Prerelease()) > 0 {
		x, err = ParseVersionString(x.String())
		if err != nil {
			return 0, err
		}
	}
	if len(y.Prerelease()) > 0 {
		y, err = ParseVersionString(y.String())
		if err != nil {
			return 0, err
		}
	}
	res := y.Compare(x)

	// if res is 0, we also compare build number
	// both x and y needs to have a parsable build number for this check to run
	if res == IsEqual && len(y.Metadata()) > 0 && len(x.Metadata()) > 0 {
		buildX, err := version.NewVersion(x.Metadata())
		if err != nil {
			return res, err
		}
		buildY, err := version.NewVersion(y.Metadata())
		if err != nil {
			return res, err
		}
		res = buildY.Compare(buildX)
	}

	return res, nil
}

// unknownStat is the response given by the appliance stats api if the appliance is offline.
const unknownStat = "unknown"

func HasDiffVersions(stats []openapi.StatsAppliancesListAllOfData) (bool, map[string]string) {
	res := map[string]string{}
	versionList := []string{}
	for _, stat := range stats {
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

const (
	MajorVersion = uint8(4)
	MinorVersion = uint8(2)
	PatchVersion = uint8(1)
)

func getUpgradeVersionType(x, y *version.Version) uint8 {
	var patch, minor, major uint8

	if x == nil || y == nil {
		return 0
	}

	xSeg := x.Segments()
	ySeg := y.Segments()

	// Patch
	if xSeg[2] < ySeg[2] {
		patch = PatchVersion
	}
	// Minor
	if xSeg[1] < ySeg[1] {
		minor = MinorVersion
	}
	// Major
	if xSeg[0] < ySeg[0] {
		major = MajorVersion
	}

	return major | minor | patch
}

func IsMajorUpgrade(current, next *version.Version) bool {
	return getUpgradeVersionType(current, next)&MajorVersion == MajorVersion
}

func IsMinorUpgrade(current, next *version.Version) bool {
	return getUpgradeVersionType(current, next)&MinorVersion == MinorVersion
}

func IsPatchUpgrade(current, next *version.Version) bool {
	return getUpgradeVersionType(current, next)&PatchVersion == PatchVersion
}

func controllerCount(appliances []openapi.Appliance) int {
	i := 0
	for _, a := range appliances {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			i++
		}
	}
	return i
}

var (
	ErrNeedsAllControllerUpgrade = errors.New("all controllers need to be prepared when doing a major or minor version upgrade.")
	ErrControllerVersionMismatch = errors.New("all controllers need to be prepared with the same version when doing a major or minor version upgrade.")
)

func countControllers(stats *openapi.ApplianceWithStatusList) int {
	var controllers int = 0
	for _, stat := range stats.GetData() {
		if controller, ok := stat.GetControllerOk(); ok {
			if controller.GetEnabled() {
				controllers++
			}
		}
	}
	return controllers
}

func CheckNeedsMultiControllerUpgrade(stats *openapi.ApplianceWithStatusList, upgradeStatusMap map[string]UpgradeStatusResult, appliances []openapi.Appliance) ([]openapi.Appliance, error) {
	var (
		errs                   *multierror.Error
		preparedControllers    []openapi.Appliance
		unpreparedControllers  []openapi.Appliance
		alreadySameVersion     []openapi.Appliance
		mismatchControllers    []openapi.Appliance
		isMajorOrMinor         bool
		offlineControllers     []openapi.Appliance
		totalControllers       = countControllers(stats)
		highestPreparedVersion = version.Must(version.NewVersion("0.0.0"))
		highestCurrentVersion  = version.Must(version.NewVersion("0.0.0"))
	)

	for _, as := range stats.GetData() {
		for _, app := range appliances {
			if as.GetId() != app.GetId() {
				continue
			}
			if online := StatsIsOnline(as); !online {
				offlineControllers = append(offlineControllers, app)
				continue
			}
			if ctrl, ok := app.GetControllerOk(); ok && ctrl.GetEnabled() {
				current, err := ParseVersionString(as.GetApplianceVersion())
				if err != nil {
					errs = multierror.Append(errs, err)
					continue
				}
				if res, _ := CompareVersionsAndBuildNumber(highestCurrentVersion, current); res > 0 {
					highestCurrentVersion = current
				}
				us := upgradeStatusMap[app.GetId()]
				if us.Status == UpgradeStatusReady {
					preparedControllers = append(preparedControllers, app)
					targetVersion, err := ParseVersionString(us.Details)
					if err != nil {
						errs = multierror.Append(errs, err)
						continue
					}
					res, _ := CompareVersionsAndBuildNumber(highestPreparedVersion, targetVersion)
					if res < 0 {
						unpreparedControllers = append(unpreparedControllers, app)
					}
					if res > 0 {
						highestPreparedVersion = targetVersion
					}
					isMajorOrMinor = IsMajorUpgrade(current, targetVersion) || IsMinorUpgrade(current, targetVersion)
				} else {
					unpreparedControllers = append(unpreparedControllers, app)
				}
			}
		}
	}
	// check if prepared controllers has mismatching version
	preparedClone := slices.Clone(preparedControllers)
	for i, a := range preparedClone {
		us, ok := upgradeStatusMap[a.GetId()]
		if !ok {
			errs = multierror.Append(errs, ErrNoApplianceStats)
			continue
		}
		targetVersion, err := ParseVersionString(us.Details)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		if res, _ := CompareVersionsAndBuildNumber(highestPreparedVersion, targetVersion); res < 0 {
			mismatchControllers = append(mismatchControllers, a)
			preparedControllers = append(preparedControllers[:i], preparedControllers[i+1:]...)
		}
	}

	// Now we need to check if the unprepared controllers
	// are already running the max prepared version
	unpreparedClone := slices.Clone(unpreparedControllers)
	for _, a := range unpreparedClone {
		for _, s := range stats.GetData() {
			if a.GetId() != s.GetId() {
				continue
			}

			currentVersion, err := ParseVersionString(s.GetApplianceVersion())
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
			if res, _ := CompareVersionsAndBuildNumber(highestPreparedVersion, currentVersion); res == 0 {
				alreadySameVersion = append(alreadySameVersion, a)
			}
		}
	}
	// if something went wrong during version identification
	// just return the errors
	if errs != nil {
		return nil, errs.ErrorOrNil()
	}

	// If unprepared controllers are already upgraded, remove them from unpreparedController
	if len(alreadySameVersion) > 0 {
		unpreparedControllers = util.Filter(unpreparedControllers, func(a openapi.Appliance) bool {
			return util.InSliceFunc(a, alreadySameVersion, func(i openapi.Appliance, c openapi.Appliance) bool {
				return i.GetId() != c.GetId()
			})
		})
	}

	// If this is true, no need to check anymore
	// we ignore offline controllers at this point
	if (totalControllers - len(offlineControllers)) == (len(preparedControllers) + len(alreadySameVersion)) {
		return nil, nil
	}

	if isMajorOrMinor && len(mismatchControllers) > 0 {
		return mismatchControllers, ErrControllerVersionMismatch
	}

	// we will only upgrade-complete the prepared controllers
	// return the unprepared controllers without error
	if isMajorOrMinor && len(unpreparedControllers) > 0 {
		return unpreparedControllers, nil
	}

	return nil, nil
}

func NeedsMultiControllerUpgrade(upgradeStatuses map[string]UpgradeStatusResult, initialStatData []openapi.ApplianceWithStatus, all, preparing []openapi.Appliance, majorOrMinor bool) (bool, error) {
	controllerCount := controllerCount(all)
	controllerPrepareCount := 0
	alreadySameVersion := 0
	unpreparedCurrentVersions := []*version.Version{}
	var highestPreparedVersion *version.Version
	for _, a := range preparing {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			if ugs := upgradeStatuses[a.GetId()]; ugs.Status == UpgradeStatusReady {
				preparedVersion, err := ParseVersionString(ugs.Details)
				if err != nil {
					return false, err
				}
				if res, _ := CompareVersionsAndBuildNumber(highestPreparedVersion, preparedVersion); highestPreparedVersion == nil || res >= 1 {
					highestPreparedVersion = preparedVersion
				}
				controllerPrepareCount++
			} else {
				for _, data := range initialStatData {
					if data.GetId() != a.GetId() {
						continue
					}
					currentVersion, err := ParseVersionString(data.GetApplianceVersion())
					if err != nil {
						return false, err
					}
					unpreparedCurrentVersions = append(unpreparedCurrentVersions, currentVersion)
				}
			}
		}
	}
	if controllerCount != controllerPrepareCount {
		for _, uv := range unpreparedCurrentVersions {
			if v, _ := CompareVersionsAndBuildNumber(highestPreparedVersion, uv); v == 0 {
				alreadySameVersion++
			}
		}
	}
	return (controllerCount != controllerPrepareCount+alreadySameVersion) && majorOrMinor, nil
}

var disallowedVersionUpgrades map[string][]string = map[string][]string{
	"6.0.0+estimated":   {"6.2.0+estimated"},
	">=6.3.5+estimated": {"6.4.0+estimated"},
}

func CheckApplianceVersionsDisallowed(currentVersion, targetVersion *version.Version) error {
	for k, v := range disallowedVersionUpgrades {
		constraint, _ := version.NewConstraint(k)
		// This checks if there a re constraints on the current version running
		if constraint.Check(currentVersion) {
			// If there are constraints, we'll check each constraint against the targetVersion
			for _, dv := range v {
				targetConstraint, _ := version.NewConstraint(dv)
				if targetConstraint.Check(targetVersion) {
					return fmt.Errorf("upgrading from '%s' to '%s' is not allowed", currentVersion, targetVersion)
				}
			}
		}
	}
	return nil
}
