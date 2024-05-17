package appliance

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/dns"
	"github.com/appgate/sdpctl/pkg/hashcode"
	"github.com/foxcpp/go-mockdns"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
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
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "gateway",
						Id:   openapi.PtrString("two"),
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
					Id:   openapi.PtrString("one"),
					Controller: &openapi.ApplianceAllOfController{
						Enabled: openapi.PtrBool(true),
					},
				},
			},
			wantOffline: []openapi.Appliance{
				{
					Name: "gateway",
					Id:   openapi.PtrString("two"),
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
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
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
					Id:   openapi.PtrString("one"),
					Controller: &openapi.ApplianceAllOfController{
						Enabled: openapi.PtrBool(true),
					},
				},
			},
			wantOffline: []openapi.Appliance{
				{
					Name: "secondary controller with log server",
					Id:   openapi.PtrString("two"),
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
		validate   bool
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
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "appgate.test",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "otherhost",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				hostname: "appgate.test",
				validate: true,
			},
			want: &openapi.Appliance{
				Name: "primary controller",
				Id:   openapi.PtrString("one"),
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(true),
				},
				AdminInterface: &openapi.ApplianceAllOfAdminInterface{
					Hostname: "appgate.test",
				},
			},
			wantErr: false,
		},
		{
			name: "no hostname",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "localhost",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "localhost",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				hostname: "appgate.test",
				validate: true,
			},
			wantErr: true,
		},
		{
			name: "no hit",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "primary controller",
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "localhost",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "localhost",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						LogServer: &openapi.ApplianceAllOfLogServer{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				hostname: "appgate.test",
				validate: true,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{})
			defer teardown()
			got, err := FindPrimaryController(tt.args.appliances, tt.args.hostname, tt.args.validate)
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
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
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
						Id:   openapi.PtrString("two"),
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
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
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
						Id:   openapi.PtrString("two"),
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
					},
				},
				FunctionLogServer: {
					{
						Name: "secondary controller with log server",
						Id:   openapi.PtrString("two"),
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
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
						Id:   openapi.PtrString("one"),
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
						Id:   openapi.PtrString("one"),
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "foo.devops",
						},
					},
					{
						Name: "gateway",
						Id:   openapi.PtrString("two"),
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
					},
					{
						Name: "portal",
						Id:   openapi.PtrString("three"),
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
		orderBy    []string
		descending bool
	}
	mockControllers := map[string]openapi.Appliance{
		"primaryController": {
			Name: "primary controller",
			Id:   openapi.PtrString("f1bef0c4-e0b6-42ac-9c40-3f6214c34869"),
			Controller: &openapi.ApplianceAllOfController{
				Enabled: openapi.PtrBool(true),
			},
			LogServer: &openapi.ApplianceAllOfLogServer{
				Enabled: openapi.PtrBool(true),
			},
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname: "foo.devops",
			},
			Hostname:  "foo.devops",
			Site:      openapi.PtrString("640039ab-8b13-494a-af9e-20a48846674a"),
			Activated: openapi.PtrBool(true),
			Tags: []string{
				"primary",
				"Jebediah Kerman",
			},
			Version: openapi.PtrInt32(16),
		},
		"secondaryController": {
			Name: "secondary controller",
			Id:   openapi.PtrString("6090fd66-6e21-4ef5-87d0-36c7a1b04a80"),
			Controller: &openapi.ApplianceAllOfController{
				Enabled: openapi.PtrBool(true),
			},
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname: "bar.purple",
			},
			Site:      openapi.PtrString("3976e914-ccf4-4704-80e1-18b7de87ff07"),
			Activated: openapi.PtrBool(false),
			Tags: []string{
				"secondary",
				"crap",
			},
			Version: openapi.PtrInt32(15),
		},
		"gateway": {
			Name: "gateway",
			Id:   openapi.PtrString("85fac76b-c526-486d-844a-520a023e76e2"),
			Gateway: &openapi.ApplianceAllOfGateway{
				Enabled: openapi.PtrBool(true),
			},
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname: "tinker.purple",
			},
			Activated: openapi.PtrBool(false),
			Site:      openapi.PtrString("15dd5630-aaf5-4d74-8c75-205e438db9a3"),
			Tags: []string{
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
		name         string
		args         args
		want         []openapi.Appliance
		wantFiltered []openapi.Appliance
		wantErr      bool
	}
	tests := []testStruct{
		{
			name: "invalid regex error",
			args: args{
				appliances: []openapi.Appliance{},
				filter: map[string]map[string]string{
					"include": {
						"name": "*",
					},
				},
				orderBy: []string{"name"},
			},
			wantErr: true,
		},
	}
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
					"include": {
						word: value,
					},
				},
				orderBy: []string{"name"},
			},
			want: []openapi.Appliance{
				mockControllers["primaryController"],
			},
			wantFiltered: []openapi.Appliance{
				mockControllers["gateway"],
				mockControllers["secondaryController"],
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
				orderBy: []string{"name"},
			},
			want: []openapi.Appliance{
				mockControllers["gateway"],
				mockControllers["secondaryController"],
			},
			wantFiltered: []openapi.Appliance{
				mockControllers["primaryController"],
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, filtered, err := FilterAppliances(tt.args.appliances, tt.args.filter, tt.args.orderBy, tt.args.descending)
			if !tt.wantErr && err != nil {
				t.Errorf("FilterAppliances() = %v", err)
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("FilterAppliances() = %v", cmp.Diff(got, tt.want))
			}
			if !cmp.Equal(filtered, tt.wantFiltered) {
				t.Errorf("FilterAppliances() = %v", cmp.Diff(filtered, tt.wantFiltered))
			}
		})
	}
}

var applianceCmpOpts = []cmp.Option{
	cmp.AllowUnexported(openapi.NullableElasticsearch{}),
	cmpopts.IgnoreFields(openapi.Appliance{}, "Controller"),
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
						Name: "connector A no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
					},
					{
						Name: "connector B no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
					},
					{
						Name: "connector C no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
					},
					{
						Name: "connector D no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
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
						Name: "connector two site b",
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
				744237154: {
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
						Name: "connector A no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
					},
					{
						Name: "connector B no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
					},
					{
						Name: "connector C no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
					},
					{
						Name: "connector D no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "c1.devops",
						},
						Site: nil,
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

				2448185940: {
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
						Name: "connector two site b",
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
		{
			name: "gateways with multiple functions enabled",
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
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["A"],
					},
				},
			},
			want: map[int][]openapi.Appliance{
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
					{
						Name: "g3",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						LogForwarder: &openapi.ApplianceAllOfLogForwarder{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g3.devops",
						},
						Site: sites["A"],
					},
				},
			},
		},
		{
			name: "large collective, multiple gateways and sites",
			args: args{
				appliances: []openapi.Appliance{
					{
						Name: "ctrl1",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "ctrl1.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["A"],
					},
					{
						Name: "ctrl2",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "ctrl2.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["B"],
					},
					{
						Name: "ctrl3",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "ctrl3.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["C"],
					},
					{
						Name: "gw1",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw1.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["A"],
					},
					{
						Name: "gw2",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw2.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["A"],
					},
					{
						Name: "gw3",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw3.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["B"],
					},
					{
						Name: "gw4",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw4.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["B"],
					},
					{
						Name: "gw5",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw5.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["C"],
					},
					{
						Name: "gw6",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw6.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["C"],
					},
				},
			},
			want: map[int][]openapi.Appliance{
				hashcode.String("controller=true"): {
					{
						Name: "ctrl1",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "ctrl1.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["A"],
					},
					{
						Name: "ctrl2",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "ctrl2.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["B"],
					},
					{
						Name: "ctrl3",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "ctrl3.devops",
						},
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["C"],
					},
				},
				hashcode.String(fmt.Sprintf("%s%s", *sites["A"], "&gateway=true")): {
					{
						Name: "gw1",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw1.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["A"],
					},
					{
						Name: "gw2",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw2.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["A"],
					},
				},
				hashcode.String(fmt.Sprintf("%s%s", *sites["B"], "&gateway=true")): {
					{
						Name: "gw3",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw3.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["B"],
					},
					{
						Name: "gw4",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw4.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["B"],
					},
				},
				hashcode.String(fmt.Sprintf("%s%s", *sites["C"], "&gateway=true")): {
					{
						Name: "gw5",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw5.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["C"],
					},
					{
						Name: "gw6",
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "gw6.devops",
						},
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						Site: sites["C"],
					},
				},
			},
		},
	}
	opts := []cmp.Option{cmp.AllowUnexported(openapi.NullableElasticsearch{})}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitAppliancesByGroup(tt.args.appliances)
			if !cmp.Equal(got, tt.want, opts...) {
				t.Logf("Got %d groups for %d sites", len(got), len(sites))
				for k, appliances := range got {
					for _, appliance := range appliances {
						t.Logf("[%d] %s - site: %s", k, appliance.GetName(), appliance.GetSite())
					}
				}
				t.Errorf("\n Diff \n %s", cmp.Diff(got, tt.want, opts...))
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
				Portal: &openapi.Portal{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&gateway=false&connector=false&log_forwarder=true&log_server=false&portal=false")),
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
				Portal: &openapi.Portal{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&gateway=false&connector=false&log_forwarder=false&log_server=true&portal=false")),
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
				Portal: &openapi.Portal{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&gateway=true")),
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
				Portal: &openapi.Portal{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&gateway=false&connector=true")),
		},
		{
			name: "portal enabled",
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
				Portal: &openapi.Portal{
					Enabled: openapi.PtrBool(true),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&gateway=false&connector=false&log_forwarder=false&log_server=false&portal=true")),
		},
		{
			name: "gateway and log_forwarder enabled",
			appliance: openapi.Appliance{
				LogForwarder: &openapi.ApplianceAllOfLogForwarder{
					Enabled: openapi.PtrBool(true),
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
				Portal: &openapi.Portal{
					Enabled: openapi.PtrBool(false),
				},
				Controller: &openapi.ApplianceAllOfController{
					Enabled: openapi.PtrBool(false),
				},
				Site: site,
			},
			expect: hashcode.String(fmt.Sprintf("%s%s", *site, "&gateway=true")),
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
				Portal: &openapi.Portal{
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
		{
			name: "multiple controllers and gateways throughout different sites and connectors without site",
			args: args{
				divisor: 5,
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
					744237154: {
						{
							Name: "connector A no site",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "connectora.devops",
							},
							Site: nil,
						},
						{
							Name: "connector Adam no site",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "adam.devops",
							},
							Site: nil,
						},
						{
							Name: "connector Eva no site",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "eva.devops",
							},
							Site: nil,
						},
						{
							Name: "connector B no site",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "connectorb.devops",
							},
							Site: nil,
						},
						{
							Name: "connector C no site",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "connectorc.devops",
							},
							Site: nil,
						},
						{
							Name: "connector D no site",
							Controller: &openapi.ApplianceAllOfController{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "connectord.devops",
							},
							Site: nil,
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
						Name: "connector A no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "connectora.devops",
						},
						Site: nil,
					},
					{
						Name: "connector Eva no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "eva.devops",
						},
						Site: nil,
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
						Name: "connector D no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "connectord.devops",
						},
						Site: nil,
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
					{
						Name: "connector C no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "connectorc.devops",
						},
						Site: nil,
					},
				},
				// index 3
				{

					{
						Name: "connector Adam no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "adam.devops",
						},
						Site: nil,
					},
					{
						Name: "connector B no site",
						Controller: &openapi.ApplianceAllOfController{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "connectorb.devops",
						},
						Site: nil,
					},
				},
			},
		},
		{
			name: "two gateways same site",
			args: args{
				divisor: 2,
				appliances: map[int][]openapi.Appliance{
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
						{
							Name: "g2",
							Gateway: &openapi.ApplianceAllOfGateway{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "g2.devops",
							},
							Site: sites["A"],
						},
					},
				},
			},
			want: [][]openapi.Appliance{
				{
					{
						Name: "g2",
						Gateway: &openapi.ApplianceAllOfGateway{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "g2.devops",
						},
						Site: sites["A"],
					},
				},
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
				},
			},
		},
		{
			name: "two connectors same site",
			args: args{
				divisor: 2,
				appliances: map[int][]openapi.Appliance{
					hashcode.String(fmt.Sprintf("%s%s", *sites["A"], "&gateway=false&connector=true")): {
						{
							Name: "conn1",
							Connector: &openapi.ApplianceAllOfConnector{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "conn1.devops",
							},
							Site: sites["A"],
						},
						{
							Name: "conn2",
							Connector: &openapi.ApplianceAllOfConnector{
								Enabled: openapi.PtrBool(true),
							},
							AdminInterface: &openapi.ApplianceAllOfAdminInterface{
								Hostname: "conn2.devops",
							},
							Site: sites["A"],
						},
					},
				},
			},
			want: [][]openapi.Appliance{
				{
					{
						Name: "conn2",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "conn2.devops",
						},
						Site: sites["A"],
					},
				},
				{
					{
						Name: "conn1",
						Connector: &openapi.ApplianceAllOfConnector{
							Enabled: openapi.PtrBool(true),
						},
						AdminInterface: &openapi.ApplianceAllOfAdminInterface{
							Hostname: "conn1.devops",
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

			if !cmp.Equal(got, tt.want, applianceCmpOpts...) {
				for k, appliances := range got {
					for v, appliance := range appliances {
						t.Logf("[%d/%d] %s - site: %s", k, v, appliance.GetName(), appliance.GetSite())
					}
				}
				t.Errorf("Got diff in\n%s\n", cmp.Diff(tt.want, got, applianceCmpOpts...))
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name          string
		hostname      string
		adminHostName string
		wantErr       bool
		want          regexp.Regexp
	}{
		{
			name:          "valid hostname",
			hostname:      "appgate.test",
			adminHostName: "appgate.test",
			wantErr:       false,
		},
		{
			name:          "not unique hostname",
			hostname:      "play.google.com",
			adminHostName: "play.google.com",
			wantErr:       true,
			want:          *regexp.MustCompile(fmt.Sprintf(`The given hostname %s does not resolve to a unique ip\.`, "play.google.com")),
		},
		{
			name:          "admin interface not hostname",
			hostname:      "controller.devops",
			adminHostName: "appgate.test",
			wantErr:       true,
			want:          *regexp.MustCompile(`Hostname validation failed. Pass the --actual-hostname flag to use the real controller hostname`),
		},
	}

	for _, tt := range tests {
		_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{
			"play.google.com.": {
				A: []string{"9.8.3.2", "9.2.1.5"},
			},
		})
		defer teardown()

		ctrl := openapi.Appliance{
			Id:        openapi.PtrString(uuid.New().String()),
			Name:      "controller",
			Activated: openapi.PtrBool(true),
			Hostname:  tt.hostname,
			AdminInterface: &openapi.ApplianceAllOfAdminInterface{
				Hostname:  tt.adminHostName,
				HttpsPort: openapi.PtrInt32(8443),
			},
			Controller: &openapi.ApplianceAllOfController{
				Enabled: openapi.PtrBool(true),
			},
		}
		err := ValidateHostname(ctrl, tt.adminHostName)
		if err != nil {
			if tt.wantErr {
				if !tt.want.MatchString(err.Error()) {
					t.Fatalf("RES: %s\nEXP: %s", err.Error(), tt.want.String())
				}
				return
			}
			t.Fatalf("WANT: PASS, GOT ERROR: %s", err.Error())
		}
	}
}

func TestStatsIsOnline(t *testing.T) {
	type args struct {
		s openapi.StatsAppliancesListAllOfData
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "pre 6.0 online",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Online: openapi.PtrBool(true),
				},
			},
			want: true,
		},
		{
			name: "online nil value",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Online: nil,
				},
			},
			want: false,
		},
		{
			name: "pre 6.0 offline",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Online: openapi.PtrBool(false),
				},
			},
			want: false,
		},
		{
			name: "status nil",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: nil,
				},
			},
			want: false,
		},
		{
			name: "status offline",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("offline"),
				},
			},
			want: false,
		},
		{
			name: "status healthy",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("healthy"),
				},
			},
			want: true,
		},
		{
			name: "status warning",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("warning"),
				},
			},
			want: true,
		},
		{
			name: "status busy",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("busy"),
				},
			},
			want: true,
		},
		{
			name: "status error",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("error"),
				},
			},
			want: true,
		},
		{
			name: "status not available",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("n/a"),
				},
			},
			want: false,
		},
		{
			name: "status unknown",
			args: args{
				s: openapi.StatsAppliancesListAllOfData{
					Status: openapi.PtrString("abc123"),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatsIsOnline(tt.args.s); got != tt.want {
				t.Errorf("StatsIsOnline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetApplianceVersion(t *testing.T) {
	v61, _ := version.NewVersion("6.1.0")
	type args struct {
		appliance openapi.Appliance
		stats     openapi.StatsAppliancesList
	}
	tests := []struct {
		name    string
		args    args
		want    *version.Version
		wantErr bool
	}{
		{
			name: "online OK",
			args: args{
				appliance: openapi.Appliance{
					Name: "controller one",
					Id:   openapi.PtrString("one"),
				},
				stats: openapi.StatsAppliancesList{
					Data: []openapi.StatsAppliancesListAllOfData{
						{
							Id:      openapi.PtrString("one"),
							Status:  openapi.PtrString("warning"),
							Version: openapi.PtrString(v61.String()),
						},
					},
				},
			},
			want:    v61,
			wantErr: false,
		},
		{
			name: "offline",
			args: args{
				appliance: openapi.Appliance{
					Name: "controller two",
					Id:   openapi.PtrString("two"),
				},
				stats: openapi.StatsAppliancesList{
					Data: []openapi.StatsAppliancesListAllOfData{
						{
							Id: openapi.PtrString("two"),
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetApplianceVersion(tt.args.appliance, tt.args.stats)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetApplianceVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetApplianceVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_orderAppliances(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
		orderBy    []string
		descending bool
	}

	// Generate appliances
	app1, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller1", "primary.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, "Default")
	app2, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller2", "secondary.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, "Default")
	app3, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller3", "backup1.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, "Default")
	app4, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller4", "backup2.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, "Default")
	app5, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller5", "balance1.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, "Default")

	// Modify appliances for tests
	app3.SetActivated(false)
	app5.SetActivated(false)

	tests := []struct {
		name    string
		args    args
		want    []openapi.Appliance
		wantErr bool
	}{
		{
			name: "order by name",
			args: args{
				orderBy:    []string{"name"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app1, app2, app3, app4, app5},
		},
		{
			name: "order by name descending",
			args: args{
				descending: true,
				orderBy:    []string{"name"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app5, app4, app3, app2, app1},
		},
		{
			name: "order by activated",
			args: args{
				orderBy:    []string{"activated"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app1, app2, app4, app3, app5},
		},
		{
			name: "order by activated mixed casing",
			args: args{
				orderBy:    []string{"AcTivated"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app1, app2, app4, app3, app5},
		},
		{
			name: "order by activated desc",
			args: args{
				descending: true,
				orderBy:    []string{"activated"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app5, app3, app4, app2, app1},
		},
		{
			name: "order by activated and hostname",
			args: args{
				orderBy:    []string{"activated", "hostname"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app4, app1, app2, app3, app5},
		},
		{
			name: "order by activated and hostname descending",
			args: args{
				descending: true,
				orderBy:    []string{"activated", "hostname"},
				appliances: []openapi.Appliance{app1, app2, app3, app4, app5},
			},
			want: []openapi.Appliance{app5, app3, app2, app1, app4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orderAppliances(tt.args.appliances, tt.args.orderBy, tt.args.descending)
			if err != nil && !tt.wantErr {
				t.Fatalf("FAIL orderAppliances() - %v", err)
			}
			assert.Equal(t, got, tt.want)
		})
	}
}

func GenerateApplianceWithStats(activeFunctions []string, name, hostname, currentVersion, targetVersion, status, upgradeStatus, site string) (openapi.Appliance, openapi.StatsAppliancesListAllOfData) {
	id := uuid.NewString()
	now := time.Now()
	ctrl := &openapi.ApplianceAllOfController{}
	ls := &openapi.ApplianceAllOfLogServer{}
	gw := &openapi.ApplianceAllOfGateway{}
	lf := &openapi.ApplianceAllOfLogForwarder{}
	con := &openapi.ApplianceAllOfConnector{}
	portal := &openapi.Portal{}

	for _, f := range activeFunctions {
		switch f {
		case FunctionController:
			ctrl.SetEnabled(true)
		case FunctionGateway:
			gw.SetEnabled(true)
		case FunctionLogServer:
			ls.SetEnabled(true)
		case FunctionLogForwarder:
			lf.SetEnabled(true)
		case FunctionPortal:
			portal.SetEnabled(true)
		case FunctionConnector:
			con.SetEnabled(true)
		}
	}

	app := openapi.Appliance{
		Id:                        openapi.PtrString(id),
		Name:                      name,
		Notes:                     nil,
		Created:                   openapi.PtrTime(now),
		Updated:                   openapi.PtrTime(now),
		Tags:                      []string{},
		Activated:                 openapi.PtrBool(true),
		PendingCertificateRenewal: openapi.PtrBool(false),
		Version:                   openapi.PtrInt32(18),
		Hostname:                  hostname,
		Site:                      openapi.PtrString(site),
		SiteName:                  new(string),
		Customization:             new(string),
		ClientInterface:           openapi.ApplianceAllOfClientInterface{},
		AdminInterface: &openapi.ApplianceAllOfAdminInterface{
			Hostname:  hostname,
			HttpsPort: openapi.PtrInt32(8443),
		},
		Networking:          openapi.ApplianceAllOfNetworking{},
		Ntp:                 &openapi.ApplianceAllOfNtp{},
		SshServer:           &openapi.ApplianceAllOfSshServer{},
		SnmpServer:          &openapi.ApplianceAllOfSnmpServer{},
		HealthcheckServer:   &openapi.ApplianceAllOfHealthcheckServer{},
		PrometheusExporter:  &openapi.PrometheusExporter{},
		Ping:                &openapi.ApplianceAllOfPing{},
		LogServer:           ls,
		Controller:          ctrl,
		Gateway:             gw,
		LogForwarder:        lf,
		Connector:           con,
		Portal:              portal,
		RsyslogDestinations: []openapi.ApplianceAllOfRsyslogDestinations{},
		HostnameAliases:     []string{},
	}
	appstatdata := *openapi.NewStatsAppliancesListAllOfDataWithDefaults()
	appstatdata.SetId(app.GetId())
	appstatdata.SetStatus(status)
	appstatdata.SetVersion(currentVersion)
	appstatdata.SetUpgrade(openapi.StatsAppliancesListAllOfUpgrade{
		Status:  &upgradeStatus,
		Details: openapi.PtrString(targetVersion),
	})
	return app, appstatdata
}
