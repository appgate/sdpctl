package network

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v17/openapi"
	"github.com/google/uuid"
)

func TestGetRealHostname(t *testing.T) {
	testCases := []struct {
		name                   string
		expect                 string
		hostname               string
		peerInterfaceHostname  string
		adminInterfaceHostname string
		wantErr                bool
		matchErr               *regexp.Regexp
	}{
		{
			name:                   "test admin interface hostname",
			expect:                 "appgate.com",
			hostname:               "fakehost1.devops",
			peerInterfaceHostname:  "fakehost2.devops",
			adminInterfaceHostname: "appgate.com",
		},
		{
			name:                  "test no admin interface",
			expect:                "appgate.com",
			hostname:              "fakehost1.devops",
			peerInterfaceHostname: "appgate.com",
		},
		{
			name:                   "empty hostname",
			expect:                 "appgate.com",
			adminInterfaceHostname: "appgate.com",
		},
		{
			name:                  "empty admin hostname",
			expect:                "appgate.com",
			hostname:              "fakehost.devops",
			peerInterfaceHostname: "appgate.com",
		},
		{
			name:     "no hostname",
			wantErr:  true,
			matchErr: regexp.MustCompile("failed to determine hostname for controller admin interface"),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := openapi.Appliance{
				Id:        openapi.PtrString(uuid.New().String()),
				Name:      "controller",
				Activated: openapi.PtrBool(true),
			}
			if len(tt.hostname) > 0 {
				ctrl.Hostname = tt.hostname
			}
			if len(tt.peerInterfaceHostname) > 0 {
				ctrl.PeerInterface = &openapi.ApplianceAllOfPeerInterface{
					Hostname: *openapi.PtrString(tt.peerInterfaceHostname),
				}
			}
			if len(tt.adminInterfaceHostname) > 0 {
				ctrl.AdminInterface = &openapi.ApplianceAllOfAdminInterface{
					Hostname: *openapi.PtrString(tt.adminInterfaceHostname),
				}
			}
			result, err := GetRealHostname(ctrl)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("GetRealHostname() error: %v", err)
				}
				if !tt.matchErr.MatchString(err.Error()) {
					t.Fatalf("GetRealHostname() - error does not match. WANT: %s, GOT: %s", tt.matchErr.String(), err.Error())
				}
			}
			if result != tt.expect {
				t.Fatalf("GetRealHostname() - unexpected result. WANT: %s, GOT: %s", tt.expect, result)
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
			hostname:      "appgate.com",
			adminHostName: "appgate.com",
			wantErr:       false,
		},
		{
			name:          "not unique hostname",
			hostname:      "play.google.com",
			adminHostName: "play.google.com",
			wantErr:       true,
			want:          *regexp.MustCompile(fmt.Sprintf(`the given hostname %s does not resolve to a unique ip\.`, "play.google.com")),
		},
		{
			name:          "admin interface not hostname",
			hostname:      "controller.devops",
			adminHostName: "appgate.com",
			wantErr:       true,
			want:          *regexp.MustCompile(`Hostname validation failed. Pass the --actual-hostname flag to use the real controller hostname`),
		},
	}

	for _, tt := range tests {
		ctrl := openapi.Appliance{
			Id:        openapi.PtrString(uuid.New().String()),
			Name:      "controller",
			Activated: openapi.PtrBool(true),
			Hostname:  tt.hostname,
			PeerInterface: &openapi.ApplianceAllOfPeerInterface{
				Hostname:  tt.adminHostName,
				HttpsPort: openapi.PtrInt32(444),
			},
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
