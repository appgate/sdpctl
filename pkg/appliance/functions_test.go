package appliance

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
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

func TestFindPrimaryController(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
		hostname   string
	}
	tests := []struct {
		name    string
		args    args
		want    *openapi.Appliance
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   "two",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foobar.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				hostname: "foo.devops",
			},
			want: &openapi.Appliance{
				Name: "primary controller",
				Id:   "one",
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(true),
				},
				AdminInterface: &openapi.ApplianceAllOfAdminInterface{
					Hostname: "foo.devops",
				},
			},
			wantErr: false,
		},
		{
			name: "no hit",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   "two",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
						PeerInterface: openapi.ApplianceAllOfPeerInterface{
							Hostname: "foo.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				hostname: "controller.devops",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindPrimaryController(tt.args.appliances, tt.args.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindPrimaryController() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("FindPrimaryController() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupByFunctions(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
	}
	tests := []struct {
		name string
		args args
		want map[string][]openapi.Appliance
	}{
		{
			name: "two controllers log server gateway",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   "two",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
						PeerInterface: openapi.ApplianceAllOfPeerInterface{
							Hostname: "foo.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
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
			},
			want: map[string][]openapi.Appliance{
				FunctionController: {
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   "two",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
						PeerInterface: openapi.ApplianceAllOfPeerInterface{
							Hostname: "foo.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				FunctionGateway: {
					{
						Name: "gateway",
						Id:   "two",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				FunctionLogServer: {
					{
						Name: "secondary controller with log server",
						Id:   "two",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
						PeerInterface: openapi.ApplianceAllOfPeerInterface{
							Hostname: "foo.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupByFunctions(tt.args.appliances)
			if !cmp.Equal(got, tt.want) {
				t.Errorf(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestActiveFunctions(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
	}
	tests := []struct {
		name string
		args args
		want map[string]bool
	}{
		{
			name: "one active controller",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
				},
			},
			want: map[string]bool{
				FunctionController: true,
			},
		},
		{
			name: "one active controller and gateway",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   "one",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "gateway",
						Id:   "two",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "portal",
						Id:   "three",
						Portal: &openapi.Portal{
							Enabled: openapi.PtrBool(false),
						},
					},
				},
			},
			want: map[string]bool{
				FunctionController: true,
				FunctionGateway:    true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActiveFunctions(tt.args.appliances); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ActiveFunctions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterAndExclude(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
		filter     map[string]map[string]string
	}
	mockControllers := map[string]openapi.Appliance{
		"primaryController": {
			Name: "primary controller",
			Id:   "f1bef0c4-e0b6-42ac-9c40-3f6214c34869",
			Controller: &openapi.ApplianceAllOfController{
				Enabled: openapi.PtrBool(true),
			},
			LogServer: &openapi.ApplianceAllOfLogServer{
				Enabled: openapi.PtrBool(true),
			},
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname: "foo.devops",
			},
			Hostname:  openapi.PtrString("foo.devops"),
			Site:      openapi.PtrString("640039ab-8b13-494a-af9e-20a48846674a"),
			Activated: openapi.PtrBool(true),
			Tags: &[]string{
				"primary",
				"Jebediah Kerman",
			},
			Version: openapi.PtrInt32(16),
		},
		"secondaryController": {
			Name: "secondary controller",
			Id:   "6090fd66-6e21-4ef5-87d0-36c7a1b04a80",
			Controller: &openapi.ApplianceAllOfController{
				Enabled: openapi.PtrBool(true),
			},
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname: "bar.purple",
			},
			Site:      openapi.PtrString("3976e914-ccf4-4704-80e1-18b7de87ff07"),
			Activated: openapi.PtrBool(false),
			Tags: &[]string{
				"secondary",
				"crap",
			},
			Version: openapi.PtrInt32(15),
		},
		"gateway": {
			Name: "gateway",
			Id:   "85fac76b-c526-486d-844a-520a023e76e2",
			Gateway: &openapi.ApplianceAllOfGateway{
				Enabled: openapi.PtrBool(true),
			},
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname: "tinker.purple",
			},
			Activated: openapi.PtrBool(false),
			Site:      openapi.PtrString("15dd5630-aaf5-4d74-8c75-205e438db9a3"),
			Tags: &[]string{
				"stargate",
			},
			Version: openapi.PtrInt32(15),
		},
	}
	keywords := map[string]string{
		"name":      "primary",
		"id":        "f1bef0c4-e0b6-42ac-9c40-3f6214c34869",
		"tags":      "Jeb",
		"tag":       "Jeb",
		"version":   "16",
		"hostname":  "foo.devops",
		"host":      "foo.devops",
		"active":    "true",
		"activated": "true",
		"site":      "640039ab-8b13-494a-af9e-20a48846674a",
		"site-id":   "640039ab-8b13-494a-af9e-20a48846674a",
		"function":  "logserver",
	}
	type testStruct struct {
		name string
		args args
		want []openapi.Appliance
	}
	tests := []testStruct{}
	for word, value := range keywords {
		tests = append(tests, testStruct{
			name: fmt.Sprintf("filter by %s", word),
			args: args{
				appliances: []openapi.Appliance{
					mockControllers["primaryController"],
					mockControllers["secondaryController"],
					mockControllers["gateway"],
				},
				filter: map[string]map[string]string{
					"filter": {
						word: value,
					},
				},
			},
			want: []openapi.Appliance{
				mockControllers["primaryController"],
			},
		})
		tests = append(tests, testStruct{
			name: fmt.Sprintf("filter by %s", word),
			args: args{
				appliances: []openapi.Appliance{
					mockControllers["primaryController"],
					mockControllers["secondaryController"],
					mockControllers["gateway"],
				},
				filter: map[string]map[string]string{
					"exclude": {
						word: value,
					},
				},
			},
			want: []openapi.Appliance{
				mockControllers["secondaryController"],
				mockControllers["gateway"],
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterAppliances(tt.args.appliances, tt.args.filter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterAppliances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldDisable(t *testing.T) {
	tests := []struct {
		From, To string
		Expect   bool
	}{
		{
			From:   "5.3",
			To:     "5.4",
			Expect: true,
		},
		{
			From:   "5.4",
			To:     "5.5",
			Expect: false,
		},
		{
			From:   "5.2.1",
			To:     "5.3.1",
			Expect: true,
		},
		{
			From:   "5.4.1",
			To:     "5.4.2",
			Expect: false,
		},
		{
			From:   "5.5.1",
			To:     "5.5.2",
			Expect: false,
		},
		{
			From:   "5.5",
			To:     "6.0",
			Expect: false,
		},
		{
			From:   "5.2.0",
			To:     "5.4.1",
			Expect: true,
		},
		{
			From:   "4.5.2",
			To:     "5.5.2",
			Expect: true,
		},
	}

	for _, tt := range tests {
		from, _ := version.NewVersion(tt.From)
		to, _ := version.NewVersion(tt.To)
		if res := ShouldDisable(from, to); res != tt.Expect {
			t.Errorf("want: %v, got: %v", tt.Expect, res)
		}
	}
}
