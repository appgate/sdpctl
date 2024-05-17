package appliance

import (
	"fmt"
	"sort"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
)

type SkipUpgrade struct {
	Reason    string
	Appliance openapi.Appliance
}

func (su SkipUpgrade) Error() string {
	return fmt.Sprintf("%s: %s", su.Appliance.GetName(), su.Reason)
}

const (
	SkipReasonNotPrepared            = "appliance is not prepared for upgrade"
	SkipReasonOffline                = "appliance is offline"
	SkipReasonFiltered               = "filtered using the '--include' and/or '--exclude' flag"
	SkipReasonAlreadyPrepared        = "appliance is already prepared for upgrade with a higher or equal version"
	SkipReasonUnsupportedUpgradePath = "Upgrading from version 6.0.0 to version 6.2.x is unsupported. Version 6.0.1 or later is required."
)

type UpgradePlan struct {
	PrimaryController       openapi.Appliance
	Controllers             []openapi.Appliance
	LogForwardersAndServers []openapi.Appliance
	Batches                 [][]openapi.Appliance
	Skipping                []SkipUpgrade
}

func NewUpgradePlan(appliances []openapi.Appliance, stats openapi.StatsAppliancesList, ctrlHostname string, filter map[string]map[string]string, orderBy []string, descending bool, targetVersion *version.Version) (*UpgradePlan, error) {
	plan := UpgradePlan{}

	primary, err := FindPrimaryController(appliances, ctrlHostname, false)
	if err != nil {
		return nil, err
	}

	postOnlineInclude, offline, err := FilterAvailable(appliances, stats.GetData())
	if err != nil {
		return nil, err
	}

	for _, o := range offline {
		plan.Skipping = append(plan.Skipping, SkipUpgrade{
			Appliance: o,
			Reason:    SkipReasonOffline,
		})
	}

	postFilterInclude, filtered, err := FilterAppliances(appliances, filter, orderBy, descending)
	if err != nil {
		return nil, err
	}

	for _, f := range filtered {
		plan.Skipping = append(plan.Skipping, SkipUpgrade{
			Appliance: f,
			Reason:    SkipReasonFiltered,
		})
	}

	finalApplianceList := postOnlineInclude
	for _, a := range postFilterInclude {
		finalApplianceList = AppendUniqueAppliance(finalApplianceList, a)
	}

	// Sort input group first
	sort.SliceStable(finalApplianceList, func(i, j int) bool {
		return finalApplianceList[i].GetName() < finalApplianceList[j].GetName()
	})

	gatewaysGroupedBySite := map[string][]openapi.Appliance{}
	other := []openapi.Appliance{}

	// LogForwarders and LogServers need to in their own group
	// when upgrading from <= 5.5 to >= 6.0
	lflsConstraint, _ := version.NewConstraint(">= 6.0.0-beta")

	var errs *multierror.Error
	for _, a := range finalApplianceList {
		stats, err := ApplianceStats(a, stats)
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.Skipping = append(plan.Skipping, SkipUpgrade{
				Appliance: a,
				Reason:    "failed to get appliance stats",
			})
			continue
		}
		currentVersion, err := ParseVersionString(stats.GetVersion())
		if err != nil {
			errs = multierror.Append(errs, err)
			plan.Skipping = append(plan.Skipping, SkipUpgrade{
				Appliance: a,
				Reason:    "failed to parse current appliance version",
			})
			continue
		}
		if ctrl, ok := a.GetControllerOk(); ok {
			if ctrl.GetEnabled() {
				if a.GetId() == primary.GetId() {
					plan.PrimaryController = a
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
		}
		other = append(other, a)
	}

	// Determine how many batches we will do
	batchSize := 0
	for _, g := range gatewaysGroupedBySite {
		if len(g) > batchSize {
			batchSize = len(g)
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
	sort.SliceStable(plan.LogForwardersAndServers, func(i, j int) bool {
		return plan.LogForwardersAndServers[i].GetName() < plan.LogForwardersAndServers[j].GetName()
	})
	for i := 0; i < len(plan.Batches); i++ {
		sort.SliceStable(plan.Batches[i], func(ix, jx int) bool {
			return plan.Batches[i][ix].GetName() < plan.Batches[i][jx].GetName()
		})
	}

	return &plan, nil
}

func ApplianceStats(a openapi.Appliance, stats openapi.StatsAppliancesList) (*openapi.StatsAppliancesListAllOfData, error) {
	for _, s := range stats.GetData() {
		if s.GetId() == a.GetId() {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("failed to find appliance stats")
}
