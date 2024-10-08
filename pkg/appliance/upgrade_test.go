package appliance

import (
	"bytes"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/stretchr/testify/assert"
)

func TestMakeUpgradePlan(t *testing.T) {
	hostname := "appgate.test"
	v62 := "6.2"
	v63 := "6.3"

	coll := GenerateCollective(t, hostname, v62, v63, PreSetApplianceNames)
	primary := coll.Appliances["primary"]
	primaryWithGateway := coll.Appliances["controller-gateway-primary"]

	type args struct {
		appliances   []openapi.Appliance
		stats        *openapi.StatsAppliancesList
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
					coll.Appliances["primary"],
					coll.Appliances["secondary"],
					coll.Appliances["gatewayA1"],
					coll.Appliances["gatewayA2"],
					coll.Appliances["gatewayA3"],
					coll.Appliances["gatewayB1"],
					coll.Appliances["gatewayB2"],
					coll.Appliances["gatewayC1"],
					coll.Appliances["gatewayC2"],
					coll.Appliances["logforwarderA1"],
					coll.Appliances["logforwarderA2"],
					coll.Appliances["portalA1"],
					coll.Appliances["connectorA1"],
					coll.Appliances["logserver"],
				},
				stats:        coll.Stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primary,
				Controllers:       []openapi.Appliance{coll.Appliances["secondary"]},
				Batches: [][]openapi.Appliance{
					{coll.Appliances["gatewayA1"], coll.Appliances["gatewayB1"], coll.Appliances["gatewayC1"], coll.Appliances["logforwarderA1"]},
					{coll.Appliances["gatewayA2"], coll.Appliances["gatewayB2"], coll.Appliances["gatewayC2"], coll.Appliances["logforwarderA2"]},
					{coll.Appliances["connectorA1"], coll.Appliances["gatewayA3"], coll.Appliances["logserver"], coll.Appliances["portalA1"]},
				},
			},
		},
		{
			name: "test grouping from unordered",
			args: args{
				appliances: []openapi.Appliance{
					coll.Appliances["primary"],
					coll.Appliances["gatewayA1"],
					coll.Appliances["gatewayB2"],
					coll.Appliances["gatewayA2"],
					coll.Appliances["logserver"],
					coll.Appliances["logforwarderA2"],
					coll.Appliances["gatewayB1"],
					coll.Appliances["connectorA1"],
					coll.Appliances["gatewayC1"],
					coll.Appliances["secondary"],
					coll.Appliances["gatewayA3"],
					coll.Appliances["gatewayC2"],
					coll.Appliances["portalA1"],
					coll.Appliances["logforwarderA1"],
				},
				stats:        coll.Stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primary,
				Controllers:       []openapi.Appliance{coll.Appliances["secondary"]},
				Batches: [][]openapi.Appliance{
					{coll.Appliances["gatewayA1"], coll.Appliances["gatewayB1"], coll.Appliances["gatewayC1"], coll.Appliances["logforwarderA1"]},
					{coll.Appliances["gatewayA2"], coll.Appliances["gatewayB2"], coll.Appliances["gatewayC2"], coll.Appliances["logforwarderA2"]},
					{coll.Appliances["connectorA1"], coll.Appliances["gatewayA3"], coll.Appliances["logserver"], coll.Appliances["portalA1"]},
				},
			},
		},
		{
			name: "test grouping with no other batches",
			args: args{
				appliances: []openapi.Appliance{
					coll.Appliances["controller-gateway-primary"],
					coll.Appliances["controller-gatewayB1"],
					coll.Appliances["logserver"],
				},
				stats:        coll.Stats,
				ctrlHostname: hostname,
				filter:       DefaultCommandFilter,
				orderBy:      nil,
				descending:   false,
			},
			want: &UpgradePlan{
				PrimaryController: &primaryWithGateway,
				Controllers:       []openapi.Appliance{coll.Appliances["controller-gatewayB1"]},
				Batches:           [][]openapi.Appliance{{coll.Appliances["logserver"]}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgradeStatusMap := map[string]UpgradeStatusResult{}
			for _, a := range tt.args.appliances {
				for _, s := range tt.args.stats.GetData() {
					if a.GetId() != s.GetId() {
						continue
					}
					us := s.GetUpgrade()
					upgradeStatusMap[a.GetId()] = UpgradeStatusResult{
						Name:    a.GetName(),
						Status:  us.GetStatus(),
						Details: us.GetDetails(),
					}
				}
			}
			got, err := NewUpgradePlan(tt.args.appliances, tt.args.stats, upgradeStatusMap, tt.args.ctrlHostname, tt.args.filter, tt.args.orderBy, tt.args.descending)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func TestUpgradePlan_PrintPreCompleteSummary(t *testing.T) {
	hostname := "appgate.test"
	v62 := "6.2"
	v621 := "6.2.1"
	v63 := "6.3"

	type inData struct {
		Appliances         []string
		from, to, hostname string
		filter             map[string]map[string]string
		orderBy            []string
		descending         bool
		backup             []string
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
					TestAppliancePrimary,
					TestApplianceSecondary,
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayC1,
					TestApplianceGatewayC2,
					TestApplianceLogForwarderA1,
					TestApplianceLogForwarderA2,
					TestAppliancePortalA1,
					TestApplianceConnectorA1,
					TestApplianceLogServer,
				},
				from:     v62,
				to:       v63,
				hostname: hostname,
				filter:   DefaultCommandFilter,
				backup:   []string{TestAppliancePrimary, TestApplianceSecondary},
			},
			wantOut: `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in steps:

 1. The primary Controller will be upgraded
    This will result in the API being unreachable while completing the primary Controller upgrade

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    primary      SiteA    6.2.0              6.3.0               ✓


 2. Additional Controllers will be upgraded in series.
    Additional Controllers will be put into maintenance mode before being upgraded. Maintenance
    mode will then be disabled once the upgrade has completed on the controller.
    This step will also reboot the upgraded Controllers for the upgrade to take effect.

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    secondary    SiteA    6.2.0              6.3.0               ✓


 3. Additional appliances will be upgraded in parallel batches. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process
    Some of the additional appliances may need to be rebooted for the upgrade to take effect

    Batch #1:

    Appliance         Site     Current version    Prepared version    Backup
    ---------         ----     ---------------    ----------------    ------
    gatewayA1         SiteA    6.2.0              6.3.0               ⨯
    gatewayB1         SiteB    6.2.0              6.3.0               ⨯
    gatewayC1         SiteC    6.2.0              6.3.0               ⨯
    logforwarderA1    SiteA    6.2.0              6.3.0               ⨯

    Batch #2:

    Appliance         Site     Current version    Prepared version    Backup
    ---------         ----     ---------------    ----------------    ------
    gatewayA2         SiteA    6.2.0              6.3.0               ⨯
    gatewayB2         SiteB    6.2.0              6.3.0               ⨯
    gatewayC2         SiteC    6.2.0              6.3.0               ⨯
    logforwarderA2    SiteA    6.2.0              6.3.0               ⨯

    Batch #3:

    Appliance      Site     Current version    Prepared version    Backup
    ---------      ----     ---------------    ----------------    ------
    connectorA1    SiteA    6.2.0              6.3.0               ⨯
    gatewayA3      SiteA    6.2.0              6.3.0               ⨯
    logserver      SiteA    6.2.0              6.3.0               ⨯
    portalA1       SiteA    6.2.0              6.3.0               ⨯


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
					TestAppliancePrimary,
					TestApplianceSecondary,
					TestApplianceControllerNotPrepared,
					TestApplianceController2NotPrepared,
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
				},
				backup: []string{"primary"},
			},
			wantOut: `
UPGRADE COMPLETE SUMMARY

Upgrade will be completed in steps:

 1. The primary Controller will be upgraded
    This will result in the API being unreachable while completing the primary Controller upgrade

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    primary      SiteA    6.2.0              6.2.1               ✓


 2. Additional Controllers will be upgraded in series.
    Additional Controllers will be put into maintenance mode before being upgraded. Maintenance
    mode will then be disabled once the upgrade has completed on the controller.
    This step will also reboot the upgraded Controllers for the upgrade to take effect.

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    secondary    SiteA    6.2.0              6.2.1               ⨯


 3. Additional appliances will be upgraded in parallel batches. The additional appliances will be split into
    batches to keep the Collective as available as possible during the upgrade process
    Some of the additional appliances may need to be rebooted for the upgrade to take effect

    Batch #1:

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    gatewayA1    SiteA    6.2.0              6.2.1               ⨯

    Batch #2:

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    gatewayA2    SiteA    6.2.0              6.2.1               ⨯


Appliances that will be skipped:
  - controller5: appliance is not prepared for upgrade
  - controller7: appliance is not prepared for upgrade
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coll := GenerateCollective(t, tt.in.hostname, tt.in.from, tt.in.to, tt.in.Appliances)
			appliances := make([]openapi.Appliance, 0, len(tt.in.Appliances))
			upgradeStatusMap := map[string]UpgradeStatusResult{}
			for _, v := range tt.in.Appliances {
				appliance, ok := coll.Appliances[v]
				if !ok {
					t.Fatalf("internal testing error: appliance name does not match any appliance")
					return
				}
				for _, stat := range coll.Stats.GetData() {
					if stat.GetId() != appliance.GetId() {
						continue
					}
					us := stat.GetUpgrade()
					upgradeStatusMap[appliance.GetId()] = UpgradeStatusResult{
						Name:    appliance.GetName(),
						Status:  us.GetStatus(),
						Details: us.GetDetails(),
					}
				}
				appliances = append(appliances, appliance)
			}
			up, err := NewUpgradePlan(appliances, coll.Stats, upgradeStatusMap, tt.in.hostname, tt.in.filter, tt.in.orderBy, tt.in.descending)
			if err != nil {
				t.Fatalf("internal test error: %v", err)
			}
			if len(tt.in.backup) > 0 {
				ids := make([]string, 0, len(tt.in.backup))
				for _, name := range tt.in.backup {
					a := coll.Appliances[name]
					ids = append(ids, a.GetId())
				}
				up.AddBackups(ids)
			}
			out := &bytes.Buffer{}
			if err := up.PrintPreCompleteSummary(out); (err != nil) != tt.wantErr {
				t.Errorf("UpgradePlan.PrintSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantOut, out.String())
		})
	}
}

func TestUpgradePlan_PrintPostCompleteSummary(t *testing.T) {
	v62 := "6.2"
	v621 := "6.2.1"

	testCases := []struct {
		name       string
		appliances []string
		expect     string
		from, to   string
	}{
		{
			name: "print no diff summary",
			appliances: []string{
				TestAppliancePrimary,
				TestApplianceGatewayA1,
			},
			from: v62,
			to:   v621,
			expect: `UPGRADE COMPLETE

Appliance    Current Version
---------    ---------------
gatewayA1    6.2.1
primary      6.2.1

`,
		},
		{
			name: "diff on three appliances",
			appliances: []string{
				TestAppliancePrimary,
				TestApplianceControllerNotPrepared,
				TestApplianceGatewayA1,
			},
			from: v62,
			to:   v621,
			expect: `UPGRADE COMPLETE

Appliance      Current Version
---------      ---------------
controller5    6.2.0
gatewayA1      6.2.1
primary        6.2.1

WARNING: Upgrade was completed, but not all appliances are running the same version.
`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			hostname := "appgate.test"
			coll := GenerateCollective(t, hostname, tt.from, tt.to, tt.appliances)
			appliances := make([]openapi.Appliance, 0, len(tt.appliances))
			upgradeStatusMap := map[string]UpgradeStatusResult{}
			for _, v := range tt.appliances {
				a := coll.Appliances[v]
				for _, s := range coll.Stats.GetData() {
					if s.GetId() != a.GetId() {
						continue
					}
					us := s.GetUpgrade()
					upgradeStatusMap[a.GetId()] = UpgradeStatusResult{
						Name:    a.GetName(),
						Status:  us.GetStatus(),
						Details: us.GetDetails(),
					}
				}
				appliances = append(appliances, a)
			}
			up, err := NewUpgradePlan(appliances, coll.Stats, upgradeStatusMap, hostname, DefaultCommandFilter, nil, false)
			if err != nil {
				t.Fatalf("PrintPostCompleteSummary() - internal test error: %v", err)
				return
			}
			buf := &bytes.Buffer{}
			err = up.PrintPostCompleteSummary(buf, coll.UpgradedStats.GetData())
			if err != nil {
				t.Fatal("error printing summary")
			}
			if !assert.Contains(t, buf.String(), tt.expect) {
				assert.Equal(t, tt.expect, buf.String())
			}
		})
	}
}
