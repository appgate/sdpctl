package appliance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
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

	if len(filter["exclude"]) <= 0 {
		return appliances
	}

	// apply exclusion filter
	filtered := []openapi.Appliance{}
	toExclude := applyApplianceFilter(appliances, filter["exclude"])
	for _, appliance := range appliances {
		aID := appliance.GetId()
		for _, exa := range toExclude {
			eID := exa.GetId()
			if aID != eID {
				filtered = append(filtered, appliance)
			}
		}
	}

	return filtered
}

func applyApplianceFilter(appliances []openapi.Appliance, filter map[string]string) []openapi.Appliance {
	var filteredAppliances []openapi.Appliance
	for _, a := range appliances {
		for k, s := range filter {
			switch k {
			case "name":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetName()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "id":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetId()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "tags", "tag":
				tagSlice := strings.Split(s, ",")
				appTags := a.GetTags()
				for _, t := range tagSlice {
					regex := regexp.MustCompile(t)
					for _, at := range appTags {
						if regex.MatchString(at) {
							filteredAppliances = append(filteredAppliances, a)
						}
					}
				}
			case "version":
				regex := regexp.MustCompile(s)
				version := a.GetVersion()
				versionString := fmt.Sprintf("%d", version)
				if regex.MatchString(versionString) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "hostname", "host":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetHostname()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "active", "activated":
				b, err := strconv.ParseBool(s)
				if err != nil {
					logrus.Warnf("Failed to parse boolean filter value: %x", err)
				}
				if a.GetActivated() == b {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "site", "site-id":
				regex := regexp.MustCompile(s)
				if regex.MatchString(a.GetSite()) {
					filteredAppliances = append(filteredAppliances, a)
				}
			case "function", "roles", "role":
				if functions := GetActiveFunctions(a); util.InSlice(s, functions) {
					filteredAppliances = append(filteredAppliances, a)
				}

			}
		}
	}
	return filteredAppliances
}
