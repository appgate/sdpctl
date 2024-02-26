package appliance

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/hashcode"
	"github.com/appgate/sdpctl/pkg/network"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v8"
	"golang.org/x/sync/errgroup"
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
			err = multierror.Append(err, fmt.Errorf("Cannot start the operation since a Controller %q is offline", a.GetName()))
		}
		if v, ok := a.GetLogServerOk(); ok && v.GetEnabled() {
			err = multierror.Append(err, fmt.Errorf("Cannot start the operation since a LogServer %q is offline", a.GetName()))
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
	if chunkSize <= 0 {
		chunkSize = 2
	}

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

	candidates := make([]openapi.Appliance, 0)
	for index, slice := range chunks {
		if len(slice) == 1 {
			item, org := slice[len(slice)-1], slice[:len(slice)-1]
			if v, ok := item.GetGatewayOk(); ok && v.GetEnabled() {
				// needed so that we don't accidentally add same site gateways to the same batch
				continue
			}
			if v, ok := item.GetConnectorOk(); ok && v.GetEnabled() {
				continue
			}
			chunks[index] = org
			candidates = append(candidates, item)
		}
	}
	chunks = append(chunks, candidates)
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
	if v, ok := appliance.GetGatewayOk(); ok {
		enabled := v.GetEnabled()
		buf.WriteString(fmt.Sprintf("%s=%t", "&gateway", enabled))
		if enabled {
			// all enabled gateways with same site need to be grouped together regardless of other functions enabled
			return hashcode.String(buf.String())
		}
	}
	if v, ok := appliance.GetConnectorOk(); ok {
		enabled := v.GetEnabled()
		buf.WriteString(fmt.Sprintf("%s=%t", "&connector", enabled))
		if enabled {
			// same rules apply to connectors as gateways
			return hashcode.String(buf.String())
		}
	}
	if v, ok := appliance.GetLogForwarderOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&log_forwarder", v.GetEnabled()))
	}
	if v, ok := appliance.GetLogServerOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&log_server", v.GetEnabled()))
	}
	if v, ok := appliance.GetPortalOk(); ok {
		buf.WriteString(fmt.Sprintf("%s=%t", "&portal", v.GetEnabled()))
	}

	return hashcode.String(buf.String())
}

func GetApplianceVersion(appliance openapi.Appliance, stats openapi.StatsAppliancesList) (*version.Version, error) {
	for _, s := range stats.GetData() {
		if s.GetId() == appliance.GetId() {
			if !StatsIsOnline(s) {
				return nil, fmt.Errorf("can't get current version of %s, the appliance is offline", appliance.GetName())
			}
			return ParseVersionString(s.GetVersion())
		}
	}
	return nil, fmt.Errorf("Could not determine appliance version %s", appliance.GetName())
}

// FindPrimaryController The given hostname should match one of the Controller's actual admin hostname.
// Hostnames should be compared in a case insensitive way.
func FindPrimaryController(appliances []openapi.Appliance, hostname string, validate bool) (*openapi.Appliance, error) {
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
			break
		}
	}
	if count > 1 {
		return nil, fmt.Errorf(
			"The given Controller hostname %s is used by more than one appliance"+
				"A unique Controller admin (or peer) hostname is required to perform the upgrade",
			hostname,
		)
	}
	if candidate != nil {
		if validate {
			if err := ValidateHostname(*candidate, hostname); err != nil {
				return nil, err
			}
		}
		log.WithField("appliance", candidate.GetName()).Info("primary controller")
		return candidate, nil
	}
	return nil, fmt.Errorf(
		"Unable to match the given Controller hostname %q with the actual Controller admin (or peer) hostname",
		hostname,
	)
}

func ValidateHostname(controller openapi.Appliance, hostname string) error {
	var h string
	if ai, ok := controller.GetAdminInterfaceOk(); ok {
		h = ai.GetHostname()
	}
	if len(h) <= 0 {
		return fmt.Errorf("Failed to determine hostname for the Controller admin interface")
	}

	cHost := strings.ToLower(h)
	nHost := strings.ToLower(hostname)
	if cHost != nHost {
		log.WithFields(log.Fields{
			"controller-hostname": cHost,
			"connected-hostname":  nHost,
		}).Error("no match")
		return fmt.Errorf("Hostname validation failed. Pass the --actual-hostname flag to use the real Controller hostname")
	}

	if err := network.ValidateHostnameUniqueness(nHost); err != nil {
		return err
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
		if util.InSlice(l, hostnames) {
			return &a, nil
		}
	}
	return nil, errors.New("No host Controller found")
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

func FilterAppliances(appliances []openapi.Appliance, filter map[string]map[string]string, orderBy []string, descending bool) ([]openapi.Appliance, []openapi.Appliance, error) {
	include := make([]openapi.Appliance, len(appliances))
	copy(include, appliances)
	var errs *multierror.Error
	var err error

	// Keep track of which appliances are filtered at the different steps
	notInclude := make(map[string]openapi.Appliance, len(appliances))
	for _, a := range appliances {
		notInclude[a.GetId()] = a
	}

	// apply normal filter
	if len(filter["include"]) > 0 {
		include, err = applyApplianceFilter(include, filter["include"])
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	for _, i := range include {
		delete(notInclude, i.GetId())
	}

	// apply exclusion filter
	exclude, err := applyApplianceFilter(include, filter["exclude"])
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	for _, exa := range exclude {
		eID := exa.GetId()
		for i, a := range include {
			if eID == a.GetId() {
				include = append(include[:i], include[i+1:]...)
			}
		}
	}

	for _, a := range notInclude {
		exclude = AppendUniqueAppliance(exclude, a)
	}

	// Sort appliances
	include, err = orderAppliances(include, orderBy, descending)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	exclude, err = orderAppliances(exclude, orderBy, descending)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	return include, exclude, errs.ErrorOrNil()
}

func AppendUniqueAppliance(appliances []openapi.Appliance, appliance openapi.Appliance) []openapi.Appliance {
	filteredAppliances := make([]openapi.Appliance, len(appliances))
	copy(filteredAppliances, appliances)

	appID := appliance.GetId()
	inFiltered := []string{}
	for _, a := range filteredAppliances {
		inFiltered = append(inFiltered, a.GetId())
	}
	if !util.InSlice(appID, inFiltered) {
		filteredAppliances = append(filteredAppliances, appliance)
	}

	return filteredAppliances
}

func AppendUniqueApplianceStats(stats []openapi.StatsAppliancesListAllOfData, stat openapi.StatsAppliancesListAllOfData) []openapi.StatsAppliancesListAllOfData {
	filtered := make([]openapi.StatsAppliancesListAllOfData, len(stats))
	copy(filtered, stats)

	appID := stat.GetId()
	inFiltered := []string{}
	for _, a := range filtered {
		inFiltered = append(inFiltered, a.GetId())
	}
	if !util.InSlice(appID, inFiltered) {
		filtered = append(filtered, stat)
	}

	return filtered
}

func applyApplianceFilter(appliances []openapi.Appliance, filter map[string]string) ([]openapi.Appliance, error) {
	var filteredAppliances []openapi.Appliance
	var warnings []string

	for _, a := range appliances {
		for k, s := range filter {
			switch k {
			case "name":
				nameList := strings.Split(s, FilterDelimiter)
				for _, name := range nameList {
					regex, err := regexp.Compile(name)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(a.GetName()) {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			case "id":
				ids := strings.Split(s, FilterDelimiter)
				for _, id := range ids {
					regex, err := regexp.Compile(id)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(a.GetId()) {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			case "tags", "tag":
				tagSlice := strings.Split(s, FilterDelimiter)
				appTags := a.GetTags()
				for _, t := range tagSlice {
					if res := util.SearchSlice(t, appTags, false); len(res) > 0 {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			case "version":
				vList := strings.Split(s, FilterDelimiter)
				for _, v := range vList {
					regex, err := regexp.Compile(v)
					if err != nil {
						return nil, err
					}
					version := a.GetVersion()
					versionString := fmt.Sprintf("%d", version)
					if regex.MatchString(versionString) {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			case "hostname", "host":
				hostList := strings.Split(s, FilterDelimiter)
				for _, host := range hostList {
					regex, err := regexp.Compile(host)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(a.GetHostname()) {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			case "active", "activated":
				b, err := strconv.ParseBool(s)
				if err != nil {
					return nil, err
				}
				if a.GetActivated() == b {
					filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
				}
			case "site", "site-id":
				siteList := strings.Split(s, FilterDelimiter)
				for _, site := range siteList {
					regex, err := regexp.Compile(site)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(a.GetSite()) {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			case "function":
				functionList := strings.Split(s, FilterDelimiter)
				for _, function := range functionList {
					functions := GetActiveFunctions(a)
					if results := util.SearchSlice(function, functions, true); len(results) > 0 {
						filteredAppliances = AppendUniqueAppliance(filteredAppliances, a)
					}
				}
			default:
				message := fmt.Sprintf("'%s' is not a filterable keyword. Ignoring", k)
				if !util.InSlice(message, warnings) {
					warnings = append(warnings, message)
				}
			}
		}
	}

	if len(warnings) > 0 {
		for _, warn := range warnings {
			log.Warnf(warn)
		}
	}

	return filteredAppliances, nil
}

func orderAppliances(appliances []openapi.Appliance, orderBy []string, descending bool) ([]openapi.Appliance, error) {
	var errs *multierror.Error
	// reverse loop the slice to prioritize the ordering. First entered has priority
	for i := len(orderBy) - 1; i >= 0; i-- {
		switch strings.ToLower(orderBy[i]) {
		case "name":
			sort.SliceStable(appliances, func(i, j int) bool { return appliances[i].GetName() < appliances[j].GetName() })
		case "id":
			sort.SliceStable(appliances, func(i, j int) bool { return appliances[i].GetId() < appliances[j].GetId() })
		case "site-id":
			sort.SliceStable(appliances, func(i, j int) bool { return appliances[i].GetSite() < appliances[j].GetSite() })
		case "site-name", "site":
			sort.SliceStable(appliances, func(i, j int) bool { return appliances[i].GetSiteName() < appliances[j].GetSiteName() })
		case "hostname", "host":
			sort.SliceStable(appliances, func(i, j int) bool { return appliances[i].GetHostname() < appliances[j].GetHostname() })
		case "activated", "active":
			sort.SliceStable(appliances, func(i, j int) bool {
				return appliances[i].GetActivated() && appliances[i].GetActivated() != appliances[j].GetActivated()
			})
		default:
			errs = multierror.Append(errs, fmt.Errorf("keyword not sortable: %s", orderBy[i]))
		}
	}
	if descending {
		return util.Reverse(appliances), errs.ErrorOrNil()
	}
	return appliances, errs.ErrorOrNil()
}

func FilterApplianceStats(stats []openapi.StatsAppliancesListAllOfData, filter map[string]map[string]string, orderBy []string, descending bool) ([]openapi.StatsAppliancesListAllOfData, []openapi.StatsAppliancesListAllOfData, error) {
	include := make([]openapi.StatsAppliancesListAllOfData, len(stats))
	copy(include, stats)
	var errs *multierror.Error
	var err error

	// Keep track of which appliances are filtered at the different steps
	notInclude := make(map[string]openapi.StatsAppliancesListAllOfData, len(stats))
	for _, a := range stats {
		notInclude[a.GetId()] = a
	}

	// apply normal filter
	if len(filter["include"]) > 0 {
		include, err = applyApplianceStatsFilter(include, filter["include"])
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	for _, i := range include {
		delete(notInclude, i.GetId())
	}

	// apply exclusion filter
	exclude, err := applyApplianceStatsFilter(include, filter["exclude"])
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	for _, exa := range exclude {
		eID := exa.GetId()
		for i, a := range include {
			if eID == a.GetId() {
				include = append(include[:i], include[i+1:]...)
			}
		}
	}

	for _, a := range notInclude {
		exclude = AppendUniqueApplianceStats(exclude, a)
	}

	// Sort appliances
	include, err = orderApplianceStats(include, orderBy, descending)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	exclude, err = orderApplianceStats(exclude, orderBy, descending)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	return include, exclude, errs.ErrorOrNil()
}

func applyApplianceStatsFilter(stats []openapi.StatsAppliancesListAllOfData, filter map[string]string) ([]openapi.StatsAppliancesListAllOfData, error) {
	var filtered []openapi.StatsAppliancesListAllOfData
	var warnings []string

	for _, s := range stats {
		for k, v := range filter {
			switch k {
			case "name":
				nameList := strings.Split(v, FilterDelimiter)
				for _, name := range nameList {
					regex, err := regexp.Compile(name)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(s.GetName()) {
						filtered = AppendUniqueApplianceStats(filtered, s)
					}
				}
			case "id":
				idList := strings.Split(v, FilterDelimiter)
				for _, id := range idList {
					regex, err := regexp.Compile(id)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(s.GetId()) {
						filtered = append(filtered, s)
					}
				}
			case "status":
				statusList := strings.Split(v, FilterDelimiter)
				for _, status := range statusList {
					regex, err := regexp.Compile(status)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(s.GetStatus()) {
						filtered = append(filtered, s)
					}
				}
			case "state":
				stateList := strings.Split(v, FilterDelimiter)
				for _, state := range stateList {
					regex, err := regexp.Compile(state)
					if err != nil {
						return nil, err
					}
					if regex.MatchString(s.GetState()) {
						filtered = append(filtered, s)
					}
				}
			case "function":
				functionList := strings.Split(v, FilterDelimiter)
				for _, function := range functionList {
					active := strings.Split(StatsActiveFunctions(s), ",")
					if util.InSlice(function, active) {
						filtered = append(filtered, s)
					}
				}
			default:
				msg := fmt.Sprintf("'%s' is not a filterable keyword. Ignoring", k)
				if !util.InSlice(msg, warnings) {
					warnings = append(warnings, msg)
				}
			}
		}
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			log.Warnln(w)
		}
	}

	return filtered, nil
}

func orderApplianceStats(stats []openapi.StatsAppliancesListAllOfData, orderBy []string, descending bool) ([]openapi.StatsAppliancesListAllOfData, error) {
	var errs *multierror.Error
	for i := len(orderBy) - 1; i >= 0; i-- {
		switch strings.ToLower(orderBy[i]) {
		case "name":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetName() < stats[j].GetName() })
		case "disk":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetDisk() < stats[j].GetDisk() })
		case "mem", "memory":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetMemory() < stats[j].GetMemory() })
		case "cpu":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetCpu() < stats[j].GetCpu() })
		case "version":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetVersion() < stats[j].GetVersion() })
		case "net-in":
			sort.SliceStable(stats, func(i, j int) bool {
				inet := stats[i].GetNetwork()
				jnet := stats[j].GetNetwork()
				irx := inet.GetRxSpeed()
				jrx := jnet.GetRxSpeed()
				return irx < jrx
			})
		case "net-out":
			sort.SliceStable(stats, func(i, j int) bool {
				inet := stats[i].GetNetwork()
				jnet := stats[j].GetNetwork()
				itx := inet.GetTxSpeed()
				jtx := jnet.GetTxSpeed()
				return itx < jtx
			})
		case "function":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetFunction() < stats[j].GetFunction() })
		case "status":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetStatus() < stats[j].GetStatus() })
		case "sessions":
			sort.SliceStable(stats, func(i, j int) bool { return stats[i].GetNumberOfSessions() < stats[j].GetNumberOfSessions() })
		default:
			errs = multierror.Append(errs, fmt.Errorf("keyword not sortable: %s", orderBy[i]))
		}
	}
	if descending {
		return util.Reverse(stats), errs.ErrorOrNil()
	}
	return stats, errs.ErrorOrNil()
}

const na = "n/a"

func StatsActiveFunctions(s openapi.StatsAppliancesListAllOfData) string {
	functions := make([]string, 0)
	if v, ok := s.GetLogServerOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, FunctionLogServer)
		}
	}
	if v, ok := s.GetLogForwarderOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, FunctionLogForwarder)
		}
	}
	if v, ok := s.GetControllerOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, FunctionController)
		}
	}
	if v, ok := s.GetConnectorOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, FunctionConnector)
		}
	}
	if v, ok := s.GetGatewayOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, FunctionGateway)
		}
	}
	if v, ok := s.GetPortalOk(); ok {
		if v.GetStatus() != na {
			functions = append(functions, FunctionPortal)
		}
	}
	return strings.Join(functions, ", ")
}

func orderApplianceFiles(files []openapi.File, orderBy []string, descending bool) ([]openapi.File, error) {
	var errs *multierror.Error
	for i := len(orderBy) - 1; i >= 0; i-- {
		switch strings.ToLower(orderBy[i]) {
		case "name":
			sort.SliceStable(files, func(i, j int) bool { return files[i].GetName() < files[j].GetName() })
		case "status":
			sort.SliceStable(files, func(i, j int) bool { return files[i].GetStatus() < files[j].GetStatus() })
		case "failure-reason":
			sort.SliceStable(files, func(i, j int) bool { return files[i].GetFailureReason() < files[j].GetFailureReason() })
		case "creation-time", "created":
			sort.SliceStable(files, func(i, j int) bool {
				iTime := files[i].GetCreationTime()
				jTime := files[j].GetCreationTime()
				return iTime.Before(jTime)
			})
		case "last-modified", "modified":
			sort.SliceStable(files, func(i, j int) bool {
				iTime := files[i].GetLastModifiedTime()
				jTime := files[j].GetLastModifiedTime()
				return iTime.Before(jTime)
			})
		case "checksum":
			sort.SliceStable(files, func(i, j int) bool { return files[i].GetChecksum() < files[j].GetChecksum() })
		default:
			errs = multierror.Append(errs, fmt.Errorf("keyword not sortable: %s", orderBy[i]))
		}
	}
	if descending {
		return util.Reverse(files), errs.ErrorOrNil()
	}
	return files, errs.ErrorOrNil()
}

const (
	statControllerReady       string = "controller_ready"
	statSingleControllerReady string = "single_controller_ready"
	statMultiControllerReady  string = "multi_controller_ready"
	statApplianceReady        string = "appliance_ready"

	// https://github.com/appgate/sdp-api-specification/blob/94d8f7970cd025c8cf92b4560c1a9a0595d66133/dashboard.yml#L477-L483
	statusHealthy      string = "healthy"
	statusBusy         string = "busy"
	statusWarning      string = "warning"
	statusError        string = "error"
	statusNotAvailable string = "n/a"
	statusOffline      string = "offline"
)

var StatReady = []string{
	statControllerReady,
	statSingleControllerReady,
	statMultiControllerReady,
	statApplianceReady,
}

var StatusNotBusy = []string{
	statusHealthy,
	statusWarning,
	statusError,
	statusNotAvailable,
	statusOffline,
}

// StatsIsOnline will return true if the Controller reports the appliance to be online in a valid status
func StatsIsOnline(s openapi.StatsAppliancesListAllOfData) bool {
	// from appliance 6.0, 'online' field has been removed in favour for status
	// we will keep GetOnline() for backwards compatibility.
	if s.Online != nil && s.GetOnline() {
		return true
	}
	if util.InSlice(s.GetStatus(), []string{statusNotAvailable, statusOffline}) {
		return false
	}
	// unknown or empty status will report appliance as offline.
	return util.InSlice(s.GetStatus(), []string{statusHealthy, statusBusy, statusWarning, statusError})
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

func PrettyBytes(v float64) string {
	for _, unit := range []string{"", "K", "M", "G", "T", "P", "E", "Z"} {
		if math.Abs(float64(v)) < 1024.0 {
			return fmt.Sprintf("%.2f%sB", v, unit)
		}
		v /= 1024.0
	}
	return fmt.Sprintf("%.2fYB", v)
}

type ManifestConfig struct {
	MediaType string
	Size      int
	Digest    string
}
type DockerManifest struct {
	SchemaVersion int
	MediaType     string
	Config        ManifestConfig
	Layers        []ManifestConfig
}

type ImageJSON struct {
	Image string `json:"image"`
}

type fileEntry struct {
	r    io.Reader
	path string
	err  error
}

type imageBundleArgs struct {
	ctx           context.Context
	client        *http.Client
	fileEntryChan chan<- fileEntry
	wg            *sync.WaitGroup
	ciMode        bool
	registry      *url.URL
	token         *string
	image         string
	tag           string
	progress      *tui.Progress
}

func DownloadDockerBundles(ctx context.Context, p *tui.Progress, client *http.Client, path string, registry *url.URL, images map[string]string, ciMode bool) (*os.File, error) {
	// Create zip-archive
	dir := filepath.Dir(path)
	if ok, err := util.FileExists(dir); err == nil && !ok {
		os.MkdirAll(dir, os.ModePerm)
	} else if err != nil {
		return nil, err
	}
	archive, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	log.Info("downloading image layers for ", filepath.Base(path))
	fileEntryChan := make(chan fileEntry)
	var wg sync.WaitGroup
	wg.Add(len(images))
	var token *string
	if registry.Host == "public.ecr.aws" {
		token, err = getPublicECRToken(client, images)
		if err != nil {
			return nil, err
		}
	}
	for image, tag := range images {
		go func(image, tag string) {
			args := imageBundleArgs{
				ctx:           ctx,
				client:        client,
				ciMode:        ciMode,
				registry:      registry,
				token:         token,
				image:         image,
				tag:           tag,
				progress:      p,
				fileEntryChan: fileEntryChan,
				wg:            &wg,
			}
			downloadDockerImageBundle(args)
		}(image, tag)
	}

	go func(wg *sync.WaitGroup, ch chan fileEntry) {
		wg.Wait()
		close(ch)
	}(&wg, fileEntryChan)

	var errs *multierror.Error
	for v := range fileEntryChan {
		if v.err != nil {
			errs = multierror.Append(errs, v.err)
			continue
		}
		fb, err := zipWriter.Create(v.path)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		size, err := io.Copy(fb, v.r)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		log.WithField("path", v.path).WithField("size", size).Debug("wrote layer")
	}

	return archive, errs.ErrorOrNil()
}

func getPublicECRToken(client *http.Client, images map[string]string) (*string, error) {
	type ecrToken struct {
		Token string
	}
	params := []string{"service=public.ecr.aws"}
	for image := range images {
		params = append(params, fmt.Sprintf("scope=repsoitory:appgate-sdp/%s:pull", image))
	}
	res, err := client.Get(fmt.Sprintf("https://public.ecr.aws/token/?%s", strings.Join(params, "&")))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, err
	}
	var token ecrToken
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token.Token, nil
}

// Image bundle should contain the following artifacts:
// - Manifest (<tag>.json)
// - Config (<digest>.json)
// - Repository/Image/Tag (image.json)
// - Layers (<layer-digest>.tar.gz)
func downloadDockerImageBundle(args imageBundleArgs) {
	defer args.wg.Done()
	headers := map[string]string{
		"Accept":       "application/vnd.docker.distribution.manifest.v2+json",
		"Content-Type": "application/json",
	}
	if args.token != nil {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", *args.token)
	}

	var username, password string
	if u := os.Getenv("SDPCTL_DOCKER_REGISTRY_USERNAME"); len(u) > 0 {
		username = u
	}
	if p := os.Getenv("SDPCTL_DOCKER_REGISTRY_PASSWORD"); len(p) > 0 {
		password = p
	}
	// use basic auth if username and password is set
	if len(username) > 0 && len(password) > 0 {
		auth := username + ":" + password
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	}

	requestRetry := func(client *http.Client, req *http.Request) (*http.Response, error) {
		return backoff.RetryWithData(func() (*http.Response, error) {
			res, err := client.Do(req)
			if err != nil {
				return nil, backoff.Permanent(err)
			}
			if res.StatusCode != http.StatusOK {
				if res.StatusCode == http.StatusNotFound {
					return nil, backoff.Permanent(errors.New("image bundle not found"))
				}
				return nil, fmt.Errorf("Recieved %s status", res.Status)
			}
			return res, nil
		}, backoff.NewExponentialBackOff())
	}

	// Download the image manifest
	// See https://github.com/distribution/distribution/blob/main/docs/spec/manifest-v2-2.md#image-manifest-field-descriptions
	manifestURL := fmt.Sprintf("%s://%s%s/%s/manifests/%s", args.registry.Scheme, args.registry.Host, prependString(args.registry.Path, "/v2"), args.image, args.tag)
	manifestReq, err := http.NewRequestWithContext(args.ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		args.fileEntryChan <- fileEntry{err: err}
		return
	}
	for k, v := range headers {
		manifestReq.Header.Set(k, v)
	}
	manifestRes, err := requestRetry(args.client, manifestReq)
	if err != nil {
		args.fileEntryChan <- fileEntry{err: fmt.Errorf("failed to fetch image manifest: %w", err)}
		return
	}
	defer manifestRes.Body.Close()

	manifestBuf := &bytes.Buffer{}
	if _, err := io.Copy(manifestBuf, manifestRes.Body); err != nil {
		args.fileEntryChan <- fileEntry{err: err}
		return
	}
	JSONManifest := DockerManifest{}
	if err := json.Unmarshal(manifestBuf.Bytes(), &JSONManifest); err != nil {
		args.fileEntryChan <- fileEntry{err: err}
		return
	}
	// <tag>.json
	args.fileEntryChan <- fileEntry{
		path: fmt.Sprintf("%s/%s.json", args.image, args.tag),
		r:    manifestBuf,
	}

	// Download the container image configuration
	// See https://github.com/opencontainers/image-spec/blob/main/config.md
	ImageID := JSONManifest.Config.Digest
	configURL := fmt.Sprintf("%s://%s%s/%s/blobs/%s", args.registry.Scheme, args.registry.Host, prependString(args.registry.Path, "/v2"), args.image, ImageID)
	configReq, err := http.NewRequestWithContext(args.ctx, http.MethodGet, configURL, nil)
	if err != nil {
		args.fileEntryChan <- fileEntry{err: err}
		return
	}
	for k, v := range headers {
		configReq.Header.Set(k, v)
	}
	configRes, err := requestRetry(args.client, configReq)
	if err != nil {
		args.fileEntryChan <- fileEntry{err: fmt.Errorf("failed to fetch image config: %w", err)}
		return
	}
	defer configRes.Body.Close()

	configBuf := &bytes.Buffer{}
	if _, err := io.Copy(configBuf, configRes.Body); err != nil {
		args.fileEntryChan <- fileEntry{err: err}
		return
	}
	// <digest>.json
	args.fileEntryChan <- fileEntry{
		path: fmt.Sprintf("%s/%s.json", args.image, strings.Replace(ImageID, "sha256:", "", 1)),
		r:    configBuf,
	}

	// Create image.json
	// It provides repository/image/tag information for arc
	// Example: { "image": "docker-registry-url:5001/aitorbot:latest" }
	imageJSON := ImageJSON{
		Image: fmt.Sprintf("%s%s/%s:%s", args.registry.Host, args.registry.Path, args.image, args.tag),
	}
	ibytes, err := json.Marshal(imageJSON)
	if err != nil {
		args.fileEntryChan <- fileEntry{err: err}
		return
	}
	args.fileEntryChan <- fileEntry{
		path: fmt.Sprintf("%s/image.json", args.image),
		r:    bytes.NewReader(ibytes),
	}

	// Download .tar.gz file containing the layers of the image
	// Layers are stored in the blob portion of the registry, keyed by digest
	// See https://docs.docker.com/registry/spec/api/#pulling-a-layer
	g, ctx := errgroup.WithContext(args.ctx)
	for _, l := range JSONManifest.Layers {
		digest := l.Digest
		g.Go(func() error {
			layerHash := strings.Replace(digest, "sha256:", "", 1)
			f := layerHash + ".tar.gz"
			layerURL := fmt.Sprintf("%s://%s%s/%s/blobs/%s", args.registry.Scheme, args.registry.Host, prependString(args.registry.Path, "/v2"), args.image, digest)
			layerReq, err := http.NewRequestWithContext(ctx, http.MethodGet, layerURL, nil)
			if err != nil {
				return err
			}
			for k, v := range headers {
				layerReq.Header.Set(k, v)
			}
			layerLog := log.WithField("layer", layerHash)
			layerLog.Info("downloading image layer")
			layerRes, err := requestRetry(args.client, layerReq)
			if err != nil {
				return fmt.Errorf("failed to fetch image layer: %w", err)
			}
			defer layerRes.Body.Close()

			bodyReader := layerRes.Body
			if args.progress != nil {
				size := layerRes.ContentLength
				bodyReader = args.progress.FileDownloadProgress("downloading layer "+f[0:11], "downloaded", size, 25, bodyReader, mpb.BarRemoveOnComplete())
			}

			layerName := fmt.Sprintf("%s/%s", args.image, f)
			buf := &bytes.Buffer{}
			if _, err := io.Copy(buf, bodyReader); err != nil {
				return err
			}
			args.fileEntryChan <- fileEntry{
				path: layerName,
				r:    buf,
			}
			layerLog.Info("download finished")
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		args.fileEntryChan <- fileEntry{err: err}
	}
}

func prependString(s, pre string) string {
	if !strings.HasPrefix(s, pre) {
		return pre + s
	}
	return s
}
