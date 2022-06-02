package appliance

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
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

Name         Disk Usage
----         ----------
controller   90%
controller2  75%

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
						PeerInterface: openapi.ApplianceAllOfPeerInterface{
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
		desc string
		v1   string
		v2   string
		want int
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
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			v1, _ := version.NewVersion(tC.v1)
			v2, _ := version.NewVersion(tC.v2)
			if res := CompareVersionsAndBuildNumber(v1, v2); res != tC.want {
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
	}{
		{
			name: "should not have diff versions",
			stats: []openapi.StatsAppliancesListAllOfData{
				{
					Name:    openapi.PtrString("controller"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-12345-release"),
				},
				{
					Name:    openapi.PtrString("controller"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-12345-release"),
				},
			},
			expect: false,
		},
		{
			name: "should have diff versions",
			stats: []openapi.StatsAppliancesListAllOfData{
				{
					Name:    openapi.PtrString("controller"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-12345-release"),
				},
				{
					Name:    openapi.PtrString("controller"),
					Id:      openapi.PtrString(uuid.NewString()),
					Version: openapi.PtrString("6.0.0-23456-release"),
				},
			},
			expect: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if res, _ := HasDiffVersions(tt.stats); res != tt.expect {
				t.Fatalf("HasDiffVersions() failed\nWANT: %v\nGOT: %v", tt.expect, res)
			}
		})
	}
}
