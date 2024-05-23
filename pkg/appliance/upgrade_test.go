package appliance

import (
	"bytes"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestMakeUpgradePlan(t *testing.T) {
	hostname := "appgate.test"
	v62, _ := version.NewVersion("6.2")
	v63, _ := version.NewVersion("6.3")

	coll := generateCollective(hostname, v62, v63)
	primary := coll.appliances["primary"]

	type args struct {
		appliances   []openapi.Appliance
		stats        openapi.StatsAppliancesList
		ctrlHostname string
		filter       map[string]map[string]string
		orderBy      []string
		descending   bool
	}
	tests := []struct {
		name    string
		args    args
		want    *UpgradePlan
		wantErr bool
	}{
		{
			name: "grouping test",
			args: args{
				appliances: []openapi.Appliance{
					coll.appliances["primary"],
					coll.appliances["secondary"],
					coll.appliances["gatewayA1"],
					coll.appliances["gatewayA2"],
					coll.appliances["gatewayA3"],
					coll.appliances["gatewayB1"],
					coll.appliances["gatewayB2"],
					coll.appliances["gatewayC1"],
					coll.appliances["gatewayC2"],
					coll.appliances["logforwarderA1"],
					coll.appliances["logforwarderA2"],
					coll.appliances["portalA1"],
					coll.appliances["connectorA1"],
					coll.appliances["logserver"],
				},
				stats:        coll.stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primary,
				Controllers:       []openapi.Appliance{coll.appliances["secondary"]},
				Batches: [][]openapi.Appliance{
					{coll.appliances["gatewayA1"], coll.appliances["gatewayB1"], coll.appliances["gatewayC1"], coll.appliances["logforwarderA1"]},
					{coll.appliances["gatewayA2"], coll.appliances["gatewayB2"], coll.appliances["gatewayC2"], coll.appliances["logforwarderA2"]},
					{coll.appliances["connectorA1"], coll.appliances["gatewayA3"], coll.appliances["logserver"], coll.appliances["portalA1"]},
				},
				adminHostname: hostname,
				stats:         coll.stats,
			},
		},
		{
			name: "test grouping from unordered",
			args: args{
				appliances: []openapi.Appliance{
					coll.appliances["primary"],
					coll.appliances["gatewayA1"],
					coll.appliances["gatewayB2"],
					coll.appliances["gatewayA2"],
					coll.appliances["logserver"],
					coll.appliances["logforwarderA2"],
					coll.appliances["gatewayB1"],
					coll.appliances["connectorA1"],
					coll.appliances["gatewayC1"],
					coll.appliances["secondary"],
					coll.appliances["gatewayA3"],
					coll.appliances["gatewayC2"],
					coll.appliances["portalA1"],
					coll.appliances["logforwarderA1"],
				},
				stats:        coll.stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primary,
				Controllers:       []openapi.Appliance{coll.appliances["secondary"]},
				Batches: [][]openapi.Appliance{
					{coll.appliances["gatewayA1"], coll.appliances["gatewayB1"], coll.appliances["gatewayC1"], coll.appliances["logforwarderA1"]},
					{coll.appliances["gatewayA2"], coll.appliances["gatewayB2"], coll.appliances["gatewayC2"], coll.appliances["logforwarderA2"]},
					{coll.appliances["connectorA1"], coll.appliances["gatewayA3"], coll.appliances["logserver"], coll.appliances["portalA1"]},
				},
				adminHostname: hostname,
				stats:         coll.stats,
			},
		},
		{
			name: "test multi controller upgrade error",
			args: args{
				appliances: []openapi.Appliance{
					coll.appliances["primary"],
					coll.appliances["controller3"],
					coll.appliances["gatewayA1"],
					coll.appliances["gatewayB2"],
					coll.appliances["gatewayA2"],
					coll.appliances["logserver"],
					coll.appliances["logforwarderA2"],
					coll.appliances["gatewayB1"],
					coll.appliances["connectorA1"],
					coll.appliances["gatewayC1"],
					coll.appliances["secondary"],
					coll.appliances["gatewayA3"],
					coll.appliances["gatewayC2"],
					coll.appliances["portalA1"],
					coll.appliances["logforwarderA1"],
				},
				stats:        coll.stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			wantErr: true,
		},
		{
			name: "test offline controller",
			args: args{
				appliances: []openapi.Appliance{
					coll.appliances["primary"],
					coll.appliances["controller4"],
					coll.appliances["gatewayA1"],
					coll.appliances["gatewayB2"],
					coll.appliances["gatewayA2"],
					coll.appliances["logserver"],
					coll.appliances["logforwarderA2"],
					coll.appliances["gatewayB1"],
					coll.appliances["connectorA1"],
					coll.appliances["gatewayC1"],
					coll.appliances["secondary"],
					coll.appliances["gatewayA3"],
					coll.appliances["gatewayC2"],
					coll.appliances["portalA1"],
					coll.appliances["logforwarderA1"],
				},
				stats:        coll.stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUpgradePlan(tt.args.appliances, tt.args.stats, tt.args.ctrlHostname, tt.args.filter, tt.args.orderBy, tt.args.descending)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpgradePlan_PrintSummary(t *testing.T) {
	hostname := "appgate.test"
	v62, _ := version.NewVersion("6.2")
	v621, _ := version.NewVersion("6.2.1")
	v63, _ := version.NewVersion("6.3")

	type inData struct {
		Appliances []string
		from, to   *version.Version
		hostname   string
		filter     map[string]map[string]string
		orderBy    []string
		descending bool
		backup     []string
	}
	tests := []struct {
		name    string
		in      inData
		wantOut string
		wantErr bool
	}{
		{
			name: "test summary",
			in: inData{
				Appliances: []string{
					"primary",
					"secondary",
					"gatewayA1",
					"gatewayA2",
					"gatewayA3",
					"gatewayB1",
					"gatewayB2",
					"gatewayC1",
					"gatewayC2",
					"logforwarderA1",
					"logforwarderA2",
					"portalA1",
					"connectorA1",
					"logserver",
				},
				from:     v62,
				to:       v63,
				hostname: hostname,
				filter:   DefaultCommandFilter,
				backup:   []string{"primary", "secondary"},
			},
			wantOut: `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in steps:

 1. The primary Controller will be upgraded
    This will result in the API being unreachable while completing the primary Controller upgrade

    Appliance    Current version    Prepared version    Backup
    ---------    ---------------    ----------------    ------
    primary      6.2.0              6.3.0               ✓


 2. Additional Controllers will be upgraded in serial
    In some cases, the Controller function on additional Controllers will need to be disabled
    before proceeding with the upgrade. The disabled Controllers will then be re-enabled once
    the upgrade is completed
    This step will also reboot the upgraded Controllers for the upgrade to take effect

    Appliance    Current version    Prepared version    Backup
    ---------    ---------------    ----------------    ------
    secondary    6.2.0              6.3.0               ✓


 3. Additional appliances will be upgraded in parallell batches. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process
    Some of the additional appliances may need to be rebooted for the upgrade to take effect

    Batch #1:

    Appliance         Current version    Prepared version    Backup
    ---------         ---------------    ----------------    ------
    gatewayA1         6.2.0              6.3.0               ⨯
    gatewayB1         6.2.0              6.3.0               ⨯
    gatewayC1         6.2.0              6.3.0               ⨯
    logforwarderA1    6.2.0              6.3.0               ⨯

    Batch #2:

    Appliance         Current version    Prepared version    Backup
    ---------         ---------------    ----------------    ------
    gatewayA2         6.2.0              6.3.0               ⨯
    gatewayB2         6.2.0              6.3.0               ⨯
    gatewayC2         6.2.0              6.3.0               ⨯
    logforwarderA2    6.2.0              6.3.0               ⨯

    Batch #3:

    Appliance      Current version    Prepared version    Backup
    ---------      ---------------    ----------------    ------
    connectorA1    6.2.0              6.3.0               ⨯
    gatewayA3      6.2.0              6.3.0               ⨯
    logserver      6.2.0              6.3.0               ⨯
    portalA1       6.2.0              6.3.0               ⨯


`,
		},
		{
			name: "with skipped",
			in: inData{
				from:     v62,
				to:       v621,
				hostname: hostname,
				filter:   DefaultCommandFilter,
				Appliances: []string{
					"primary",
					"secondary",
					"controller5",
					"gatewayA1",
					"gatewayA2",
				},
				backup: []string{"primary"},
			},
			wantOut: `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in steps:

 1. The primary Controller will be upgraded
    This will result in the API being unreachable while completing the primary Controller upgrade

    Appliance    Current version    Prepared version    Backup
    ---------    ---------------    ----------------    ------
    primary      6.2.0              6.2.1               ✓


 2. Additional Controllers will be upgraded in serial
    In some cases, the Controller function on additional Controllers will need to be disabled
    before proceeding with the upgrade. The disabled Controllers will then be re-enabled once
    the upgrade is completed
    This step will also reboot the upgraded Controllers for the upgrade to take effect

    Appliance    Current version    Prepared version    Backup
    ---------    ---------------    ----------------    ------
    secondary    6.2.0              6.2.1               ⨯


 3. Additional appliances will be upgraded in parallell batches. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process
    Some of the additional appliances may need to be rebooted for the upgrade to take effect

    Batch #1:

    Appliance    Current version    Prepared version    Backup
    ---------    ---------------    ----------------    ------
    gatewayA1    6.2.0              6.2.1               ⨯

    Batch #2:

    Appliance    Current version    Prepared version    Backup
    ---------    ---------------    ----------------    ------
    gatewayA2    6.2.0              6.2.1               ⨯


Appliances that will be skipped:
  - controller5: appliance is not prepared for upgrade
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coll := generateCollective(tt.in.hostname, tt.in.from, tt.in.to)
			appliances := make([]openapi.Appliance, 0, len(tt.in.Appliances))
			for _, v := range tt.in.Appliances {
				appliances = append(appliances, coll.appliances[v])
			}
			up, err := NewUpgradePlan(appliances, coll.stats, tt.in.hostname, tt.in.filter, tt.in.orderBy, tt.in.descending)
			if err != nil {
				t.Fatalf("internal test error: %v", err)
			}
			if len(tt.in.backup) > 0 {
				ids := make([]string, 0, len(tt.in.backup))
				for _, name := range tt.in.backup {
					a := coll.appliances[name]
					ids = append(ids, a.GetId())
				}
				up.AddBackups(ids)
			}
			out := &bytes.Buffer{}
			if err := up.PrintSummary(out); (err != nil) != tt.wantErr {
				t.Errorf("UpgradePlan.PrintSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantOut, out.String())
		})
	}
}

type collectiveTestStruct struct {
	appliances map[string]openapi.Appliance
	stats      openapi.StatsAppliancesList
}

func generateCollective(hostname string, from, to *version.Version) collectiveTestStruct {
	stats := *openapi.NewStatsAppliancesListWithDefaults()
	appliances := map[string]openapi.Appliance{}

	siteA := uuid.NewString()
	siteB := uuid.NewString()
	siteC := uuid.NewString()

	primary, s := GenerateApplianceWithStats([]string{FunctionController}, "primary", hostname, from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count := stats.GetControllerCount()
	stats.SetControllerCount(count + 1)
	appliances[primary.GetName()] = primary

	secondary, s := GenerateApplianceWithStats([]string{FunctionController}, "secondary", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)
	appliances[secondary.GetName()] = secondary

	// not prepared controller
	controller3, s := GenerateApplianceWithStats([]string{FunctionController}, "controller3", "", from.String(), "", statusHealthy, UpgradeStatusIdle, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)
	appliances[controller3.GetName()] = controller3

	// offline controller
	controller4, s := GenerateApplianceWithStats([]string{FunctionController}, "controller4", "", from.String(), "", statusOffline, UpgradeStatusIdle, false, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)
	appliances[controller4.GetName()] = controller4

	// already same version
	controller5, s := GenerateApplianceWithStats([]string{FunctionController}, "controller5", "", to.String(), "", statusHealthy, UpgradeStatusIdle, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)
	appliances[controller5.GetName()] = controller5

	gatewayA1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayA1", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayA1.GetName()] = gatewayA1

	gatewayA2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayA2", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayA2.GetName()] = gatewayA2

	gatewayA3, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayA3", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayA3.GetName()] = gatewayA3

	gatewayB1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayB1", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayB1.GetName()] = gatewayB1

	gatewayB2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayB2", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayB2.GetName()] = gatewayB2

	gatewayC1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayC1", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayC1.GetName()] = gatewayC1

	gatewayC2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gatewayC2", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)
	appliances[gatewayC2.GetName()] = gatewayC2

	logforwarderA1, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarderA1", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)
	appliances[logforwarderA1.GetName()] = logforwarderA1

	logforwarderA2, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarderA2", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)
	appliances[logforwarderA2.GetName()] = logforwarderA2

	portalA1, s := GenerateApplianceWithStats([]string{FunctionPortal}, "portalA1", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetPortalCount()
	stats.SetPortalCount(count + 1)
	appliances[portalA1.GetName()] = portalA1

	connectorA1, s := GenerateApplianceWithStats([]string{FunctionConnector}, "connectorA1", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetConnectorCount()
	stats.SetConnectorCount(count + 1)
	appliances[connectorA1.GetName()] = connectorA1

	logServer, s := GenerateApplianceWithStats([]string{FunctionLogServer}, "logserver", "", from.String(), to.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogServerCount()
	stats.SetLogServerCount(count + 1)
	appliances[logServer.GetName()] = logServer

	return collectiveTestStruct{
		appliances: appliances,
		stats:      stats,
	}
}
