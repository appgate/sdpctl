package appliance

import (
	"bytes"
	"strings"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/stretchr/testify/assert"
)

type testUpgradePlan struct {
	PrimaryController string
	Controllers       []string
	Batches           [][]string
	input             []string
}

func (tup *testUpgradePlan) allApplianceNames() []string {
	names := []string{}
	if tup.PrimaryController != "" {
		names = append(names, tup.PrimaryController)
	}
	names = append(names, tup.Controllers...)
	names = append(names, tup.input...)
	return names
}

func (tup *testUpgradePlan) getUpgradePlan(t *testing.T, collective *CollectiveTestStruct) *UpgradePlan {
	t.Helper()
	ug := &UpgradePlan{
		PrimaryController: collective.GetAppliance(tup.PrimaryController),
		Controllers:       []openapi.Appliance{},
		Batches:           make([][]openapi.Appliance, len(tup.Batches)),
	}

	for _, v := range tup.Controllers {
		ug.Controllers = append(ug.Controllers, *collective.GetAppliance(v))
	}
	for i, batch := range tup.Batches {
		ug.Batches[i] = make([]openapi.Appliance, 0, len(batch))
		for _, name := range batch {
			a := collective.GetAppliance(name)
			ug.Batches[i] = append(ug.Batches[i], *a)
		}
	}

	return ug
}

func TestMakeUpgradePlan(t *testing.T) {

	type args struct {
		maxUnavailable int
		filter         map[string]map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    testUpgradePlan
		wantErr bool
	}{
		{
			name: "grouping test",
			args: args{
				maxUnavailable: 1,
			},
			want: testUpgradePlan{
				PrimaryController: TestAppliancePrimary,
				Controllers:       []string{TestApplianceSecondary},
				Batches: [][]string{
					{"gatewayA1", "gatewayB1", "gatewayC2", "logforwarderA1"},
					{"connectorA1", "gatewayA2", "gatewayB2", "logforwarderA2"},
					{"gatewayA3", "gatewayC1", "logserver", "portalA1"},
				},
				input: []string{"gatewayA1",
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
			},
		},
		{
			name: "grouping with max unavailable 2",
			args: args{
				maxUnavailable: 2,
			},
			want: testUpgradePlan{
				PrimaryController: TestAppliancePrimary,
				Controllers:       []string{TestApplianceSecondary},
				Batches: [][]string{
					{"gatewayA1", "gatewayA2", "gatewayB1", "gatewayB2", "logforwarderA1", "logforwarderA2"},
					{"connectorA1", "gatewayA3", "gatewayC1", "gatewayC2", "logserver", "portalA1"},
				},
				input: []string{"gatewayA1",
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
			},
		},
		{
			name: "test grouping from unordered",
			args: args{
				maxUnavailable: 1,
			},
			want: testUpgradePlan{
				PrimaryController: TestAppliancePrimary,
				Controllers:       []string{TestApplianceSecondary},
				Batches: [][]string{
					{"gatewayA1", "gatewayB1", "gatewayC2", "logforwarderA1"},
					{"connectorA1", "gatewayA2", "gatewayB2", "logforwarderA2"},
					{"gatewayA3", "gatewayC1", "logserver", "portalA1"},
				},
				input: []string{
					"gatewayA1",
					"gatewayB2",
					"gatewayA2",
					"logserver",
					"logforwarderA2",
					"gatewayB1",
					"connectorA1",
					"gatewayC1",
					"secondary",
					"gatewayA3",
					"gatewayC2",
					"portalA1",
					"logforwarderA1",
				},
			},
		},
		{
			name: "test batch creation with high max unavailable",
			args: args{
				maxUnavailable: 9,
			},
			want: testUpgradePlan{
				PrimaryController: TestAppliancePrimary,
				Controllers:       []string{TestApplianceSecondary},
				Batches: [][]string{
					{"gatewayA1", "gatewayA10", "gatewayA11", "gatewayA12", "gatewayA2", "gatewayA3", "gatewayA4", "gatewayA5", "gatewayA6", "gatewayB1", "gatewayB2", "logforwarderA1", "logforwarderA2"},
					{"connectorA1", "gatewayA7", "gatewayA8", "gatewayA9", "gatewayC1", "gatewayC2", "logserver", "portalA1"},
				},
				input: []string{
					"gatewayA1",
					"gatewayA2",
					"gatewayA3",
					"gatewayA4",
					"gatewayA5",
					"gatewayA6",
					"gatewayA7",
					"gatewayA8",
					"gatewayA9",
					"gatewayA10",
					"gatewayA11",
					"gatewayA12",
					"gatewayB1",
					"gatewayB2",
					"logserver",
					"logforwarderA1",
					"logforwarderA2",
					"connectorA1",
					"gatewayC1",
					"secondary",
					"gatewayC2",
					"portalA1",
				},
			},
		},
		{
			name: "test grouping with no other batches",
			args: args{
				maxUnavailable: 1,
			},
			want: testUpgradePlan{
				PrimaryController: TestApplianceControllerGatewayPrimary,
				Controllers:       []string{"controller-gatewayB1"},
				Batches:           [][]string{{"logserver"}},
				input: []string{
					"logserver",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostname := "appgate.test"
			coll := GenerateCollective(t, hostname, "6.3.5", "6.4.1", tt.want.allApplianceNames())
			got, err := NewUpgradePlan(coll.GetAppliances(), coll.Stats, coll.GetUpgradeStatusMap(), hostname, tt.args.filter, nil, false, tt.args.maxUnavailable)
			if tt.wantErr {
				assert.Error(t, err)
			}
			want := tt.want.getUpgradePlan(t, coll)
			assert.EqualExportedValues(t, want, got)
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
    gatewayC2         SiteC    6.2.0              6.3.0               ⨯
    logforwarderA1    SiteA    6.2.0              6.3.0               ⨯

    Batch #2:

    Appliance         Site     Current version    Prepared version    Backup
    ---------         ----     ---------------    ----------------    ------
    connectorA1       SiteA    6.2.0              6.3.0               ⨯
    gatewayA2         SiteA    6.2.0              6.3.0               ⨯
    gatewayB2         SiteB    6.2.0              6.3.0               ⨯
    logforwarderA2    SiteA    6.2.0              6.3.0               ⨯

    Batch #3:

    Appliance    Site     Current version    Prepared version    Backup
    ---------    ----     ---------------    ----------------    ------
    gatewayA3    SiteA    6.2.0              6.3.0               ⨯
    gatewayC1    SiteC    6.2.0              6.3.0               ⨯
    logserver    SiteA    6.2.0              6.3.0               ⨯
    portalA1     SiteA    6.2.0              6.3.0               ⨯


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
					us := stat.GetDetails().Upgrade
					upgradeStatusMap[appliance.GetId()] = UpgradeStatusResult{
						Name:    appliance.GetName(),
						Status:  us.GetStatus(),
						Details: us.GetDetails(),
					}
				}
				appliances = append(appliances, appliance)
			}
			up, err := NewUpgradePlan(appliances, coll.Stats, upgradeStatusMap, tt.in.hostname, tt.in.filter, tt.in.orderBy, tt.in.descending, 1)
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
					us := s.GetDetails().Upgrade
					upgradeStatusMap[a.GetId()] = UpgradeStatusResult{
						Name:    a.GetName(),
						Status:  us.GetStatus(),
						Details: us.GetDetails(),
					}
				}
				appliances = append(appliances, a)
			}
			up, err := NewUpgradePlan(appliances, coll.Stats, upgradeStatusMap, hostname, DefaultCommandFilter, nil, false, 1)
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

func Test_calculateBatches(t *testing.T) {
	type args struct {
		appliances     []string
		maxUnavailable int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "basic test",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayB1,
				},
			},
			want: 1,
		},
		{
			name: "two batches",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
				},
			},
			want: 2,
		},
		{
			name: "only other",
			args: args{
				appliances: []string{
					TestAppliancePortalA1,
				},
			},
			want: 1,
		},
		{
			name: "three sites",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
					TestApplianceGatewayC1,
					TestApplianceGatewayC2,
				},
			},
			want: 3,
		},
		{
			name: "with max unavailable default",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
					TestApplianceGatewayC1,
					TestApplianceGatewayC2,
				},
				maxUnavailable: 1,
			},
			want: 3,
		},
		{
			name: "with max unavailable 3",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
					TestApplianceGatewayC1,
					TestApplianceGatewayC2,
				},
				maxUnavailable: 3,
			},
			want: 1,
		},
		{
			name: "with max unavailable 2",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
					TestApplianceGatewayC1,
					TestApplianceGatewayC2,
				},
				maxUnavailable: 2,
			},
			want: 2,
		},
		{
			name: "with max unavailable 2 on two sites",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
				},
				maxUnavailable: 2,
			},
			want: 2,
		},
		{
			name: "with max unavailable 2 on three sites",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayC1,
					TestApplianceGatewayC2,
				},
				maxUnavailable: 2,
			},
			want: 1,
		},
		{
			name: "with max unavailable 2, three appliances per site",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
				},
				maxUnavailable: 2,
			},
			want: 2,
		},
		{
			name: "12 appliances with max unavailable 9",
			args: args{
				appliances: []string{
					TestApplianceGatewayA1,
					TestApplianceGatewayA2,
					TestApplianceGatewayA3,
					TestApplianceGatewayA4,
					TestApplianceGatewayA5,
					TestApplianceGatewayA6,
					TestApplianceGatewayA7,
					TestApplianceGatewayA8,
					TestApplianceGatewayA9,
					TestApplianceGatewayA10,
					TestApplianceGatewayA11,
					TestApplianceGatewayA12,
					TestApplianceGatewayB1,
					TestApplianceGatewayB2,
					TestApplianceGatewayB3,
				},
				maxUnavailable: 9,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coll := GenerateCollective(t, "appgate.test", "6.3.5", "6.4.1", tt.args.appliances)

			gatewaysBySite := map[string][]openapi.Appliance{}
			logForwardersBySite := map[string][]openapi.Appliance{}
			other := []openapi.Appliance{}
			for k, a := range coll.Appliances {
				if strings.Contains(k, "gateway") && (!strings.Contains(k, "controller") || !strings.Contains(k, "primary")) {
					apps, ok := gatewaysBySite[a.GetSiteName()]
					if !ok {
						apps = []openapi.Appliance{}
					}
					gatewaysBySite[a.GetSiteName()] = append(apps, a)
					continue
				}
				if strings.Contains(k, "logforwarder") {
					apps, ok := gatewaysBySite[a.GetSiteName()]
					if !ok {
						apps = []openapi.Appliance{}
					}
					logForwardersBySite[a.GetSiteName()] = append(apps, a)
					continue
				}
				other = append(other, a)
			}

			if got := calculateBatches(gatewaysBySite, logForwardersBySite, other, tt.args.maxUnavailable); got != tt.want {
				t.Errorf("calculateBatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_divideByMaxUnavailable(t *testing.T) {
	type args struct {
		appliances     map[string][]string
		maxUnavailable int
	}
	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "basic test",
			args: args{
				appliances: map[string][]string{
					"SiteA": {TestApplianceGatewayA1},
				},
				maxUnavailable: 1,
			},
			want: [][]string{
				{TestApplianceGatewayA1},
			},
		},
		{
			name: "three groups",
			args: args{
				appliances: map[string][]string{
					"SiteA": {TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3},
				},
				maxUnavailable: 1,
			},
			want: [][]string{
				{TestApplianceGatewayA1},
				{TestApplianceGatewayA2},
				{TestApplianceGatewayA3},
			},
		},
		{
			name: "two groups",
			args: args{
				appliances: map[string][]string{
					"SiteA": {TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3},
				},
				maxUnavailable: 2,
			},
			want: [][]string{
				{TestApplianceGatewayA1, TestApplianceGatewayA2},
				{TestApplianceGatewayA3},
			},
		},
		{
			name: "one group",
			args: args{
				appliances: map[string][]string{
					"SiteA": {TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3},
				},
				maxUnavailable: 3,
			},
			want: [][]string{
				{TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applianceList := make([]string, 0, len(tt.args.appliances))
			for _, a := range tt.args.appliances {
				applianceList = append(applianceList, a...)
			}
			coll := GenerateCollective(t, "appgate.test", "6.3.5", "6.4.1", applianceList)
			applianceMap := make(map[string][]openapi.Appliance)
			for key, list := range tt.args.appliances {
				l := make([]openapi.Appliance, 0, len(list))
				for _, item := range list {
					a := coll.Appliances[item]
					l = append(l, a)
				}
				applianceMap[key] = append(applianceMap[key], l...)
			}
			want := make([][]openapi.Appliance, 0)
			for _, group := range tt.want {
				apps := make([]openapi.Appliance, 0, len(group))
				for _, item := range group {
					apps = append(apps, coll.Appliances[item])
				}
				want = append(want, apps)
			}
			if got := divideByMaxUnavailable(applianceMap, tt.args.maxUnavailable); !assert.EqualValues(t, want, got) {
				t.Errorf("divideByMaxUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
