package appliance

import (
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
	primary, s := GenerateApplianceWithStats([]string{FunctionController}, "primary-controller", hostname, v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count := stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	secondary, s := GenerateApplianceWithStats([]string{FunctionController}, "secondary-controller", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetControllerCount()
	stats.SetControllerCount(count + 1)

	gatewayA1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayA2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayA3, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-A3", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayB1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-B1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayB2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-B2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteB)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayC1, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-C1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	gatewayC2, s := GenerateApplianceWithStats([]string{FunctionGateway}, "gateway-C2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteC)
	stats.Data = append(stats.Data, s)
	count = stats.GetGatewayCount()
	stats.SetGatewayCount(count + 1)

	logforwarderA1, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarder-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)

	logforwarderA2, s := GenerateApplianceWithStats([]string{FunctionLogForwarder}, "logforwarder-A2", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogForwarderCount()
	stats.SetLogForwarderCount(count + 1)

	portalA1, s := GenerateApplianceWithStats([]string{FunctionPortal}, "portal-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetPortalCount()
	stats.SetPortalCount(count + 1)

	connectorA1, s := GenerateApplianceWithStats([]string{FunctionConnector}, "connector-A1", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetConnectorCount()
	stats.SetConnectorCount(count + 1)

	logServer, s := GenerateApplianceWithStats([]string{FunctionLogServer}, "logserver", "", v62.String(), v63.String(), statusHealthy, UpgradeStatusReady, siteA)
	stats.Data = append(stats.Data, s)
	count = stats.GetLogServerCount()
	stats.SetLogServerCount(count + 1)

	type args struct {
		appliances    []openapi.Appliance
		stats         openapi.StatsAppliancesList
		ctrlHostname  string
		filter        map[string]map[string]string
		orderBy       []string
		descending    bool
		targetVersion *version.Version
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
				stats:         stats,
				ctrlHostname:  hostname,
				filter:        DefaultCommandFilter,
				orderBy:       nil,
				descending:    false,
				targetVersion: v63,
			},
			want: &UpgradePlan{
				PrimaryController: primary,
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
				stats:         stats,
				ctrlHostname:  hostname,
				filter:        DefaultCommandFilter,
				orderBy:       nil,
				descending:    false,
				targetVersion: v63,
			},
			want: &UpgradePlan{
				PrimaryController: primary,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUpgradePlan(tt.args.appliances, tt.args.stats, tt.args.ctrlHostname, tt.args.filter, tt.args.orderBy, tt.args.descending)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
