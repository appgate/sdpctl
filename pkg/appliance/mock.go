package appliance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	"github.com/appgate/sdpctl/pkg/appliance/backup"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/google/uuid"
)

const (
	TestAppliancePrimary                  = "primary"
	TestApplianceUnpreparedPrimary        = "primary-unprepared"
	TestApplianceSecondary                = "secondary"
	TestApplianceController3              = "controller3"
	TestApplianceControllerOffline        = "controller4"
	TestApplianceControllerNotPrepared    = "controller5"
	TestApplianceController2NotPrepared   = "controller7"
	TestApplianceControllerMismatch       = "controller6"
	TestApplianceControllerGatewayPrimary = "controller-gateway-primary"
	TestApplianceGatewayA1                = "gatewayA1"
	TestApplianceGatewayA2                = "gatewayA2"
	TestApplianceGatewayA3                = "gatewayA3"
	TestApplianceGatewayB1                = "gatewayB1"
	TestApplianceGatewayB2                = "gatewayB2"
	TestApplianceGatewayB3                = "gatewayB3"
	TestApplianceGatewayC1                = "gatewayC1"
	TestApplianceGatewayC2                = "gatewayC2"
	TestApplianceLogForwarderA1           = "logforwarderA1"
	TestApplianceLogForwarderA2           = "logforwarderA2"
	TestApplianceLogForwarderB1           = "logforwarderB1"
	TestApplianceLogForwarderB2           = "logforwarderB2"
	TestApplianceLogForwarderC1           = "logforwarderC1"
	TestApplianceLogForwarderC2           = "logforwarderC2"
	TestAppliancePortalA1                 = "portalA1"
	TestApplianceConnectorA1              = "connectorA1"
	TestApplianceLogServer                = "logserver"
	TestApplianceControllerGatewayA1      = "controller-gatewayA1"
	TestApplianceControllerGatewayB1      = "controller-gatewayB1"

	TestSiteA = "SiteA"
	TestSiteB = "SiteB"
	TestSiteC = "SiteC"
)

var (
	PreSetApplianceNames                                  = []string{TestAppliancePrimary, TestApplianceSecondary, TestApplianceController3, TestApplianceControllerOffline, TestApplianceControllerNotPrepared, TestApplianceController2NotPrepared, TestApplianceControllerMismatch, TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3, TestApplianceGatewayB1, TestApplianceGatewayB2, TestApplianceGatewayB3, TestApplianceGatewayC1, TestApplianceGatewayC2, TestApplianceLogForwarderA1, TestApplianceLogForwarderA2, TestAppliancePortalA1, TestApplianceConnectorA1, TestApplianceLogServer, TestApplianceControllerGatewayA1, TestApplianceControllerGatewayB1, TestApplianceControllerGatewayPrimary}
	InitialTestStats     *openapi.ApplianceWithStatusList = openapi.NewApplianceWithStatusListWithDefaults()
	UpgradedTestStats    *openapi.ApplianceWithStatusList = openapi.NewApplianceWithStatusListWithDefaults()
)

type CollectiveTestStruct struct {
	Appliances    map[string]openapi.Appliance
	Stats         *openapi.ApplianceWithStatusList
	UpgradedStats *openapi.ApplianceWithStatusList
}

func GenerateCollective(t *testing.T, hostname, from, to string, appliances []string) *CollectiveTestStruct {
	t.Helper()
	defer t.Cleanup(func() {
		InitialTestStats = openapi.NewApplianceWithStatusListWithDefaults()
		UpgradedTestStats = openapi.NewApplianceWithStatusListWithDefaults()
	})
	res := CollectiveTestStruct{
		Stats:         InitialTestStats,
		UpgradedStats: UpgradedTestStats,
		Appliances:    map[string]openapi.Appliance{},
	}

	siteA := uuid.NewString()
	siteNameA := TestSiteA
	siteB := uuid.NewString()
	siteNameB := TestSiteB
	siteC := uuid.NewString()
	siteNameC := TestSiteC

	for _, n := range appliances {
		switch n {
		case TestAppliancePrimary:
			res.addAppliance(n, hostname, siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionController})
		case TestApplianceUnpreparedPrimary:
			res.addAppliance(n, hostname, siteA, siteNameA, from, from, statusHealthy, UpgradeStatusIdle, []string{FunctionController})
		case TestApplianceSecondary:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionController})
		case TestApplianceController3:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusIdle, []string{FunctionController})
		case TestApplianceControllerOffline:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusOffline, UpgradeStatusIdle, []string{FunctionController})
		case TestApplianceControllerNotPrepared, TestApplianceController2NotPrepared:
			res.addAppliance(n, "", siteA, siteNameA, from, from, statusHealthy, UpgradeStatusIdle, []string{FunctionController})
		case TestApplianceControllerMismatch:
			res.addAppliance(n, "", siteA, siteNameA, "6.1", from, statusHealthy, UpgradeStatusReady, []string{FunctionController})
		case TestApplianceControllerGatewayPrimary:
			res.addAppliance(n, hostname, siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionController, FunctionGateway})
		case TestApplianceGatewayA1, TestApplianceGatewayA2, TestApplianceGatewayA3:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionGateway})
		case TestApplianceGatewayB1, TestApplianceGatewayB2, TestApplianceGatewayB3:
			res.addAppliance(n, "", siteB, siteNameB, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionGateway})
		case TestApplianceGatewayC1, TestApplianceGatewayC2:
			res.addAppliance(n, "", siteC, siteNameC, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionGateway})
		case TestApplianceLogForwarderA1, TestApplianceLogForwarderA2, TestApplianceLogForwarderB1, TestApplianceLogForwarderB2, TestApplianceLogForwarderC1, TestApplianceLogForwarderC2:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionLogForwarder})
		case TestAppliancePortalA1:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionPortal})
		case TestApplianceConnectorA1:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionPortal})
		case TestApplianceLogServer:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionLogServer})
		case TestApplianceControllerGatewayA1:
			res.addAppliance(n, "", siteA, siteNameA, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionController, FunctionGateway})
		case TestApplianceControllerGatewayB1:
			res.addAppliance(n, "", siteB, siteNameB, from, to, statusHealthy, UpgradeStatusReady, []string{FunctionController, FunctionGateway})
		default:
		}
	}

	return &res
}

func (cts *CollectiveTestStruct) addAppliance(name, hostname, site, siteName, fromVersion, toVersion, status, upgradeStatus string, functions []string) {
	a, s, u := GenerateApplianceWithStats(functions, name, hostname, fromVersion, toVersion, status, upgradeStatus, site, siteName)
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

func (cts *CollectiveTestStruct) GetAppliance(name string) *openapi.Appliance {
	for _, a := range cts.Appliances {
		if a.GetName() == name {
			return &a
		}
	}
	return nil
}

func (cts *CollectiveTestStruct) GetAppliances() []openapi.Appliance {
	a := make([]openapi.Appliance, 0, len(cts.Appliances))
	for _, app := range cts.Appliances {
		a = append(a, app)
	}
	return a
}

func (cts *CollectiveTestStruct) GetUpgradeStatusMap() map[string]UpgradeStatusResult {
	upgradeStatusMap := map[string]UpgradeStatusResult{}
	for _, a := range cts.GetAppliances() {
		for _, s := range cts.Stats.GetData() {
			if a.GetId() != s.GetId() {
				continue
			}
			us := s.GetDetails().Upgrade
			upgradeStatusMap[a.GetId()] = UpgradeStatusResult{
				Name:    a.GetName(),
				Status:  us.GetStatus(),
				Details: us.GetDetails(),
			}
		}
	}
	return upgradeStatusMap
}

func (cts *CollectiveTestStruct) GenerateStubs(appliances []openapi.Appliance, stats, upgradedStats openapi.ApplianceWithStatusList) []httpmock.Stub {
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
		URL: "/admin/appliances/status",
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
				us := s.GetDetails().Upgrade
				if us.GetStatus() != UpgradeStatusIdle && count <= 0 {
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
				res := openapi.NewAppliancesIdBackupPost202ResponseWithDefaults()
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
						Errors: []string{},
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

func GenerateApplianceWithStats(activeFunctions []string, name, hostname, currentVersion, targetVersion, status, upgradeStatus string, site, siteName string) (openapi.Appliance, openapi.ApplianceWithStatus, openapi.ApplianceWithStatus) {
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
	currentStatsData := *openapi.NewApplianceWithStatusWithDefaults()
	currentStatsData.SetId(app.GetId())
	currentStatsData.SetName(app.GetName())
	currentStatsData.SetSiteName(siteName)
	currentStatsData.SetStatus(status)
	currentStatsData.SetApplianceVersion(currentVersion)
	currentStatsData.Details = openapi.NewApplianceWithStatusAllOfDetails()
	currentStatsData.Details.SetVolumeNumber(0)
	currentStatsData.Details.SetUpgrade(openapi.ApplianceWithStatusAllOfDetailsUpgrade{
		Status:  openapi.PtrString(upgradeStatus),
		Details: openapi.PtrString(targetVersion),
	})

	upgradedStatsData := *openapi.NewApplianceWithStatusWithDefaults()
	upgradedStatsData.SetId(app.GetId())
	upgradedStatsData.SetName(app.GetName())
	upgradedStatsData.SetSiteName(siteName)
	upgradedStatsData.SetStatus(status)
	upgradedStatsData.SetApplianceVersion(targetVersion)
	upgradedStatsData.Details = openapi.NewApplianceWithStatusAllOfDetails()
	upgradedStatsData.Details.SetVolumeNumber(1)
	upgradedStatsData.Details.SetUpgrade(openapi.ApplianceWithStatusAllOfDetailsUpgrade{
		Status: openapi.PtrString(UpgradeStatusIdle),
	})

	return app, currentStatsData, upgradedStatsData
}
