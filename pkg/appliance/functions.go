package appliance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/appgate/sdpctl/pkg/hashcode"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

const (
	FunctionController   = "Controller"
	FunctionGateway      = "Gateway"
	FunctionPortal       = "Portal"
	FunctionConnector    = "Connector"
	FunctionLogServer    = "LogServer"
	FunctionLogForwarder = "LogForwarder"
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
				if StatsIsOnline(stat) {
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

// SplitAppliancesByGroup return a map of slices of appliances based on their active function and site.
// e.g All active gateways in the same site are grouped together.
func SplitAppliancesByGroup(appliances []openapi.Appliance) map[int][]openapi.Appliance {
	result := make(map[int][]openapi.Appliance)
	for _, a := range appliances {
		groupID := applianceGroupHash(a)
		result[groupID] = append(result[groupID], a)
	}
	return result
}

// ChunkApplianceGroup separates the result from SplitAppliancesByGroup into different slices based on the appliance
// functions and site configuration
func ChunkApplianceGroup(chunkSize int, applianceGroups map[int][]openapi.Appliance) [][]openapi.Appliance {
	logrus.WithFields(logrus.Fields{
		"chunkSize":       chunkSize,
		"applianceGroups": applianceGroups,
	}).Debug("Group input log")
	if chunkSize == 0 {
		chunkSize = 2
	}
	// var chunks [][]openapi.Appliance
	chunks := make([][]openapi.Appliance, chunkSize)
	for i := range chunks {
		chunks[i] = make([]openapi.Appliance, 0)
	}
	// for consistency, we need to sort all input and output slices to generate a consistent result
	for id := range applianceGroups {
		sort.Slice(applianceGroups[id], func(i, j int) bool {
			return applianceGroups[id][i].GetName() < applianceGroups[id][j].GetName()
		})
	}

	keys := make([]int, 0, len(applianceGroups))
	for k := range applianceGroups {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	count := 0
	for _, slice := range applianceGroups {
		for range slice {
			count += 1
		}
	}

	for i := 0; i <= count; i++ {
		// select which initial slice we are going to put the appliance in
		// the appliance may be moved later if the slice ends up to big.
		index := i % chunkSize
		chunk := chunks[index]
		for _, groupID := range keys {
			slice := applianceGroups[groupID]
			if len(slice) > 0 {
				item, slice := slice[len(slice)-1], slice[:len(slice)-1]
				applianceGroups[groupID] = slice
				temp := make([]openapi.Appliance, 0)
				temp = append(temp, item)
				chunk = append(chunk, temp...)
			}
		}
		chunks[index] = chunk
	}

	// make sure we sort each slice for a consistent output and remove any empty slices.
	var r [][]openapi.Appliance
	for index := range chunks {
		sort.Slice(chunks[index], func(i, j int) bool {
			return chunks[index][i].GetName() < chunks[index][j].GetName()
		})

		if len(chunks[index]) > 0 {
			r = append(r, chunks[index])
		}
	}

	for i, c := range r {
		batchAppliances := []string{}
		for _, bapp := range c {
			batchAppliances = append(batchAppliances, bapp.GetName())
		}
		logrus.Infof("[%d] Appliance upgrade chunk includes %v", i, strings.Join(batchAppliances, ", "))
	}

	return r
}

// applianceGroupHash return a unique id hash based on the active function of the appliance and their site ID.
func applianceGroupHash(appliance openapi.Appliance) int {
	var buf bytes.Buffer
	if v, ok := appliance.GetControllerOk(); ok {
		if enabled := v.GetEnabled(); enabled {
			buf.WriteString(fmt.Sprintf("%s=%t", "controller", enabled))
			// we want to group all controllers to the same group
			return hashcode.String(buf.String())
		}
	}
	if len(appliance.GetSite()) > 0 {
		buf.WriteString(appliance.GetSite())
	}
	if v, ok := appliance.GetLogForwarderOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&log_forwarder", v.GetEnabled()))
	}
	if v, ok := appliance.GetLogServerOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&log_server", v.GetEnabled()))
	}
	if v, ok := appliance.GetGatewayOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&gateway", v.GetEnabled()))
	}
	if v, ok := appliance.GetConnectorOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&connector", v.GetEnabled()))
	}
	if v, ok := appliance.GetPortalOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&portal", v.GetEnabled()))
	}

	str := buf.String()
	hash := hashcode.String(str)
	logrus.WithFields(logrus.Fields{
		"string":   str,
		"hashcode": hash,
	}).Debug("generated hash group")
	return hash
}

func ActiveSitesInAppliances(slice []openapi.Appliance) int {
	keys := make(map[string]bool)
	for _, a := range slice {
		if v, ok := a.GetSiteOk(); ok {
			if _, ok := keys[*v]; !ok {
				keys[*v] = true
			}
		}
	}
	return len(keys)
}

func GetApplianceVersion(appliance openapi.Appliance, stats openapi.StatsAppliancesList) (*version.Version, error) {
	for _, s := range stats.GetData() {
		if s.GetId() == appliance.GetId() {
			return ParseVersionString(s.GetVersion())
		}
	}
	return nil, fmt.Errorf("could not determine appliance version %s", appliance.GetName())
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
		if err := ValidateHostname(*candidate, hostname); err != nil {
			return nil, err
		}
		return candidate, nil
	}
	return nil, fmt.Errorf(
		"Unable to match the given Controller hostname %q with the actual Controller admin (or peer) hostname",
		hostname,
	)
}

func GetRealHostname(controller openapi.Appliance) (string, error) {
	realHost := controller.GetHostname()
	if i, ok := controller.GetPeerInterfaceOk(); ok {
		realHost = i.GetHostname()
	}
	if i, ok := controller.GetAdminInterfaceOk(); ok {
		realHost = i.GetHostname()
	}
	if err := ValidateHostname(controller, realHost); err != nil {
		return "", err
	}
	return realHost, nil
}

func ValidateHostname(controller openapi.Appliance, hostname string) error {
	var h string
	if ai, ok := controller.GetAdminInterfaceOk(); ok {
		h = ai.GetHostname()
	}
	if pi, ok := controller.GetPeerInterfaceOk(); ok && len(h) <= 0 {
		h = pi.GetHostname()
	}
	if len(h) <= 0 {
		return fmt.Errorf("failed to determine hostname for controller admin interface")
	}

	cHost := strings.ToLower(h)
	nHost := strings.ToLower(hostname)
	if cHost != nHost {
		logrus.WithFields(logrus.Fields{
			"controller-hostname": cHost,
			"connected-hostname":  nHost,
		}).Error("no match")
		return fmt.Errorf("Hostname validation failed. Pass the --actual-hostname flag to use the real controller hostname")
	}

	var errs error
	errCount := 0
	ctx := context.Background()
	ipv4s, err := net.DefaultResolver.LookupIP(ctx, "ip4", nHost)
	if err != nil {
		errCount++
		err = fmt.Errorf("ipv4: %w", err)
		errs = multierror.Append(err, errs)
	}
	ipv6s, err := net.DefaultResolver.LookupIP(ctx, "ip6", nHost)
	if err != nil {
		errCount++
		err = fmt.Errorf("ipv6: %w", err)
		errs = multierror.Append(err, errs)
	}
	// We check errors, but only one needs to succeed, so we also count the errors before determining if we return an error
	if errs != nil && errCount > 1 {
		return errs
	}
	v4length := len(ipv4s)
	v6length := len(ipv6s)
	if v4length > 1 || v6length > 1 {
		var ips []string
		for _, i := range ipv4s {
			ips = append(ips, i.String())
		}
		for _, i := range ipv6s {
			ips = append(ips, i.String())
		}
		return fmt.Errorf(`The given hostname %s does not resolve to a unique IP.
A unique IP is necessary to ensure that the upgrade scipt
connects to only the (primary) administration controller.
The hostname resolves to the following IPs: %s`, nHost, strings.Join(ips, ", "))
	}

	return nil
}

func FindCurrentController(appliances []openapi.Appliance, hostname string) (*openapi.Appliance, error) {
	l := strings.ToLower(hostname)
	for _, a := range appliances {
		hostnames := []string{}
		hostnames = append(hostnames, strings.ToLower(a.GetHostname()))
		if v, ok := a.GetAdminInterfaceOk(); ok {
			hostnames = append(hostnames, strings.ToLower(v.GetHostname()))
		}
		if v, ok := a.GetPeerInterfaceOk(); ok {
			hostnames = append(hostnames, strings.ToLower(v.GetHostname()))
		}
		if util.InSlice(l, hostnames) {
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

var DefaultCommandFilter = map[string]map[string]string{
	"include": {},
	"exclude": {},
}

func FilterAppliances(appliances []openapi.Appliance, filter map[string]map[string]string) []openapi.Appliance {
	result := make([]openapi.Appliance, len(appliances))
	copy(result, appliances)
	// apply normal filter
	if len(filter["include"]) > 0 {
		result = applyApplianceFilter(result, filter["include"])
	}

	// apply exclusion filter
	toExclude := applyApplianceFilter(result, filter["exclude"])
	for _, exa := range toExclude {
		eID := exa.GetId()
		for i, a := range result {
			if eID == a.GetId() {
				result = append(result[:i], result[i+1:]...)
			}
		}
	}

	return result
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
					if res := util.SearchSlice(t, appTags, false); len(res) > 0 {
						appendUnique(a)
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
			case "function":
				functionList := strings.Split(s, FilterDelimiter)
				for _, function := range functionList {
					functions := GetActiveFunctions(a)
					if results := util.SearchSlice(function, functions, true); len(results) > 0 {
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

const (
	// https://github.com/appgate/sdp-api-specification/blob/94d8f7970cd025c8cf92b4560c1a9a0595d66133/dashboard.yml#L477-L483
	statsHealthy      string = "healthy"
	statsBusy         string = "busy"
	statsWarning      string = "warning"
	statsError        string = "error"
	statsNotAvailable string = "n/a"
	statsOffline      string = "offline"
)

var NotBusyStatus = []string{
	statsHealthy,
	statsWarning,
	statsError,
	statsNotAvailable,
	statsOffline,
}

// StatsIsOnline will return true if the controller reports the appliance to be online in a valid status
func StatsIsOnline(s openapi.StatsAppliancesListAllOfData) bool {
	// from appliance 6.0, 'online' field has been removed in favour for status
	// we will keep GetOnline() for backwards compatibility.
	if s.GetOnline() {
		return true
	}
	if util.InSlice(s.GetStatus(), []string{statsNotAvailable, statsOffline}) {
		return false
	}
	// unkown or empty status will report appliance as offline.
	return util.InSlice(s.GetStatus(), []string{statsHealthy, statsBusy, statsWarning, statsError})
}

func ShouldDisable(from, to *version.Version) bool {
	compare, _ := version.NewVersion("5.4")

	if from.LessThan(compare) {
		majorChange := from.Segments()[0] < to.Segments()[0]
		minorChange := from.Segments()[1] < to.Segments()[1]
		return majorChange || minorChange
	}

	return false
}
