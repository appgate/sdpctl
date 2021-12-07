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
						Name: openapi.PtrString("controller"),
						Disk: openapi.PtrFloat32(90),
					},
				},
			},
			want: `
Some appliances have very little space available

  - controller  Disk usage: 90%


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
			want: "controller, gateway, connector",
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
