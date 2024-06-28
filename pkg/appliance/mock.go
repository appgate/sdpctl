package appliance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v20/openapi"
	"github.com/appgate/sdpctl/pkg/appliance/backup"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/uuid"
)

const (
	TestAppliancePrimary               = "primary"
	TestApplianceUnpreparedPrimary     = "primary-unprepared"
	TestApplianceSecondary             = "secondary"
	TestApplianceController3           = "controller3"
	TestApplianceControllerOffline     = "controller4"
	TestApplianceControllerNotPrepared = "controller5"
	TestApplianceControllerMismatch    = "controller6"
	TestApplianceGatewayA1             = "gatewayA1"
	TestApplianceGatewayA2             = "gatewayA2"
	TestApplianceGatewayA3             = "gatewayA3"
	TestApplianceGatewayB1             = "gatewayB1"
	TestApplianceGatewayB2             = "gatewayB2"
	TestApplianceGatewayB3             = "gatewayB3"
	TestApplianceGatewayC1             = "gatewayC1"
	TestApplianceGatewayC2             = "gatewayC2"
	TestApplianceLogForwarderA1        = "logforwarderA1"
	TestApplianceLogForwarderA2        = "logforwarderA2"
	TestAppliancePortalA1              = "portalA1"
	TestApplianceConnectorA1           = "connectorA1"
	TestApplianceLogServer             = "logserver"
)

var (
	PreSetApplianceNames                              = []string{TestAppliancePrimary, TestApplianceSecondary, TestApplianceController3, TestApplianceControllerOffline, TestApplianceControllerNotPrepared, TestApplianceControllerMismatch, TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3, TestApplianceGatewayB1, TestApplianceGatewayB2, TestApplianceGatewayB3, TestApplianceGatewayC1, TestApplianceGatewayC2, TestApplianceLogForwarderA1, TestApplianceLogForwarderA2, TestAppliancePortalA1, TestApplianceConnectorA1, TestApplianceLogServer}
	InitialTestStats     *openapi.StatsAppliancesList = openapi.NewStatsAppliancesListWithDefaults()
	UpgradedTestStats    *openapi.StatsAppliancesList = openapi.NewStatsAppliancesListWithDefaults()
)

type CollectiveTestStruct struct {
	Appliances    map[string]openapi.Appliance
	Stats         *openapi.StatsAppliancesList
	UpgradedStats *openapi.StatsAppliancesList
}

func GenerateCollective(t *testing.T, hostname, from, to string, appliances []string) *CollectiveTestStruct {
	t.Helper()
	defer t.Cleanup(func() {
		InitialTestStats = openapi.NewStatsAppliancesListWithDefaults()
		UpgradedTestStats = openapi.NewStatsAppliancesListWithDefaults()
	})
	res := CollectiveTestStruct{
		Stats:         InitialTestStats,
		UpgradedStats: UpgradedTestStats,
		Appliances:    map[string]openapi.Appliance{},
	}

	siteA := uuid.NewString()
	siteNameA := "SiteA"
	siteB := uuid.NewString()
	siteNameB := "SiteB"
	siteC := uuid.NewString()
	siteNameC := "SiteC"

	for _, n := range appliances {
		switch n {
		case TestAppliancePrimary:
			res.addAppliance(n, hostname, siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionController})
		case TestApplianceUnpreparedPrimary:
			res.addAppliance(n, hostname, siteA, siteNameA, from, from, statusHealthy, UpgradeStatusIdle, true, []string{FunctionController})
		case TestApplianceSecondary:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionController})
		case TestApplianceController3:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusIdle, true, []string{FunctionController})
		case TestApplianceControllerOffline:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusOffline, UpgradeStatusIdle, false, []string{FunctionController})
		case TestApplianceControllerNotPrepared:
			res.addAppliance(n, "", siteA, siteNameA, from, from, statusHealthy, UpgradeStatusIdle, true, []string{FunctionController})
		case TestApplianceControllerMismatch:
			res.addAppliance(n, "", siteA, siteNameA, "6.1", from, statusHealthy, UpgradeStatusReady, true, []string{FunctionController})
		case TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionGateway})
		case TestApplianceGatewayB1, TestApplianceGatewayB2, TestApplianceGatewayB3:
			res.addAppliance(n, "", siteB, siteNameB, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionGateway})
		case TestApplianceGatewayC1, TestApplianceGatewayC2:
			res.addAppliance(n, "", siteC, siteNameC, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionGateway})
		case TestApplianceLogForwarderA1, TestApplianceLogForwarderA2:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionLogForwarder})
		case TestAppliancePortalA1:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionPortal})
		case TestApplianceConnectorA1:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionPortal})
		case TestApplianceLogServer:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, true, []string{FunctionLogServer})
		default:
		}
	}

	return &res
}

func (cts *CollectiveTestStruct) addAppliance(name, hostname, site, siteName, fromVersion, toVersion, status, upgradeStatus string, online bool, functions []string) {
	a, s, u := GenerateApplianceWithStats(functions, name, hostname, fromVersion, toVersion, status, upgradeStatus, online, site, siteName)
	for _, f := range functions {
		switch f {
		case FunctionController:
			count := cts.Stats.GetControllerCount() + 1
			cts.Stats.SetControllerCount(count)
			cts.UpgradedStats.SetControllerCount(count)
		case FunctionGateway:
			count := cts.Stats.GetGatewayCount() + 1
			cts.Stats.SetGatewayCount(count)
			cts.UpgradedStats.SetGatewayCount(count)
		case FunctionLogServer:
			count := cts.Stats.GetLogServerCount() + 1
			cts.Stats.SetLogServerCount(count)
			cts.UpgradedStats.SetLogServerCount(count)
		case FunctionLogForwarder:
			count := cts.Stats.GetLogForwarderCount() + 1
			cts.Stats.SetLogForwarderCount(count)
			cts.UpgradedStats.SetLogForwarderCount(count)
		case FunctionPortal:
			count := cts.Stats.GetPortalCount() + 1
			cts.Stats.SetPortalCount(count)
			cts.UpgradedStats.SetPortalCount(count)
		case FunctionConnector:
			count := cts.Stats.GetConnectorCount() + 1
			cts.Stats.SetConnectorCount(count)
			cts.UpgradedStats.SetConnectorCount(count)
		}
	}
	cts.Appliances[a.GetName()] = a
	cts.Stats.Data = append(cts.Stats.Data, s)
	cts.UpgradedStats.Data = append(cts.UpgradedStats.Data, u)
}

var (
	changeRequestResponder = func(w http.ResponseWriter, r *http.Request) {
		changeID := uuid.NewString()
		body := fmt.Sprintf(`{"id": "%s" }`, changeID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(body))
	}
	DefaultResponder = func(callback func(rw http.ResponseWriter, req *http.Request, count int)) http.HandlerFunc {
		count := 0
		return func(w http.ResponseWriter, r *http.Request) {
			callback(w, r, count)
		}
	}
	mutatingResponder = func(callback func(count int) ([]byte, error)) http.HandlerFunc {
		count := 0
		return func(w http.ResponseWriter, r *http.Request) {
			mutated, err := callback(count)
			if err != nil {
				panic(fmt.Sprintf("Internal testing error; request mutation failed %q", err))
			}
			count++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(mutated))
		}
	}
)

func (cts *CollectiveTestStruct) GetAppliances() []openapi.Appliance {
	a := make([]openapi.Appliance, 0, len(cts.Appliances))
	for _, app := range cts.Appliances {
		a = append(a, app)
	}
	return a
}

func (cts *CollectiveTestStruct) GenerateStubs(appliances []openapi.Appliance, stats, upgradedStats openapi.StatsAppliancesList) []httpmock.Stub {
	stubs := []httpmock.Stub{}

	// appliance list applianceListStub
	applianceListStub := httpmock.Stub{
		URL: "/admin/appliances",
		Responder: func(w http.ResponseWriter, r *http.Request) {
			l := openapi.ApplianceList{}
			count := len(appliances)
			l.TotalCount = openapi.PtrInt32(int32(count))
			l.Data = append(l.Data, appliances...)
			b, err := json.Marshal(l)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		},
	}
	stubs = append(stubs, applianceListStub)

	// appliance stats stub
	statsStub := httpmock.Stub{
		URL: "/admin/stats/appliances",
		Responder: mutatingResponder(func(count int) ([]byte, error) {
			if count > 0 {
				return json.Marshal(upgradedStats)
			}
			return json.Marshal(stats)
		}),
	}
	stubs = append(stubs, statsStub)

	stubs = append(stubs, httpmock.Stub{
		URL: "/admin/appliances/{appliance}/upgrade",
		Responder: DefaultResponder(func(rw http.ResponseWriter, req *http.Request, count int) {
			id := req.PathValue("appliance")
			for _, s := range stats.GetData() {
				if s.GetId() != id {
					continue
				}
				us := s.GetUpgrade()
				if count <= 0 {
					us.SetStatus(UpgradeStatusReady)
				} else if count == 1 {
					us.SetStatus(UpgradeStatusIdle)
					us.SetDetails("")
				}
				body, err := us.MarshalJSON()
				if err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				rw.Header().Set("Content-Type", "application/json")
				rw.Write(body)
			}
		}),
	})

	// global settings stub
	globalSettingsStub := httpmock.Stub{
		URL: "/admin/global-settings",
		Responder: func(w http.ResponseWriter, r *http.Request) {
			s := openapi.NewGlobalSettingsWithDefaults()
			s.SetBackupApiEnabled(true)
			s.SetBackupPassphrase("admin")
			b, err := json.Marshal(s)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			w.Write(b)
		},
	}
	stubs = append(stubs, globalSettingsStub)

	// appliance id upgrade status, complete and maintenance stubs
	for _, a := range appliances {
		stubs = append(stubs, httpmock.Stub{
			URL:       fmt.Sprintf("/admin/appliances/%s/upgrade/complete", a.GetId()),
			Responder: changeRequestResponder,
		})
		stubs = append(stubs, httpmock.Stub{
			URL:       fmt.Sprintf("/admin/appliances/%s/maintenance", a.GetId()),
			Responder: changeRequestResponder,
		})

		backupID := uuid.NewString()
		stubs = append(stubs, httpmock.Stub{
			URL: fmt.Sprintf("/admin/appliances/%s/backup", a.GetId()),
			Responder: func(w http.ResponseWriter, r *http.Request) {
				res := openapi.NewAppliancesIdBackupPost200ResponseWithDefaults()
				res.SetId(backupID)
				b, err := json.Marshal(res)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				w.Write(b)
			},
		})
		stubs = append(stubs, httpmock.Stub{
			URL: fmt.Sprintf("/admin/appliances/%s/backup/%s/status", a.GetId(), backupID),
			Responder: func(w http.ResponseWriter, r *http.Request) {
				res := openapi.NewAppliancesIdBackupBackupIdStatusGet200ResponseWithDefaults()
				res.SetStatus(backup.Done)
				res.SetResult(backup.Success)
				b, err := json.Marshal(res)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				w.Write(b)
			},
		})
		stubs = append(stubs, httpmock.Stub{
			URL:       fmt.Sprintf("/admin/appliances/%s/backup/%s", a.GetId(), backupID),
			Responder: httpmock.FileResponse(),
		})

		stubs = append(stubs, httpmock.Stub{
			URL: fmt.Sprintf("/admin/appliances/%s/name-resolution-status", a.GetId()),
			Responder: func(w http.ResponseWriter, r *http.Request) {
				res := openapi.NewAppliancesIdNameResolutionStatusGet200ResponseWithDefaults()
				res.Resolutions = &map[string]openapi.AppliancesIdNameResolutionStatusGet200ResponseResolutionsValue{
					"aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev": {
						Partial:  openapi.PtrBool(false),
						Finals:   []string{"3.120.51.78", "35.156.237.184"},
						Partials: []string{"dns://all.GW-ELB-2001535196.eu-central-1.elb.amazonaws.com", "dns://all.purple-lb-1785267452.eu-central-1.elb.amazonaws.com"},
						Errors:   []string{},
					},
				}
				b, err := res.MarshalJSON()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				w.Write(b)
			},
		})
	}

	return stubs
}

func GenerateApplianceWithStats(activeFunctions []string, name, hostname, currentVersion, targetVersion, status, upgradeStatus string, online bool, site, siteName string) (openapi.Appliance, openapi.StatsAppliancesListAllOfData, openapi.StatsAppliancesListAllOfData) {
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
		SiteName:                  openapi.PtrString(siteName),
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
	currentStatsData := *openapi.NewStatsAppliancesListAllOfDataWithDefaults()
	currentStatsData.SetId(app.GetId())
	currentStatsData.SetName(app.GetName())
	currentStatsData.SetSiteName(siteName)
	currentStatsData.SetStatus(status)
	currentStatsData.SetVersion(currentVersion)
	currentStatsData.SetOnline(online)
	currentStatsData.SetVolumeNumber(0)
	currentStatsData.SetUpgrade(openapi.StatsAppliancesListAllOfUpgrade{
		Status:  &upgradeStatus,
		Details: openapi.PtrString(targetVersion),
	})

	upgradedStatsData := *openapi.NewStatsAppliancesListAllOfDataWithDefaults()
	upgradedStatsData.SetId(app.GetId())
	upgradedStatsData.SetName(app.GetName())
	upgradedStatsData.SetSiteName(siteName)
	upgradedStatsData.SetStatus(status)
	upgradedStatsData.SetOnline(online)
	upgradedStatsData.SetVersion(targetVersion)
	upgradedStatsData.SetVolumeNumber(1)
	upgradedStatsData.SetUpgrade(openapi.StatsAppliancesListAllOfUpgrade{
		Status: openapi.PtrString(UpgradeStatusIdle),
	})

	return app, currentStatsData, upgradedStatsData
}
