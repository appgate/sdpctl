package appliance

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v21/openapi"
	"github.com/appgate/sdpctl/pkg/dns"
	"github.com/foxcpp/go-mockdns"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestFilterAvailable(t *testing.T) {
	type args struct {
		appliances []openapi.Appliance
		stats      []openapi.ApplianceWithStatus
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
				stats: []openapi.ApplianceWithStatus{
					{
						Id:     openapi.PtrString("one"),
						Status: openapi.PtrString("healthy"),
					},
					{
						Id:     openapi.PtrString("two"),
						Status: openapi.PtrString("offline"),
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
				stats: []openapi.ApplianceWithStatus{
					{
						Id:     openapi.PtrString("one"),
						Status: openapi.PtrString("healthy"),
					},
					{
						Id:     openapi.PtrString("two"),
						Status: openapi.PtrString("offline"),
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
		s openapi.ApplianceWithStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "status nil",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: nil,
				},
			},
			want: false,
		},
		{
			name: "status offline",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: openapi.PtrString("offline"),
				},
			},
			want: false,
		},
		{
			name: "status healthy",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: openapi.PtrString("healthy"),
				},
			},
			want: true,
		},
		{
			name: "status warning",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: openapi.PtrString("warning"),
				},
			},
			want: true,
		},
		{
			name: "status busy",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: openapi.PtrString("busy"),
				},
			},
			want: true,
		},
		{
			name: "status error",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: openapi.PtrString("error"),
				},
			},
			want: true,
		},
		{
			name: "status not available",
			args: args{
				s: openapi.ApplianceWithStatus{
					Status: openapi.PtrString("n/a"),
				},
			},
			want: false,
		},
		{
			name: "status unknown",
			args: args{
				s: openapi.ApplianceWithStatus{
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
		stats     openapi.ApplianceWithStatusList
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
				stats: openapi.ApplianceWithStatusList{
					Data: []openapi.ApplianceWithStatus{
						{
							Id:               openapi.PtrString("one"),
							Status:           openapi.PtrString("warning"),
							ApplianceVersion: openapi.PtrString(v61.String()),
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
				stats: openapi.ApplianceWithStatusList{
					Data: []openapi.ApplianceWithStatus{
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
	app1, _, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller1", "primary.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, true, "Default", "default")
	app2, _, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller2", "secondary.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, true, "Default", "default")
	app3, _, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller3", "backup1.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, true, "Default", "default")
	app4, _, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller4", "backup2.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, true, "Default", "default")
	app5, _, _ := GenerateApplianceWithStats([]string{FunctionController}, "controller5", "balance1.appgate.com", "6.1.1-12345", "6.2.1-12345", "healthy", UpgradeStatusReady, true, "Default", "default")

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
