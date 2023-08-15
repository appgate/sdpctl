package dns

import (
	"log"
	"net"
	"os"
	"strconv"

	"github.com/appgate/sdpctl/pkg/util"
	"github.com/foxcpp/go-mockdns"
)

func mergeDNSRecords(maps ...map[string]mockdns.Zone) map[string]mockdns.Zone {
	result := make(map[string]mockdns.Zone)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

var defaultRecords = map[string]mockdns.Zone{
	"appgate.test.": {
		A: []string{"1.2.3.4"},
	},
}

type discardLog struct{}

func (l discardLog) Printf(f string, args ...interface{}) {}

func RunMockDNSServer(dnsRecords map[string]mockdns.Zone) (string, func()) {
	var logger mockdns.Logger
	logger = discardLog{}
	if v, err := strconv.ParseBool(util.Getenv("DEBUG", "false")); v && err == nil {
		logger = log.New(os.Stderr, "mockdns server: ", log.LstdFlags)
	}
	srv, err := mockdns.NewServerWithLogger(mergeDNSRecords(defaultRecords, dnsRecords), logger, false)
	if err != nil {
		panic("test panic; cant run dns mock server")
	}
	runningMockServerAddress := srv.LocalAddr().String()
	// defer srv.Close()

	srv.PatchNet(net.DefaultResolver)
	// defer mockdns.UnpatchNet(net.DefaultResolver)

	return runningMockServerAddress, func() {
		srv.Close()
		mockdns.UnpatchNet(net.DefaultResolver)
	}
}
