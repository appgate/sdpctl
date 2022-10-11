package network

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
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
	resolver := net.DefaultResolver
	ctx := context.Background()

	// errors ignored since we don't know if
	// both v4 and v6 is supported.
	// for example,  *net.AddrError can return
	// - no suitable address found
	// if ipv6 is not supported on the host.
	ipv4, _ := resolver.LookupIP(ctx, "ip4", addr)
	ipv6, _ := resolver.LookupIP(ctx, "ip6", addr)

	if len(ipv4) > 1 || len(ipv6) > 1 {
		return &ResolverError{
			ip:       append(ipv4, ipv6...),
			hostname: addr,
		}
	}
	return nil
}
