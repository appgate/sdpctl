package appliance

import (
	"fmt"
	"strings"

	util "github.com/appgate/appgatectl/internal"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

// ActiveFunctions returns a map of all active functions in the appliances.
func ActiveFunctions(appliances []openapi.Appliance) map[string]bool {
	functions := make(map[string]bool)
	for _, a := range appliances {
		if v, ok := a.GetControllerOk(); ok && v.GetEnabled() {
			functions["controller"] = true
		}
		if v, ok := a.GetGatewayOk(); ok && v.GetEnabled() {
			functions["gateway"] = true
		}
		if v, ok := a.GetPortalOk(); ok && v.GetEnabled() {
			functions["portal"] = true
		}
		if v, ok := a.GetConnectorOk(); ok && v.GetEnabled() {
			functions["connector"] = true
		}
		if v, ok := a.GetLogServerOk(); ok && v.GetEnabled() {
			functions["log_server"] = true
		}
	}
	return functions
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
