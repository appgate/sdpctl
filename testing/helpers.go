package testing

import (
	"time"

	"github.com/appgate/sdp-api-client-go/api/v18/openapi"
	"github.com/google/uuid"
)

const (
	FunctionController = iota
	FunctionGateway
	FunctionLogForwarder
	FunctionLogServer
	FunctionPortal
	FunctionConnector
)

func GenerateApplianceWithStats(activeFunctions []int, name, hostname, version, status string) (openapi.Appliance, openapi.StatsAppliancesListAllOfData) {
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
		Id:                                   openapi.PtrString(id),
		Name:                                 name,
		Notes:                                nil,
		Created:                              openapi.PtrTime(now),
		Updated:                              openapi.PtrTime(now),
		Tags:                                 []string{},
		ConnectToPeersUsingClientPortWithSpa: nil,
		PeerInterface:                        &openapi.ApplianceAllOfPeerInterface{},
		Activated:                            openapi.PtrBool(true),
		PendingCertificateRenewal:            openapi.PtrBool(false),
		Version:                              openapi.PtrInt32(18),
		Hostname:                             hostname,
		Site:                                 openapi.PtrString("Default Site"),
		SiteName:                             new(string),
		Customization:                        new(string),
		ClientInterface:                      openapi.ApplianceAllOfClientInterface{},
		AdminInterface: &openapi.ApplianceAllOfAdminInterface{
			Hostname:  hostname,
			HttpsPort: openapi.PtrInt32(8443),
		},
		Networking:          openapi.ApplianceAllOfNetworking{},
		Ntp:                 &openapi.ApplianceAllOfNtp{},
		SshServer:           &openapi.ApplianceAllOfSshServer{},
		SnmpServer:          &openapi.ApplianceAllOfSnmpServer{},
		HealthcheckServer:   &openapi.ApplianceAllOfHealthcheckServer{},
		PrometheusExporter:  &openapi.ApplianceAllOfPrometheusExporter{},
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
	appstatdata.SetVersion(version)
	return app, appstatdata
}
