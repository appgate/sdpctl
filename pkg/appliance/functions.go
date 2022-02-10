package appliance

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

const (
	FunctionController   = "controller"
	FunctionGateway      = "gateway"
	FunctionPortal       = "portal"
	FunctionConnector    = "connector"
	FunctionLogServer    = "logserver"
	FunctionLogForwarder = "logforwarder"
	FilterDelimiter      = "&"
)

// GroupByFunctions group appliances by function
func GroupByFunctions(appliances []openapi.Appliance) map[string][]openapi.Appliance {
	r := make(map[string][]openapi.Appliance)
	for _, a := range appliances {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			r[FunctionController] = append(r[FunctionController], a)
		}
		if v, ok := a.GetGatewayOk(); ok && v.GetEnabled() {
			r[FunctionGateway] = append(r[FunctionGateway], a)
		}
		if v, ok := a.GetPortalOk(); ok && v.GetEnabled() {
			r[FunctionPortal] = append(r[FunctionPortal], a)
		}
		if v, ok := a.GetConnectorOk(); ok && v.GetEnabled() {
			r[FunctionConnector] = append(r[FunctionConnector], a)
		}
		if v, ok := a.GetLogServerOk(); ok && v.GetEnabled() {
			r[FunctionLogServer] = append(r[FunctionLogServer], a)
		}
		if v, ok := a.GetLogForwarderOk(); ok && v.GetEnabled() {
			r[FunctionLogForwarder] = append(r[FunctionLogForwarder], a)
		}
	}
	return r
}

// ActiveFunctions returns a map of all active functions in the appliances.
func ActiveFunctions(appliances []openapi.Appliance) map[string]bool {
	functions := make(map[string]bool)
	for _, a := range appliances {
		res := GetActiveFunctions(a)
		if util.InSlice(FunctionController, res) {
			functions[FunctionController] = true
		}
		if util.InSlice(FunctionGateway, res) {
			functions[FunctionGateway] = true
		}
		if util.InSlice(FunctionPortal, res) {
			functions[FunctionPortal] = true
		}
		if util.InSlice(FunctionConnector, res) {
			functions[FunctionConnector] = true
		}
		if util.InSlice(FunctionLogServer, res) {
			functions[FunctionLogServer] = true
		}
		if util.InSlice(FunctionLogForwarder, res) {
			functions[FunctionLogForwarder] = true
		}
	}
	return functions
}

func GetActiveFunctions(appliance openapi.Appliance) []string {
	functions := []string{}

	if v, ok := appliance.GetControllerOk(); ok && v.GetEnabled() {
		functions = append(functions, FunctionController)
	}
	if v, ok := appliance.GetGatewayOk(); ok && v.GetEnabled() {
		functions = append(functions, FunctionGateway)
	}
	if v, ok := appliance.GetPortalOk(); ok && v.GetEnabled() {
		functions = append(functions, FunctionPortal)
	}
	if v, ok := appliance.GetConnectorOk(); ok && v.GetEnabled() {
		functions = append(functions, FunctionConnector)
	}
	if v, ok := appliance.GetLogServerOk(); ok && v.GetEnabled() {
		functions = append(functions, FunctionLogServer)
	}
	if v, ok := appliance.GetLogForwarderOk(); ok && v.GetEnabled() {
		functions = append(functions, FunctionLogForwarder)
	}

	return functions
}

// WithAdminOnPeerInterface List all appliances still using the peer interface for the admin API, this is now deprecated.
func WithAdminOnPeerInterface(appliances []openapi.Appliance) []openapi.Appliance {
	peer := make([]openapi.Appliance, 0)
	for _, a := range appliances {
		if _, ok := a.GetAdminInterfaceOk(); !ok {
			peer = append(peer, a)
		}
	}
	return peer
}

// FilterAvailable return lists of online, offline, errors that will be used during upgrade
func FilterAvailable(appliances []openapi.Appliance, stats []openapi.StatsAppliancesListAllOfData) ([]openapi.Appliance, []openapi.Appliance, error) {
	result := make([]openapi.Appliance, 0)
	offline := make([]openapi.Appliance, 0)
	var err error
	// filter out offline appliances
	for _, a := range appliances {
		for _, stat := range stats {
			if a.GetId() == stat.GetId() {
				if stat.GetOnline() {
					result = append(result, a)
				} else {
					offline = append(offline, a)
				}
			}
		}
	}
	for _, a := range offline {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			err = multierror.Append(err, fmt.Errorf("cannot start the operation since a controller %q is offline.", a.GetName()))
		}
		if v, ok := a.GetLogServerOk(); ok && v.GetEnabled() {
			err = multierror.Append(err, fmt.Errorf("cannot start the operation since a logserver %q is offline.", a.GetName()))
		}
	}
	return result, offline, err
}

func GetPrimaryControllerVersion(primary openapi.Appliance, stats openapi.StatsAppliancesList) (*version.Version, error) {
	for _, s := range stats.GetData() {
		if s.GetId() == primary.GetId() {
			return version.NewVersion(s.GetVersion())
		}
	}
	return nil, fmt.Errorf("could not determine appliance version of the primary controller %s", primary.GetName())
}

// FindPrimaryController The given hostname should match one of the controller's actual admin hostname.
// Hostnames should be compared in a case insensitive way.
func FindPrimaryController(appliances []openapi.Appliance, hostname string) (*openapi.Appliance, error) {
	controllers := make([]openapi.Appliance, 0)
	type details struct {
		ID        string
		Hostnames []string
		Appliance openapi.Appliance
	}
	data := make(map[string]details)
	for _, a := range appliances {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			controllers = append(controllers, a)
		}
	}
	for _, controller := range controllers {
		var hostnames []string
		hostnames = append(hostnames, strings.ToLower(controller.GetPeerInterface().Hostname))
		if v, ok := controller.GetAdminInterfaceOk(); ok {
			hostnames = append(hostnames, strings.ToLower(v.GetHostname()))
		}
		if v, ok := controller.GetPeerInterfaceOk(); ok {
			hostnames = append(hostnames, strings.ToLower(v.GetHostname()))
		}
		data[controller.GetId()] = details{
			ID:        controller.GetId(),
			Hostnames: hostnames,
			Appliance: controller,
		}
	}
	count := 0
	var candidate *openapi.Appliance
	for _, c := range data {
		if util.InSlice(strings.ToLower(hostname), c.Hostnames) {
			count++
			candidate = &c.Appliance
			break
		}
	}
	if count > 1 {
		return nil, fmt.Errorf(
			"The given Controller hostname %s is used by more than one appliance."+
				"A unique Controller admin (or peer) hostname is required to perform the upgrade.",
			hostname,
		)
	}
	if candidate != nil {
		return candidate, nil
	}
	return nil, fmt.Errorf(
		"Unable to match the given Controller hostname %q with the actual Controller admin (or peer) hostname",
		hostname,
	)
}

func FindCurrentController(appliances []openapi.Appliance, hostname string) (*openapi.Appliance, error) {
	for _, a := range appliances {
		if a.GetHostname() == hostname {
			return &a, nil
		}
	}
	return nil, errors.New("No host controller found")
}

// AutoscalingGateways return the template appliance and all gateways
func AutoscalingGateways(appliances []openapi.Appliance) (*openapi.Appliance, []openapi.Appliance) {
	autoscalePrefix := "Autoscaling Instance"
	var template *openapi.Appliance
	r := make([]openapi.Appliance, 0)
	for _, a := range appliances {
		if util.InSlice("template", a.GetTags()) && !a.GetActivated() {
			template = &a
		}
		if v, ok := a.GetGatewayOk(); ok && v.GetEnabled() && strings.HasPrefix(a.GetName(), autoscalePrefix) {
			r = append(r, a)
		}
	}
	return template, r
}

func FilterAppliances(appliances []openapi.Appliance, filter map[string]map[string]string) []openapi.Appliance {
	// apply normal filter
	if len(filter["filter"]) > 0 {
		appliances = applyApplianceFilter(appliances, filter["filter"])
	}

	// apply exclusion filter
	toExclude := applyApplianceFilter(appliances, filter["exclude"])
	for _, exa := range toExclude {
		eID := exa.GetId()
		for i, a := range appliances {
			if eID == a.GetId() {
				appliances = append(appliances[:i], appliances[i+1:]...)
			}
		}
	}

	return appliances
}

func applyApplianceFilter(appliances []openapi.Appliance, filter map[string]string) []openapi.Appliance {
	var filteredAppliances []openapi.Appliance
	var warnings []string

	appendUnique := func(app openapi.Appliance) {
		appID := app.GetId()
		inFiltered := []string{}
		for _, a := range filteredAppliances {
			inFiltered = append(inFiltered, a.GetId())
		}
		if !util.InSlice(appID, inFiltered) {
			filteredAppliances = append(filteredAppliances, app)
		}
	}

	for _, a := range appliances {
		for k, s := range filter {
			switch k {
			case "name":
				nameList := strings.Split(s, FilterDelimiter)
				for _, name := range nameList {
					regex := regexp.MustCompile(name)
					if regex.MatchString(a.GetName()) {
						appendUnique(a)
					}
				}
			case "id":
				ids := strings.Split(s, FilterDelimiter)
				for _, id := range ids {
					regex := regexp.MustCompile(id)
					if regex.MatchString(a.GetId()) {
						appendUnique(a)
					}
				}
			case "tags", "tag":
				tagSlice := strings.Split(s, FilterDelimiter)
				appTags := a.GetTags()
				for _, t := range tagSlice {
					regex := regexp.MustCompile(t)
					for _, at := range appTags {
						if regex.MatchString(at) {
							appendUnique(a)
						}
					}
				}
			case "version":
				vList := strings.Split(s, FilterDelimiter)
				for _, v := range vList {
					regex := regexp.MustCompile(v)
					version := a.GetVersion()
					versionString := fmt.Sprintf("%d", version)
					if regex.MatchString(versionString) {
						appendUnique(a)
					}
				}
			case "hostname", "host":
				hostList := strings.Split(s, FilterDelimiter)
				for _, host := range hostList {
					regex := regexp.MustCompile(host)
					if regex.MatchString(a.GetHostname()) {
						appendUnique(a)
					}
				}
			case "active", "activated":
				b, err := strconv.ParseBool(s)
				if err != nil {
					message := fmt.Sprintf("Failed to parse boolean filter value: %x", err)
					if !util.InSlice(message, warnings) {
						warnings = append(warnings, message)
					}
				}
				if a.GetActivated() == b {
					appendUnique(a)
				}
			case "site", "site-id":
				siteList := strings.Split(s, FilterDelimiter)
				for _, site := range siteList {
					regex := regexp.MustCompile(site)
					if regex.MatchString(a.GetSite()) {
						appendUnique(a)
					}
				}
			case "function", "roles", "role":
				roleList := strings.Split(s, FilterDelimiter)
				for _, role := range roleList {
					if functions := GetActiveFunctions(a); util.InSlice(role, functions) {
						appendUnique(a)
					}
				}
			default:
				message := fmt.Sprintf("'%s' is not a filterable keyword. Ignoring.", k)
				if !util.InSlice(message, warnings) {
					warnings = append(warnings, message)
				}
			}
		}
	}

	if len(warnings) > 0 {
		for _, warn := range warnings {
			logrus.Warnf(warn)
		}
	}

	return filteredAppliances
}
