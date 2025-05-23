package appliance

import (
	"bytes"
	"errors"
	"reflect"
	"runtime"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestPrintDiskSpaceWarningMessage(t *testing.T) {
	type args struct {
		stats      []openapi.ApplianceWithStatus
		apiVersion int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "warning version < 18",
			args: args{
				stats: []openapi.ApplianceWithStatus{
					{
						Name:             "controller",
						Disk:             openapi.PtrFloat32(90),
						ApplianceVersion: openapi.PtrString("5.5.4"),
					},
					{
						Name:             "controller2",
						Disk:             openapi.PtrFloat32(75),
						ApplianceVersion: openapi.PtrString("5.5.4"),
					},
				},
				apiVersion: 17,
			},
			want: `
WARNING: Some appliances have very little space available

Name           Disk Usage
----           ----------
controller     90%
controller2    75%

Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.
`,
		},
		{
			name: "warning version >= 18",
			args: args{
				stats: []openapi.ApplianceWithStatus{
					{
						Name:             "controller",
						Disk:             openapi.PtrFloat32(90),
						ApplianceVersion: openapi.PtrString("5.5.4"),
						Details: &openapi.ApplianceWithStatusAllOfDetails{
							Disk: &openapi.SystemInfo{
								Total: openapi.PtrInt64(int64(100000000000)),
								Used:  openapi.PtrInt64(int64(90000000000)),
								Free:  openapi.PtrInt64(int64(10000000000)),
							},
						},
					},
					{
						Name:             "controller2",
						Disk:             openapi.PtrFloat32(75),
						ApplianceVersion: openapi.PtrString("5.5.4"),
						Details: &openapi.ApplianceWithStatusAllOfDetails{
							Disk: &openapi.SystemInfo{
								Total: openapi.PtrInt64(int64(100000000000)),
								Used:  openapi.PtrInt64(int64(75000000000)),
								Free:  openapi.PtrInt64(int64(25000000000)),
							},
						},
					},
				},
				apiVersion: 18,
			},
			want: `
WARNING: Some appliances have very little space available

Name           Disk Usage (used / total)
----           -------------------------
controller     90.00% (83.82GB / 93.13GB)
controller2    75.00% (69.85GB / 93.13GB)

Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			PrintDiskSpaceWarningMessage(&b, tt.args.stats, tt.args.apiVersion)
			if res := b.String(); res != tt.want {
				t.Errorf("ShowDiskSpaceWarning() - want: %s, got: %s", tt.want, res)
			}
		})
	}
}

func TestHasLowDiskSpace(t *testing.T) {
	type args struct {
		stats []openapi.ApplianceWithStatus
	}
	tests := []struct {
		name string
		args args
		want []openapi.ApplianceWithStatus
	}{
		{
			name: "with low disk space",
			args: args{
				stats: []openapi.ApplianceWithStatus{
					{
						Name: "controller",
						Disk: openapi.PtrFloat32(75),
					},
					{
						Name: "gateway",
						Disk: openapi.PtrFloat32(1),
					},
				},
			},
			want: []openapi.ApplianceWithStatus{
				{
					Name: "controller",
					Disk: openapi.PtrFloat32(75),
				},
			},
		},
		{
			name: "no low disk space",
			args: args{
				stats: []openapi.ApplianceWithStatus{
					{
						Name: "controller",
						Disk: openapi.PtrFloat32(2),
					},
					{
						Name: "gateway",
						Disk: openapi.PtrFloat32(1),
					},
				},
			},
			want: []openapi.ApplianceWithStatus{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasLowDiskSpace(tt.args.stats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HasLowDiskSpace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplianceGroupDescription(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "controller and gateway and connector",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "controller",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "gateway",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "connector",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
			},
			want: "Connector, Controller, Gateway",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applianceGroupDescription(tt.args.appliances); got != tt.want {
				t.Errorf("applianceGroupDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShowAutoscalingWarningMessage(t *testing.T) {
	type args struct {
		templateAppliance *openapi.Appliance
		gateways          []openapi.Appliance
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "show warning",
			args: args{
				templateAppliance: &openapi.Appliance{
					Name: "gateway template",
				},
				gateways: []openapi.Appliance{
					{
						Name: "Autoscaling Instance gateway abc",
					},
				},
			},
			want: `

There is an auto-scale template configured: gateway template


Found 1 auto-scaled gateway(s):

  - Autoscaling Instance gateway abc
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShowAutoscalingWarningMessage(tt.args.templateAppliance, tt.args.gateways)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShowAutoscalingWarningMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", got, tt.want)
			}
		})
	}
}

func TestCompareVersionAndBuildNumber(t *testing.T) {
	testCases := []struct {
		desc    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{
			desc: "should equal",
			v1:   "6.0.0-beta+12345",
			v2:   "6.0.0-beta+12345",
			want: IsEqual,
		},
		{
			desc: "v1 greater than v2",
			v1:   "6.0.0-beta+23456",
			v2:   "6.0.0-beta+12345",
			want: IsLower,
		},
		{
			desc: "v2 greater than v1",
			v1:   "6.0.0-beta+12345",
			v2:   "6.0.0-beta+23456",
			want: IsGreater,
		},
		{
			desc: "v2 no build number",
			v1:   "6.0.1+30125",
			v2:   "6.0.1",
			want: IsEqual,
		},
		{
			desc: "v1 no build number",
			v1:   "6.0.1",
			v2:   "6.0.1+30125",
			want: IsEqual,
		},
		{
			desc: "no build number",
			v1:   "6.0.1",
			v2:   "6.0.1",
			want: IsEqual,
		},
		{
			desc:    "no v1 version",
			v1:      "",
			v2:      "6.0.1",
			wantErr: true,
		},
		{
			desc:    "no v2 version",
			v1:      "6.0.1",
			v2:      "",
			wantErr: true,
		},
		{
			desc:    "no version",
			v1:      "",
			v2:      "",
			wantErr: true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			v1, _ := version.NewVersion(tC.v1)
			v2, _ := version.NewVersion(tC.v2)
			res, err := CompareVersionsAndBuildNumber(v1, v2)
			if err != nil && !tC.wantErr {
				t.Fatal("unexpected error in CompareVersionAndBuildNumber()", err)
			}
			if res != tC.want {
				t.Fatalf("Unexpected version compare:\nWANT\t%d\nGOT\t%d", tC.want, res)
			}
		})
	}
}

func TestHasDiffVersions(t *testing.T) {
	testCases := []struct {
		name   string
		stats  []openapi.ApplianceWithStatus
		expect bool
		count  int // the number of keys in the map that should be returned.
	}{
		{
			name: "should not have diff versions",
			stats: []openapi.ApplianceWithStatus{
				{
					Name:             "controller one",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("6.0.0-12345-release"),
				},
				{
					Name:             "controller two",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("6.0.0-12345-release"),
				},
			},
			expect: false,
			count:  2,
		},
		{
			name: "should have diff versions",
			stats: []openapi.ApplianceWithStatus{
				{
					Name:             "controller primary",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("6.0.0-12345-release"),
				},
				{
					Name:             "controller secondary",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("6.0.0-23456-release"),
				},
				{
					Name:             "portal - the cake is a lie",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("6.0.0-23456-release"),
				},
			},
			expect: true,
			count:  3,
		},
		{
			name: "one offline appliance",
			stats: []openapi.ApplianceWithStatus{
				{
					Name:             "gateway",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("unkown"),
				},
				{
					Name:             "controller one",
					Id:               openapi.PtrString(uuid.NewString()),
					ApplianceVersion: openapi.PtrString("6.0.0-23456-release"),
				},
			},
			expect: true,
			count:  2,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, list := HasDiffVersions(tt.stats)

			if res != tt.expect || len(list) != tt.count {
				for key, value := range list {
					t.Logf("%s %s", key, value)
				}
				t.Fatalf("HasDiffVersions() got list count %d, expect %d - got res %v expected %v", len(list), tt.count, res, tt.expect)
			}
		})
	}
}

func TestGetUpgradeVersionType(t *testing.T) {
	v601, _ := version.NewVersion("6.0.1")
	v602, _ := version.NewVersion("6.0.2")
	v610, _ := version.NewVersion("6.1.0")
	v700, _ := version.NewVersion("7.0.0")
	type args struct {
		x *version.Version
		y *version.Version
	}
	tests := []struct {
		name string
		args args
		want uint8
	}{
		{
			name: "major upgrade",
			args: args{
				x: v610,
				y: v700,
			},
			want: MajorVersion,
		},
		{
			name: "minor upgrade",
			args: args{
				x: v602,
				y: v610,
			},
			want: MinorVersion,
		},
		{
			name: "patch upgrade",
			args: args{
				x: v601,
				y: v602,
			},
			want: PatchVersion,
		},
		{
			name: "equal version",
			args: args{
				x: v601,
				y: v601,
			},
			want: uint8(0),
		},
		{
			name: "downgrade",
			args: args{
				x: v700,
				y: v601,
			},
			want: PatchVersion,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getUpgradeVersionType(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("getUpgradeVersionType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpgradeCheckFunctions(t *testing.T) {
	v601, _ := version.NewVersion("6.0.1")
	v602, _ := version.NewVersion("6.0.2")
	v610, _ := version.NewVersion("6.1.0")
	v700, _ := version.NewVersion("7.0.0")
	type args struct {
		x *version.Version
		y *version.Version
	}
	tests := []struct {
		name      string
		checkFunc func(x, y *version.Version) bool
		args      args
		want      bool
	}{
		{
			name:      "patch version upgrade",
			checkFunc: IsPatchUpgrade,
			args: args{
				x: v601,
				y: v602,
			},
			want: true,
		},
		{
			name:      "patch version downgrade",
			checkFunc: IsPatchUpgrade,
			args: args{
				x: v602,
				y: v601,
			},
			want: false,
		},
		{
			name:      "patch version equal",
			checkFunc: IsPatchUpgrade,
			args: args{
				x: v602,
				y: v602,
			},
			want: false,
		},
		{
			name:      "minor version upgrade",
			checkFunc: IsMinorUpgrade,
			args: args{
				x: v602,
				y: v610,
			},
			want: true,
		},
		{
			name:      "minor version downgrade",
			checkFunc: IsMinorUpgrade,
			args: args{
				x: v610,
				y: v602,
			},
			want: false,
		},
		{
			name:      "minor version equal",
			checkFunc: IsMinorUpgrade,
			args: args{
				x: v610,
				y: v610,
			},
			want: false,
		},
		{
			name:      "major version upgrade",
			checkFunc: IsMajorUpgrade,
			args: args{
				x: v610,
				y: v700,
			},
			want: true,
		},
		{
			name:      "major version downgrade",
			checkFunc: IsMajorUpgrade,
			args: args{
				x: v700,
				y: v610,
			},
			want: false,
		},
		{
			name:      "major version equal",
			checkFunc: IsMajorUpgrade,
			args: args{
				x: v700,
				y: v700,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checkFunc(tt.args.x, tt.args.y); got != tt.want {
				funcName := runtime.FuncForPC(reflect.ValueOf(tt.checkFunc).Pointer()).Name()
				t.Errorf("%v() = %v, want %v", funcName, got, tt.want)
			}
		})
	}
}

func TestCheckNeedsMultiControllerUpgrade(t *testing.T) {
	// unprepared
	c1, s1, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller1", "primary.test", "6.2.0", "", statusHealthy, UpgradeStatusIdle, "default", "default")
	c2, s2, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller2", "", "6.2.0", "", statusHealthy, UpgradeStatusIdle, "default", "default")
	c3, s3, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller3", "", "6.2.0", "", statusHealthy, UpgradeStatusIdle, "default", "default")

	// prepared
	c4, s4, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller4", "", "6.2.0", "6.3.0", statusHealthy, UpgradeStatusReady, "default", "default")
	c5, s5, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller5", "", "6.2.0", "6.3.0", statusHealthy, UpgradeStatusReady, "default", "default")
	c6, s6, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller6", "", "6.2.0", "6.3.0", statusHealthy, UpgradeStatusReady, "default", "default")
	c12, s12, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller12", "", "6.3.0", "6.3.1", statusHealthy, UpgradeStatusReady, "default", "default")

	// unprepared max version
	c7, s7, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller7", "", "6.3.0", "", statusHealthy, UpgradeStatusIdle, "default", "default")
	c9, s9, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller9", "", "6.3.1", "", statusHealthy, UpgradeStatusIdle, "default", "default")
	c10, s10, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller10", "", "6.3.1", "", statusHealthy, UpgradeStatusIdle, "default", "default")
	c11, s11, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller11", "", "6.3.1", "", statusHealthy, UpgradeStatusIdle, "default", "default")

	// offline
	c8, s8, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller8", "", "6.2.0", "", statusOffline, UpgradeStatusIdle, "default", "default")

	type inData struct {
		appliance openapi.Appliance
		stat      openapi.ApplianceWithStatus
	}
	tests := []struct {
		name      string
		in        []inData
		want      []openapi.Appliance
		wantErr   bool
		wantError error
	}{
		{
			name: "none prepared",
			in: []inData{
				{c1, s1},
				{c2, s2},
				{c3, s3},
			},
		},
		{
			name: "all prepared",
			in: []inData{
				{c4, s4},
				{c5, s5},
				{c6, s6},
			},
		},
		{
			name: "mix unprepared and prepared",
			in: []inData{
				{c1, s1},
				{c2, s2},
				{c4, s4},
				{c5, s5},
			},
			wantErr: false,
			want:    []openapi.Appliance{c1, c2},
		},
		{
			name: "mix with unprepared max version",
			in: []inData{
				{c1, s1},
				{c4, s4},
				{c7, s7},
			},
			wantErr: false,
			want:    []openapi.Appliance{c1},
		},
		{
			name: "offline controller",
			in: []inData{
				{c4, s4},
				{c8, s8},
			},
		},
		{
			name: "only one left to upgrade",
			in: []inData{
				{c12, s12},
				{c9, s9},
				{c10, s10},
				{c11, s11},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := openapi.ApplianceWithStatusList{}
			argAppliances := make([]openapi.Appliance, 0, len(tt.in))
			for _, d := range tt.in {
				stats.Data = append(stats.Data, d.stat)
				argAppliances = append(argAppliances, d.appliance)
			}
			upgradeStatusMap := map[string]UpgradeStatusResult{}
			for _, s := range stats.GetData() {
				us := s.GetDetails().Upgrade
				upgradeStatusMap[s.GetId()] = UpgradeStatusResult{
					Status:  us.GetStatus(),
					Details: us.GetDetails(),
					Name:    s.GetName(),
				}
			}
			got, err := CheckNeedsMultiControllerUpgrade(&stats, upgradeStatusMap, argAppliances)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckNeedsMultiControllerUpgrade() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) == tt.wantErr && !errors.Is(err, tt.wantError) {
				t.Errorf("CheckNeedsMultiControllerUpgrade() error = %v, wantError %v", err, tt.wantError)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckApplianceVersionsDisallowed(t *testing.T) {
	type args struct {
		currentVersion string
		targetVersion  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test 6.0.0->6.2.0",
			args: args{
				currentVersion: "6.0.0",
				targetVersion:  "6.2.0",
			},
			wantErr: true,
		},
		{
			name: "test 6.3.5->6.4.0",
			args: args{
				currentVersion: "6.3.5",
				targetVersion:  "6.4.0",
			},
			wantErr: true,
		},
		{
			name: "test 6.3.6->6.4.0",
			args: args{
				currentVersion: "6.3.6",
				targetVersion:  "6.4.0",
			},
			wantErr: true,
		},
		{
			name: "test 6.3.4->6.4.0",
			args: args{
				currentVersion: "6.3.4",
				targetVersion:  "6.4.0",
			},
			wantErr: false,
		},
		{
			name: "test 6.3.5->6.4.1",
			args: args{
				currentVersion: "6.3.5",
				targetVersion:  "6.4.1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentVersion, _ := version.NewVersion(tt.args.currentVersion)
			targetVersion, _ := version.NewVersion(tt.args.targetVersion)
			if err := CheckApplianceVersionsDisallowed(currentVersion, targetVersion); (err != nil) != tt.wantErr {
				t.Errorf("CheckApplianceVersionsDisallowed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
