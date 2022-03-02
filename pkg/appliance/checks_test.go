package appliance

import (
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/go-cmp/cmp"
)

func TestShowDiskSpaceWarningMessage(t *testing.T) {
	type args struct {
		stats []openapi.StatsAppliancesListAllOfData
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
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
						Name:    openapi.PtrString("gateway"),
						Disk:    openapi.PtrFloat32(4),
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
Some appliances have very little space available

  - controller  Disk usage: 90%
  - controller2  Disk usage: 75%

Upgrading requires the upload and decompression of big images.
To avoid problems during the upgrade process it's recommended to
increase the space on those appliances.
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShowDiskSpaceWarningMessage(tt.args.stats)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShowDiskSpaceWarningMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", got, tt.want)
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
		want bool
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
			want: true,
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
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasLowDiskSpace(tt.args.stats); got != tt.want {
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
			want: "connector, controller, gateway",
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
The following controller is still configured without the Admin/API TLS Connection:

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
