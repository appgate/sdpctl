package network

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
)

type ResolverError struct {
	ip       []net.IP
	hostname string
}

func (r *ResolverError) Error() string {
	ips := make([]string, 0, len(r.ip))
	for _, addr := range r.ip {
		ips = append(ips, addr.String())
	}
	sort.Strings(ips)
	return fmt.Sprintf(`the given hostname %s does not resolve to a unique ip.
A unique ip is necessary to ensure that the upgrade script
connects to only the (primary) administration Controller.
The hostname resolves to the following ips:
%s
`, r.hostname, strings.Join(ips, "\n"))
}

func GetRealHostname(controller openapi.Appliance) (string, error) {
	realHost := controller.GetHostname()
	if i, ok := controller.GetPeerInterfaceOk(); ok {
		realHost = i.GetHostname()
	}
	if i, ok := controller.GetAdminInterfaceOk(); ok {
		realHost = i.GetHostname()
	}
	if err := ValidateHostname(controller, realHost); err != nil {
		return "", err
	}
	return realHost, nil
}

func ValidateHostname(controller openapi.Appliance, hostname string) error {
	var h string
	if ai, ok := controller.GetAdminInterfaceOk(); ok {
		h = ai.GetHostname()
	}
	if pi, ok := controller.GetPeerInterfaceOk(); ok && len(h) <= 0 {
		h = pi.GetHostname()
	}
	if len(h) <= 0 {
		return fmt.Errorf("failed to determine hostname for controller admin interface")
	}

	cHost := strings.ToLower(h)
	nHost := strings.ToLower(hostname)
	if cHost != nHost {
		log.WithFields(log.Fields{
			"controller-hostname": cHost,
			"connected-hostname":  nHost,
		}).Error("no match")
		return fmt.Errorf("Hostname validation failed. Pass the --actual-hostname flag to use the real controller hostname")
	}

	if err := ValidateHostnameUniqueness(nHost); err != nil {
		return err
	}

	return nil
}

func ValidateHostnameUniqueness(hostname string) error {
	var errs error
	errCount := 0
	ctx := context.Background()
	resolver := net.DefaultResolver
	ipv4s, err := resolver.LookupIP(ctx, "ip4", hostname)
	if err != nil {
		errCount++
		err = fmt.Errorf("ipv4: %w", err)
		errs = multierror.Append(err, errs)
	}
	ipv6s, err := resolver.LookupIP(ctx, "ip6", hostname)
	if err != nil {
		errCount++
		err = fmt.Errorf("ipv6: %w", err)
		errs = multierror.Append(err, errs)
	}
	// We check errors, but only one needs to succeed, so we also count the errors before determining if we return an error
	if errs != nil && errCount > 1 {
		return errs
	}
	v4length := len(ipv4s)
	v6length := len(ipv6s)
	if v4length > 1 || v6length > 1 {
		return &ResolverError{
			ip:       append(ipv4s, ipv6s...),
			hostname: hostname,
		}
	}

	return nil
}
