package appliance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

func PrintDiskSpaceWarningMessage(out io.Writer, stats []openapi.StatsAppliancesListAllOfData) {
	p := util.NewPrinter(out, 4)
	p.AddHeader("Name", "Disk Usage")
	for _, a := range stats {
		p.AddLine(a.GetName(), fmt.Sprintf("%v%%", a.GetDisk()))
	}

	fmt.Fprint(out, "\nWARNING: Some appliances have very little space available\n\n")
	p.Print()
	fmt.Fprintln(out, `
Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.`)
}

func HasLowDiskSpace(stats []openapi.StatsAppliancesListAllOfData) []openapi.StatsAppliancesListAllOfData {
	result := []openapi.StatsAppliancesListAllOfData{}
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

func appliancePeerPorts(appliances []openapi.Appliance) string {
	ports := make([]int, 0)
	for _, a := range appliances {
		if v, ok := a.GetPeerInterfaceOk(); ok {
			if v, ok := v.GetHttpsPortOk(); ok && *v > 0 {
				ports = util.AppendIfMissing(ports, int(*v))
			}
		}
	}
	return strings.Trim(strings.Replace(fmt.Sprint(ports), " ", ",", -1), "[]")
}

const applianceUsingPeerWarning = `
Version 5.4 and later are designed to operate with the admin port (default 8443)
separate from the deprecated peer port (set to {{.CurrentPort}}).
It is recommended to switch to port 8443 before continuing
The following {{.Functions}} {{.Noun}} still configured without the Admin/API TLS Connection:
{{range .Appliances}}
  - {{.Name -}}
{{end}}
`

func ShowPeerInterfaceWarningMessage(peerAppliances []openapi.Appliance) (string, error) {
	type stub struct {
		CurrentPort string
		Functions   string
		Noun        string
		Appliances  []openapi.Appliance
	}
	noun := "are"
	if len(peerAppliances) == 1 {
		noun = "is"
	}
	u := unique(peerAppliances)
	data := stub{
		CurrentPort: appliancePeerPorts(u),
		Functions:   applianceGroupDescription(u),
		Noun:        noun,
		Appliances:  u,
	}
	t := template.Must(template.New("peer").Parse(applianceUsingPeerWarning))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
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

// CheckVersionsEqual will check if appliance versions are equal to the version being uploaded on all appliances
// Returns a slice of appliances that are not equal, a slice of appliances that have the same version and an error
func CheckVersionsEqual(ctx context.Context, stats openapi.StatsAppliancesList, appliances []openapi.Appliance, v *version.Version) ([]openapi.Appliance, []openapi.Appliance) {
	skip := []openapi.Appliance{}
	keep := []openapi.Appliance{}

	for _, appliance := range appliances {
		for _, stat := range stats.GetData() {
			if stat.GetId() == appliance.GetId() {
				statV, err := ParseVersionString(stat.GetVersion())
				if err != nil {
					log.Warn("failed to parse version from stats")
					continue
				}
				statBuildNr, _ := strconv.ParseInt(statV.Metadata(), 10, 64)
				uploadBuildNr, _ := strconv.ParseInt(v.Metadata(), 10, 64)
				if statV.Equal(v) && statBuildNr == uploadBuildNr {
					log.WithField("appliance", appliance.GetName()).Info("Appliance is already at the same version. Skipping.")
					skip = append(skip, appliance)
				} else {
					keep = append(keep, appliance)
				}
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
func CompareVersionsAndBuildNumber(x, y *version.Version) int {
	res := y.Compare(x)

	// if res is 0, we also compare build number
	if res == IsEqual {
		buildX, _ := version.NewVersion(x.Metadata())
		buildY, _ := version.NewVersion(y.Metadata())
		res = buildY.Compare(buildX)
	}

	return res
}

func HasDiffVersions(stats []openapi.StatsAppliancesListAllOfData) (bool, map[string]string) {
	res := map[string]string{}
	versionList := []string{}
	for _, stat := range stats {
		statVersionString := stat.GetVersion()
		v, err := ParseVersionString(statVersionString)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"appliance": stat.GetName(),
				"version":   statVersionString,
			}).Warn("failed to parse version string")
			return false, res
		}
		versionString := v.String()
		res[stat.GetName()] = versionString
		versionList = append(versionList, versionString)
	}
	unique := uniqueString(versionList)
	return len(unique) != 1, res
}
