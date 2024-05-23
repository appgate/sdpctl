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
	siteA := uuid.NewString()
	siteB := uuid.NewString()
	siteC := uuid.NewString()
	hostname := "appgate.test"
	v62, _ := version.NewVersion("6.2")
	v63, _ := version.NewVersion("6.3")

	stats := *openapi.NewStatsAppliancesListWithDefaults()
	primary, s := GenerateApplianceWithStats([]string{FunctionController}, "primary-controller", hostname, v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count := stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	secondary, s := GenerateApplianceWithStats([]string{FunctionController}, "secondary-controller", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	// not prepared controller
	controller3, s := GenerateApplianceWithStats([]string{FunctionController}, "controller-3", "", v62.String(), "", statusHealthy, UpgradeStatusIdle, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	// offline controller
	controller4, s := GenerateApplianceWithStats([]string{FunctionController}, "controller-4", "", v62.String(), "", statusOffline, UpgradeStatusIdle, false, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	gatewayA1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayA2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayA3, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A3", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayB1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-B1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayB2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-B2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayC1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-C1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayC2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-C2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	logforwarderA1, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarder-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)

	logforwarderA2, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarder-A2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)

	portalA1, s := GenerateApplianceWithStats([]string{FunctionPortal}, "portal-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetPortalCount()
	stats.SetPortalCount(count + 1)

	connectorA1, s := GenerateApplianceWithStats([]string{FunctionConnector}, "connector-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetConnectorCount()
	stats.SetConnectorCount(count + 1)

	logServer, s := GenerateApplianceWithStats([]string{FunctionLogServer}, "logserver", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogServerCount()
	stats.SetLogServerCount(count + 1)

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
					primary,
					secondary,
					gatewayA1,
					gatewayA2,
					gatewayA3,
					gatewayB1,
					gatewayB2,
					gatewayC1,
					gatewayC2,
					logforwarderA1,
					logforwarderA2,
					portalA1,
					connectorA1,
					logServer,
				},
				stats:        stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primary,
				Controllers:       []openapi.Appliance{secondary},
				Batches: [][]openapi.Appliance{
					{gatewayA1, gatewayB1, gatewayC1, logforwarderA1},
					{gatewayA2, gatewayB2, gatewayC2, logforwarderA2},
					{connectorA1, gatewayA3, logServer, portalA1},
				},
				adminHostname: hostname,
				stats:         stats,
			},
		},
		{
			name: "test grouping from unordered",
			args: args{
				appliances: []openapi.Appliance{
					primary,
					gatewayA1,
					gatewayB2,
					gatewayA2,
					logServer,
					logforwarderA2,
					gatewayB1,
					connectorA1,
					gatewayC1,
					secondary,
					gatewayA3,
					gatewayC2,
					portalA1,
					logforwarderA1,
				},
				stats:        stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primary,
				Controllers:       []openapi.Appliance{secondary},
				Batches: [][]openapi.Appliance{
					{gatewayA1, gatewayB1, gatewayC1, logforwarderA1},
					{gatewayA2, gatewayB2, gatewayC2, logforwarderA2},
					{connectorA1, gatewayA3, logServer, portalA1},
				},
				adminHostname: hostname,
				stats:         stats,
			},
		},
		{
			name: "test multi controller upgrade error",
			args: args{
				appliances: []openapi.Appliance{
					primary,
					controller3,
					gatewayA1,
					gatewayB2,
					gatewayA2,
					logServer,
					logforwarderA2,
					gatewayB1,
					connectorA1,
					gatewayC1,
					secondary,
					gatewayA3,
					gatewayC2,
					portalA1,
					logforwarderA1,
				},
				stats:        stats,
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
					primary,
					controller4,
					gatewayA1,
					gatewayB2,
					gatewayA2,
					logServer,
					logforwarderA2,
					gatewayB1,
					connectorA1,
					gatewayC1,
					secondary,
					gatewayA3,
					gatewayC2,
					portalA1,
					logforwarderA1,
				},
				stats:        stats,
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
	siteA := uuid.NewString()
	siteB := uuid.NewString()
	siteC := uuid.NewString()
	hostname := "appgate.test"
	v62, _ := version.NewVersion("6.2")
	v63, _ := version.NewVersion("6.3")

	stats := *openapi.NewStatsAppliancesListWithDefaults()
	primary, s := GenerateApplianceWithStats([]string{FunctionController}, "primary-controller", hostname, v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count := stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	secondary, s := GenerateApplianceWithStats([]string{FunctionController}, "secondary-controller", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	// // not prepared controller
	// controller3, s := GenerateApplianceWithStats([]string{FunctionController}, "controller-3", "", v62.String(), "", statusHealthy, UpgradeStatusIdle, true, siteA)
	// stats.Data = append(stats.Data, s)
	// count = stats.GetControllerCount()
	// stats.SetControllerCount(count + 1)

	// // offline controller
	// controller4, s := GenerateApplianceWithStats([]string{FunctionController}, "controller-4", "", v62.String(), "", statusOffline, UpgradeStatusIdle, false, siteA)
	// stats.Data = append(stats.Data, s)
	// count = stats.GetControllerCount()
	// stats.SetControllerCount(count + 1)

	gatewayA1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayA2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayA3, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A3", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayB1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-B1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayB2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-B2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayC1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-C1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayC2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-C2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	logforwarderA1, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarder-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)

	logforwarderA2, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarder-A2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)

	portalA1, s := GenerateApplianceWithStats([]string{FunctionPortal}, "portal-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetPortalCount()
	stats.SetPortalCount(count + 1)

	connectorA1, s := GenerateApplianceWithStats([]string{FunctionConnector}, "connector-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetConnectorCount()
	stats.SetConnectorCount(count + 1)

	logServer, s := GenerateApplianceWithStats([]string{FunctionLogServer}, "logserver", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, true, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogServerCount()
	stats.SetLogServerCount(count + 1)

	type inData struct {
		Appliances []openapi.Appliance
		Stats      openapi.StatsAppliancesList
		hostname   string
		filter     map[string]map[string]string
		orderBy    []string
		descending bool
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
				Stats: stats,
				Appliances: []openapi.Appliance{
					primary,
					secondary,
					gatewayA1,
					gatewayA2,
					gatewayA3,
					gatewayB1,
					gatewayB2,
					gatewayC1,
					gatewayC2,
					logforwarderA1,
					logforwarderA2,
					portalA1,
					connectorA1,
					logServer,
				},
				hostname: "appgate.test",
				filter:   DefaultCommandFilter,
			},
			wantOut: `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in steps:

 1. The primary Controller will be upgraded
    This will result in the API being unreachable while completing the primary Controller upgrade

    Appliance             Current version    Prepared version
    ---------             ---------------    ----------------
    primary-controller    6.2.0              6.3.0


 2. Additional Controllers will be upgraded in serial
    In some cases, the Controller function on additional Controllers will need to be disabled
    before proceeding with the upgrade. The disabled Controllers will then be re-enabled once
    the upgrade is completed
    This step will also reboot the upgraded Controllers for the upgrade to take effect

    Appliance               Current version    Prepared version
    ---------               ---------------    ----------------
    secondary-controller    6.2.0              6.3.0


 3. Additional appliances will be upgraded in parallell batches. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process
    Some of the additional appliances may need to be rebooted for the upgrade to take effect

    Batch #1:

    Appliance          Current version    Prepared version
    ---------          ---------------    ----------------
    gateway-A1         6.2.0              6.3.0
    gateway-B1         6.2.0              6.3.0
    gateway-C1         6.2.0              6.3.0
    logforwarder-A1    6.2.0              6.3.0

    Batch #2:

    Appliance          Current version    Prepared version
    ---------          ---------------    ----------------
    gateway-A2         6.2.0              6.3.0
    gateway-B2         6.2.0              6.3.0
    gateway-C2         6.2.0              6.3.0
    logforwarder-A2    6.2.0              6.3.0

    Batch #3:

    Appliance       Current version    Prepared version
    ---------       ---------------    ----------------
    connector-A1    6.2.0              6.3.0
    gateway-A3      6.2.0              6.3.0
    logserver       6.2.0              6.3.0
    portal-A1       6.2.0              6.3.0


`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up, err := NewUpgradePlan(tt.in.Appliances, tt.in.Stats, tt.in.hostname, tt.in.filter, tt.in.orderBy, tt.in.descending)
			if err != nil {
				t.Fatalf("internal test error: %v", err)
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
