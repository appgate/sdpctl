package appliance

import (
	"reflect"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
)

func TestFilterAvailable(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
		stats      []openapi.StatsAppliancesListAllOfData
	}
	tests := []struct {
		name        string
		args        args
		wantOnline  []openapi.Appliance
		wantOffline []openapi.Appliance
		wantErr     bool
	}{
		{
			name: "one available no errors",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "gateway",
						Id:   "two",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Id:     openapi.PtrString("one"),
						Online: openapi.PtrBool(true),
					},
					{
						Id:     openapi.PtrString("two"),
						Online: openapi.PtrBool(false),
					},
				},
			},
			wantOnline: []openapi.Appliance{
				{
					Name: "primary controller",
					Id:   "one",
					Controller: &openapi.ApplianceAllOfController{
						Enabled: openapi.PtrBool(true),
					},
				},
			},
			wantOffline: []openapi.Appliance{
				{
					Name: "gateway",
					Id:   "two",
					Gateway: &openapi.ApplianceAllOfGateway{
						Enabled: openapi.PtrBool(true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "one available one offline controller logserver want error",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   "two",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				stats: []openapi.StatsAppliancesListAllOfData{
					{
						Id:     openapi.PtrString("one"),
						Online: openapi.PtrBool(true),
					},
					{
						Id:     openapi.PtrString("two"),
						Online: openapi.PtrBool(false),
					},
				},
			},
			wantOnline: []openapi.Appliance{
				{
					Name: "primary controller",
					Id:   "one",
					Controller: &openapi.ApplianceAllOfController{
						Enabled: openapi.PtrBool(true),
					},
				},
			},
			wantOffline: []openapi.Appliance{
				{
					Name: "secondary controller with log server",
					Id:   "two",
					Controller: &openapi.ApplianceAllOfController{
						Enabled: openapi.PtrBool(true),
					},
					LogServer: &openapi.ApplianceAllOfLogServer{
						Enabled: openapi.PtrBool(true),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			online, offline, err := FilterAvailable(tt.args.appliances, tt.args.stats)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilterAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(online, tt.wantOnline) {
				t.Errorf("FilterAvailable() got = %v, want %v", online, tt.wantOnline)
			}
			if !reflect.DeepEqual(offline, tt.wantOffline) {
				t.Errorf("FilterAvailable() got offline = %v, want %v", offline, tt.wantOffline)
			}
		})
	}
}
