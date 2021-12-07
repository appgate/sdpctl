package appliance

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

const showDiskSpaceWarning = `
Some appliances have very little space available
{{range .Stats}}
  - {{ .Name }}  Disk usage: {{ .Disk -}}%
{{end}}

Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.
`

func ShowDiskSpaceWarningMessage(stats []openapi.StatsAppliancesListAllOfData) (string, error) {
	type stub struct {
		Stats []openapi.StatsAppliancesListAllOfData
	}
	data := stub{
		Stats: stats,
	}
	t := template.Must(template.New("").Parse(showDiskSpaceWarning))
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

func HasLowDiskSpace(stats []openapi.StatsAppliancesListAllOfData) bool {
	for _, s := range stats {
		if s.GetDisk() >= 75 {
			return true
		}
	}
	return false
}

func applianceGroupDescription(appliances []openapi.Appliance) string {
	functions := ActiveFunctions(appliances)
	var funcs []string
	for k, value := range functions {
		if _, ok := functions[k]; ok && value {
			funcs = append(funcs, k)
		}
	}
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
