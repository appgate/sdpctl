package appliance

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/appgate/sdpctl/pkg/hashcode"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
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

func TestSplitAppliancesByGroup(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
	}
	sites := map[string]*string{
		"A": openapi.PtrString("d9fd012a-212a-4b90-9a63-63fef93a834b"),
		"B": openapi.PtrString("aa01780b-3e3c-408f-80b4-59eb5c1d4b4a"),
		"C": openapi.PtrString("23b2ca4b-cfa8-4d20-a6b3-219952cc4468"),
	}
	tests := []struct {
		name string
		args args
		want map[int][]openapi.Appliance
	}{
		{
			name: "gateway different sites",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "g1",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: sites["C"],
					},
				},
			},
			want: map[int][]openapi.Appliance{
				hashcode.String(fmt.Sprintf("%s%s", *sites["A"], "&gateway=true")): {
					{
						Name: "g1",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g1.devops",
						},
						Site: sites["A"],
					},
				},
				hashcode.String(fmt.Sprintf("%s%s", *sites["B"], "&gateway=true")): {
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["B"],
					},
				},
				hashcode.String(fmt.Sprintf("%s%s", *sites["C"], "&gateway=true")): {
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: sites["C"],
					},
				},
			},
		},
		{
			name: "split appliance by site group",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "c1",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "g1",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "c2",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: nil,
					},
					{
						Name: "g5",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(false),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g5.devops",
						},
						Site: nil,
					},
					{
						Name: "cc1",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "cc11.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "lf1",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf1.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "lf2",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "lf3",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf3.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "cc2",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "cc2.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g6",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g6.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g7",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g7.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "c3",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c3.devops",
						},
						Site: sites["C"],
					},
				},
			},
			want: map[int][]openapi.Appliance{
				hashcode.String("controller=true"): {
					// all controllers
					{
						Name: "c1",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "c2",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "c3",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c3.devops",
						},
						Site: sites["C"],
					},
				},
				// gateway site A
				hashcode.String(fmt.Sprintf("%s%s", *sites["A"], "&gateway=true")): {
					{
						Name: "g1",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g1.devops",
						},
						Site: sites["A"],
					},
				},
				// disabled gateway no site assigned
				hashcode.String(fmt.Sprintf("%s%s", "", "&gateway=false")): {
					{
						Name: "g5",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(false),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g5.devops",
						},
						Site: nil,
					},
				},
				// logforwader site A
				hashcode.String(fmt.Sprintf("%s%s", *sites["A"], "&log_forwarder=true")): {
					{
						Name: "lf3",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf3.devops",
						},
						Site: sites["A"],
					},
				},
				// connector site C
				hashcode.String(fmt.Sprintf("%s%s", *sites["C"], "&connector=true")): {
					{
						Name: "cc2",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "cc2.devops",
						},
						Site: sites["C"],
					},
				},
				// gateways site B
				hashcode.String(fmt.Sprintf("%s%s", *sites["B"], "&gateway=true")): {
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["B"],
					},
				},
				// Enabled gateway no site
				hashcode.String(fmt.Sprintf("%s%s", "", "&gateway=true")): {
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: nil,
					},
				},
				// connector site B
				hashcode.String(fmt.Sprintf("%s%s", *sites["B"], "&connector=true")): {
					{
						Name: "cc1",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "cc11.devops",
						},
						Site: sites["B"],
					},
				},
				// logforwaders site B
				hashcode.String(fmt.Sprintf("%s%s", *sites["B"], "&log_forwarder=true")): {
					{
						Name: "lf1",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf1.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "lf2",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf2.devops",
						},
						Site: sites["B"],
					},
				},
				// enabled gateways site C
				hashcode.String(fmt.Sprintf("%s%s", *sites["C"], "&gateway=true")): {
					{
						Name: "g6",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g6.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g7",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g7.devops",
						},
						Site: sites["C"],
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitAppliancesByGroup(tt.args.appliances)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitAppliancesByGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplianceGroupHash(t *testing.T) {
	site := openapi.PtrString(uuid.New().String())
	tests := []struct {
		name      string
		appliance openapi.Appliance
		expect    int
	}{
		{
			name: "log forwarder enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(true),
				},
				LogServer: &openapi.ApplianceAllOfLogServer{
					Enabled: openapi.PtrBool(false),
				},
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(false),
				},
				Connector: &openapi.ApplianceAllOfConnector{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&log_forwarder=true&log_server=false&gateway=false&connector=false")),
		},
		{
			name: "log server enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(false),
				},
				LogServer: &openapi.ApplianceAllOfLogServer{
					Enabled: openapi.PtrBool(true),
				},
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(false),
				},
				Connector: &openapi.ApplianceAllOfConnector{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&log_forwarder=false&log_server=true&gateway=false&connector=false")),
		},
		{
			name: "gateway enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(false),
				},
				LogServer: &openapi.ApplianceAllOfLogServer{
					Enabled: openapi.PtrBool(false),
				},
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(true),
				},
				Connector: &openapi.ApplianceAllOfConnector{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&log_forwarder=false&log_server=false&gateway=true&connector=false")),
		},
		{
			name: "connector enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(false),
				},
				LogServer: &openapi.ApplianceAllOfLogServer{
					Enabled: openapi.PtrBool(false),
				},
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(false),
				},
				Connector: &openapi.ApplianceAllOfConnector{
					Enabled: openapi.PtrBool(true),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&log_forwarder=false&log_server=false&gateway=false&connector=true")),
		},
		{
			name: "controller enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(false),
				},
				LogServer: &openapi.ApplianceAllOfLogServer{
					Enabled: openapi.PtrBool(false),
				},
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(false),
				},
				Connector: &openapi.ApplianceAllOfConnector{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(true),
				},
				Site: site,
			},
			expect: hashcode.String("controller=true"),
		},
		{
			name: "controller and gateway enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(false),
				},
				LogServer: &openapi.ApplianceAllOfLogServer{
					Enabled: openapi.PtrBool(false),
				},
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(true),
				},
				Connector: &openapi.ApplianceAllOfConnector{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(true),
				},
				Site: site,
			},
			expect: hashcode.String("controller=true"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := applianceGroupHash(tt.appliance); result != tt.expect {
				t.Errorf("FAILED! Expected: %d, Got: %d", tt.expect, result)
			}
		})
	}
}

func TestChunkApplianceGroupLength(t *testing.T) {
	sites := map[string]*string{
		"A": openapi.PtrString("d9fd012a-212a-4b90-9a63-63fef93a834b"),
		"B": openapi.PtrString("aa01780b-3e3c-408f-80b4-59eb5c1d4b4a"),
		"C": openapi.PtrString("23b2ca4b-cfa8-4d20-a6b3-219952cc4468"),
	}
	chunkSize := 3
	appliances := map[int][]openapi.Appliance{
		335032170: {
			{
				Name: "g1-siteA",
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(true),
				},
				AdminInterface: &openapi.ApplianceAllOfAdminInterface{
					Hostname: "g1.devops",
				},
				Site: sites["A"],
			},
			{
				Name: "g2-siteB",
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(true),
				},
				AdminInterface: &openapi.ApplianceAllOfAdminInterface{
					Hostname: "g2.devops",
				},
				Site: sites["B"],
			},
			{
				Name: "g3-siteC",
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(true),
				},
				AdminInterface: &openapi.ApplianceAllOfAdminInterface{
					Hostname: "g3.devops",
				},
				Site: sites["C"],
			},
			{
				Name: "g4-siteC",
				Gateway: &openapi.ApplianceAllOfGateway{
					Enabled: openapi.PtrBool(true),
				},
				AdminInterface: &openapi.ApplianceAllOfAdminInterface{
					Hostname: "g4.devops",
				},
				Site: sites["C"],
			},
		},
	}
	count := 0
	got := ChunkApplianceGroup(chunkSize, appliances)
	for _, v := range got {
		for range v {
			count += 1
		}
	}
	if count != 4 {
		t.Fatalf("Expected 4, got %d", count)
	}
}

func TestChunkApplianceGroup(t *testing.T) {
	type args struct {
		divisor    int
		appliances map[int][]openapi.Appliance
	}
	sites := map[string]*string{
		"A": openapi.PtrString("d9fd012a-212a-4b90-9a63-63fef93a834b"),
		"B": openapi.PtrString("aa01780b-3e3c-408f-80b4-59eb5c1d4b4a"),
		"C": openapi.PtrString("23b2ca4b-cfa8-4d20-a6b3-219952cc4468"),
	}
	tests := []struct {
		name string
		args args
		want [][]openapi.Appliance
	}{
		{
			name: "test empty",
			args: args{
				divisor:    2,
				appliances: make(map[int][]openapi.Appliance),
			},
			want: nil,
		},
		{
			name: "1 controller 2 sites with 2 gateways each",
			args: args{
				divisor: 2,
				appliances: map[int][]openapi.Appliance{
					276419119: {
						// all controllers
						{
							Name: "c1",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "c1.devops",
							},
							Site: sites["A"],
						},
					},
					// gateway site A
					2441219521: {
						{
							Name: "g1",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g1.devops",
							},
							Site: sites["A"],
						},
						{
							Name: "g4",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g4.devops",
							},
							Site: sites["A"],
						},
					},
					675122154: {
						{
							Name: "g2",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g2.devops",
							},
							Site: sites["B"],
						},
						{
							Name: "g3",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g3.devops",
							},
							Site: sites["B"],
						},
					},
				},
			},
			want: [][]openapi.Appliance{
				// array 0
				{
					{
						Name: "c1",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: sites["A"],
					},
				},
				// array 1
				{
					{
						Name: "g1",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["B"],
					},
				},
			},
		},
		{
			name: "multiple controllers and gateways throughout different sites",
			args: args{
				divisor: 3,
				appliances: map[int][]openapi.Appliance{
					2764191119: {
						// all controllers
						{
							Name: "c1",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "c1.devops",
							},
							Site: sites["A"],
						},
						{
							Name: "c2",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "c2.devops",
							},
							Site: sites["B"],
						},
						{
							Name: "c3",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "c3.devops",
							},
							Site: sites["C"],
						},
					},
					// gateway site A
					24421219521: {
						{
							Name: "g1",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g1.devops",
							},
							Site: sites["A"],
						},
					},
					// disabled gateway no site assigned
					41799232720: {
						{
							Name: "g5",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(false),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g5.devops",
							},
							Site: nil,
						},
					},
					// logforwader site A
					14065405277: {
						{
							Name: "lf3",
							LogForwarder: &openapi.ApplianceAllOfLogForwarder{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "lf3.devops",
							},
							Site: sites["A"],
						},
					},
					// connector site C
					7357445990: {
						{
							Name: "cc2",
							Connector: &openapi.ApplianceAllOfConnector{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "cc2.devops",
							},
							Site: sites["C"],
						},
					},
					// gateways site B
					6751262154: {
						{
							Name: "g2",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g2.devops",
							},
							Site: sites["B"],
						},
						{
							Name: "g3",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g3.devops",
							},
							Site: sites["B"],
						},
					},
					// Enabled gateway no site
					4298276475: {
						{
							Name: "g4",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g4.devops",
							},
							Site: nil,
						},
					},
					// connector site B
					32808589746: {
						{
							Name: "cc1",
							Connector: &openapi.ApplianceAllOfConnector{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "cc11.devops",
							},
							Site: sites["B"],
						},
					},
					// logforwaders site B
					24079971497: {
						{
							Name: "lf1",
							LogForwarder: &openapi.ApplianceAllOfLogForwarder{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "lf1.devops",
							},
							Site: sites["B"],
						},
						{
							Name: "lf2",
							LogForwarder: &openapi.ApplianceAllOfLogForwarder{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "lf2.devops",
							},
							Site: sites["B"],
						},
					},
					// enabled gateways site C
					121861035433: {
						{
							Name: "g6",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g6.devops",
							},
							Site: sites["C"],
						},
						{
							Name: "g7",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g7.devops",
							},
							Site: sites["C"],
						},
					},
				},
			},
			want: [][]openapi.Appliance{
				// index 0
				{
					{
						Name: "c3",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c3.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "cc1",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "cc11.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "cc2",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "cc2.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g1",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: nil,
					},
					{
						Name: "g5",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(false),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g5.devops",
						},
						Site: nil,
					},
					{
						Name: "g7",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g7.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "lf2",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "lf3",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf3.devops",
						},
						Site: sites["A"],
					},
				},
				// index 1
				{
					{
						Name: "c2",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g6",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g6.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "lf1",
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "lf1.devops",
						},
						Site: sites["B"],
					},
				},
				// index 2
				{
					{
						Name: "c1",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: sites["A"],
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantAppliances := 0
			for _, slice := range tt.args.appliances {
				for range slice {
					wantAppliances += 1
				}
			}

			got := ChunkApplianceGroup(tt.args.divisor, tt.args.appliances)
			gotAppliances := 0
			for _, slice := range got {
				for range slice {
					gotAppliances += 1
				}
			}
			if wantAppliances != gotAppliances {
				t.Fatalf("Got %d appliances, expected %d", gotAppliances, wantAppliances)
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("Got diff in\n%s\n", cmp.Diff(tt.want, got, cmpopts.IgnoreFields(openapi.Appliance{}, "Controller")))
			}
		})
	}
}

func TestActiveSitesInAppliances(t *testing.T) {
	type args struct {
		slice []openapi.Appliance
	}
	sites := map[string]*string{
		"A": openapi.PtrString("d9fd012a-212a-4b90-9a63-63fef93a834b"),
		"B": openapi.PtrString("aa01780b-3e3c-408f-80b4-59eb5c1d4b4a"),
		"C": openapi.PtrString("23b2ca4b-cfa8-4d20-a6b3-219952cc4468"),
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "count sites",
			args: args{
				slice: []openapi.Appliance{
					{
						Name: "c1",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: sites["A"],
					},
					{
						Name: "c2",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c2.devops",
						},
						Site: nil,
					},
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["B"],
					},
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["C"],
					},
					{
						Name: "g4",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g4.devops",
						},
						Site: sites["C"],
					},
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActiveSitesInAppliances(tt.args.slice); got != tt.want {
				t.Errorf("ActiveSitesInAppliances() = %v, want %v", got, tt.want)
			}
		})
	}
}
