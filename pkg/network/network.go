package network

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
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

// ValidateHostnameUniqueness Validate that the given hostname resolves to at most 1 ip per ip version.
func ValidateHostnameUniqueness(addr string) error {
	var errs error
	errCount := 0
	ctx := context.Background()
	resolver := net.DefaultResolver
	ipv4, err := resolver.LookupIP(ctx, "ip4", addr)
	if err != nil {
		errCount++
		errs = multierror.Append(errs, fmt.Errorf("ipv4: %w", err))
	}
	ipv6, err := resolver.LookupIP(ctx, "ip6", addr)
	if err != nil {
		errCount++
		err = fmt.Errorf("ipv6: %w", err)
		errs = multierror.Append(err, errs)
	}
	// We check errors, but only one needs to succeed, so we also count the errors before determining if we return an error
	if errs != nil && errCount > 1 {
		return errs
	}
	v4length := len(ipv4)
	v6length := len(ipv6)
	if v4length > 1 || v6length > 1 {
		return &ResolverError{
			ip:       append(ipv4, ipv6...),
			hostname: addr,
		}
	}

	return nil
}
