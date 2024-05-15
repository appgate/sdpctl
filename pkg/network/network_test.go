package network

import (
	"reflect"
	"slices"
	"testing"

	"github.com/appgate/sdpctl/pkg/dns"
	"github.com/foxcpp/go-mockdns"
)

func TestResolveHostnameIPs(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		want    []string
		wantErr error
	}{
		{
			name: "test resolve to 1",
			want: []string{"9.6.5.7"},
		},
		{
			name:    "no resolution",
			wantErr: ErrNameResolution,
		},
		{
			name: "resolve multiple",
			want: []string{"9.4.5.6", "8.5.6.2"},
		},
		{
			name: "resolve with custom port in hostname",
			addr: "https://appgate.test:8567",
			want: []string{"9.8.7.6"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slices.Sort(tt.want)
			_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{
				"appgate.test.": {
					A: tt.want,
				},
			})
			defer teardown()
			addr := "https://appgate.test"
			if len(tt.addr) > 0 {
				addr = tt.addr
			}
			got, err := ResolveHostnameIPs(addr)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("ResolveHostnameIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveHostnameIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateHostnameUniqueness(t *testing.T) {
	type resolution struct {
		ipv4 []string
		ipv6 []string
	}
	tests := []struct {
		name       string
		resolution resolution
		wantErr    bool
	}{
		{
			name: "resolve no error",
			resolution: resolution{
				ipv4: []string{"7.6.8.5"},
			},
		},
		{
			name: "want error",
			resolution: resolution{
				ipv4: []string{"5.6.7.8", "4.5.6.7"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{
				"appgate.test.": {
					A:    tt.resolution.ipv4,
					AAAA: tt.resolution.ipv6,
				},
			})
			defer teardown()
			if err := ValidateHostnameUniqueness("appgate.test"); (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostnameUniqueness() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
