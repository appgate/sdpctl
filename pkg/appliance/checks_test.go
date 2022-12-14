package appliance

import (
	"bytes"
	"reflect"
	"runtime"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
)

func TestPrintDiskSpaceWarningMessage(t *testing.T) {
	type args struct {
		stats []openapi.StatsAppliancesListAllOfData
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "warning",
			args: args{
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Name:    openapi.PtrString("controller"),
						Disk:    openapi.PtrFloat32(90),
						Version: openapi.PtrString("5.5.4"),
					},
					{
						Name:    openapi.PtrString("controller2"),
						Disk:    openapi.PtrFloat32(75),
						Version: openapi.PtrString("5.5.4"),
					},
				},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			PrintDiskSpaceWarningMessage(&b, tt.args.stats)
			if res := b.String(); res != tt.want {
				t.Errorf("ShowDiskSpaceWarning() - want: %s, got: %s", tt.want, res)
			}
		})
	}
}

func TestHasLowDiskSpace(t *testing.T) {
	type args struct {
		stats []openapi.StatsAppliancesListAllOfData
	}
	tests := []struct {
		name string
		args args
		want []openapi.StatsAppliancesListAllOfData
	}{
		{
			name: "with low disk space",
			args: args{
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Name: openapi.PtrString("controller"),
						Disk: openapi.PtrFloat32(75),
					},
					{
						Name: openapi.PtrString("gateway"),
						Disk: openapi.PtrFloat32(1),
					},
				},
			},
			want: []openapi.StatsAppliancesListAllOfData{
				{
					Name: openapi.PtrString("controller"),
					Disk: openapi.PtrFloat32(75),
				},
			},
		},
		{
			name: "no low disk space",
			args: args{
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Name: openapi.PtrString("controller"),
						Disk: openapi.PtrFloat32(2),
					},
					{
						Name: openapi.PtrString("gateway"),
						Disk: openapi.PtrFloat32(1),
					},
				},
			},
			want: []openapi.StatsAppliancesListAllOfData{},
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

func TestShowPeerInterfaceWarningMessage(t *testing.T) {
	type args struct {
		peerAppliances []openapi.Appliance
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "prepare example",
			args: args{
				peerAppliances: []openapi.Appliance{
					{
						Name: "controller",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						PeerInterface: &openapi.ApplianceAllOfPeerInterface{
							HttpsPort: openapi.PtrInt32(443),
						},
					},
				},
			},
			want: `
Version 5.4 and later are designed to operate with the admin port (default 8443)
separate from the deprecated peer port (set to 443).
It is recommended to switch to port 8443 before continuing
The following Controller is still configured without the Admin/API TLS Connection:

  - controller
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShowPeerInterfaceWarningMessage(tt.args.peerAppliances)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShowPeerInterfaceWarningMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", got, tt.want)
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


Found 1 auto-scaled gateway running version < 16:

  - Autoscaling Instance gateway abc

Make sure that the health check for those auto-scaled gateways is disabled.
Not disabling the health checks in those auto-scaled gateways could cause them to be deleted, breaking all the connections established with them.
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
		stats  []openapi.StatsAppliancesListAllOfData
		expect bool
		count  int // the number of keys in the map that should be returned.
	}{
		{
			name: "should not have diff versions",
			stats: []openapi.StatsAppliancesListAllOfData{
				{
					Name:    openapi.PtrString("controller one"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-12345-release"),
				},
				{
					Name:    openapi.PtrString("controller two"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-12345-release"),
				},
			},
			expect: false,
			count:  2,
		},
		{
			name: "should have diff versions",
			stats: []openapi.StatsAppliancesListAllOfData{
				{
					Name:    openapi.PtrString("controller primary"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-12345-release"),
				},
				{
					Name:    openapi.PtrString("controller secondary"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-23456-release"),
				},
				{
					Name:    openapi.PtrString("portal - the cake is a lie"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-23456-release"),
				},
			},
			expect: true,
			count:  3,
		},
		{
			name: "one offline appliance",
			stats: []openapi.StatsAppliancesListAllOfData{
				{
					Name:    openapi.PtrString("gateway"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("unkown"),
				},
				{
					Name:    openapi.PtrString("controller one"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-23456-release"),
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
