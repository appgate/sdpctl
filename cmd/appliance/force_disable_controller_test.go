package appliance

import (
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_printSummary(t *testing.T) {
	app1, app1data := generateApplianceWithStats("appliance1", "appliance1.example.com", "6.1.1-12345", "healthy")
	app2, app2data := generateApplianceWithStats("appliance2", "appliance2.example.com", "6.1.1-12345", "healthy")
	app3, app3data := generateApplianceWithStats("appliance3", "appliance3.example.com", "unknown", "offline")
	stats := openapi.NewStatsAppliancesListAllOf()
	stats.Data = append(stats.Data, app1data, app2data, app3data)
	type args struct {
		stats               []openapi.StatsAppliancesListAllOfData
		primaryControllerID string
		disable             []openapi.Appliance
		offline             []openapi.Appliance
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "disable one controller",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app1},
				offline: []openapi.Appliance{},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance1    appliance1.example.com    healthy    6.1.1-12345

`,
		},
		{
			name: "disable two controllers",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app1, app2},
				offline: []openapi.Appliance{},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance1    appliance1.example.com    healthy    6.1.1-12345
appliance2    appliance2.example.com    healthy    6.1.1-12345

`,
		},
		{
			name: "disable two controllers, one offline",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app1, app2},
				offline: []openapi.Appliance{app3},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance1    appliance1.example.com    healthy    6.1.1-12345
appliance2    appliance2.example.com    healthy    6.1.1-12345


WARNING:
The following Controllers are unreachable and will likely not recieve the announcement. Please confirm that these controllers are, in fact, offline before continuing:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance3    appliance3.example.com    offline    unknown

`,
		},
		{
			name: "disable offline controller",
			args: args{
				stats:   stats.GetData(),
				disable: []openapi.Appliance{app3},
				offline: []openapi.Appliance{},
			},
			want: `
FORCE-DISABLE-CONTROLLER SUMMARY

This will force disable the selected controllers and announce it to the remaining controllers. The following Controllers are going to be disabled:

Name          Hostname                  Status     Version
----          --------                  ------     -------
appliance3    appliance3.example.com    offline    unknown

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := printSummary(tt.args.stats, tt.args.primaryControllerID, tt.args.disable, tt.args.offline)
			if (err != nil) != tt.wantErr {
				t.Errorf("printSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func generateApplianceWithStats(name, hostname, version, status string) (openapi.Appliance, openapi.StatsAppliancesListAllOfData) {
	app := openapi.NewApplianceWithDefaults()
	id := uuid.NewString()
	app.SetId(id)
	app.SetName(name)
	app.SetHostname(hostname)
	appstatdata := *openapi.NewStatsAppliancesListAllOfDataWithDefaults()
	appstatdata.SetId(app.GetId())
	appstatdata.SetStatus(status)
	appstatdata.SetVersion(version)
	return *app, appstatdata
}
