package configuration

import (
	"testing"

	"github.com/appgate/sdpctl/pkg/keyring"
	zkeyring "github.com/zalando/go-keyring"
)

func TestConfigCheckAuth(t *testing.T) {
	zkeyring.MockInit()
	if err := keyring.SetBearer("controller.appgate.com", "abc123456789"); err != nil {
		t.Fatalf("unable to mock keyring in TestConfigCheckAuth() %v", err)
	}
	type fields struct {
		URL                      string
		Provider                 string
		Insecure                 bool
		Debug                    bool
		Version                  int
		BearerToken              string
		ExpiresAt                string
		CredentialsFile          string
		DeviceID                 string
		PemFilePath              string
		PrimaryControllerVersion string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				ExpiresAt: "2031-12-08 08:15:39.137584 +0000 UTC",
				URL:       "https://controller.appgate.com",
				Provider:  "local",
			},
			want: true,
		},
		{
			name: "invalid expire date",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
				URL:       "https://controller.appgate.com",
				Provider:  "local",
			},
			want: false,
		},
		{
			name: "no token",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
			},
			want: false,
		},
		{
			name: "no url",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
				Provider:  "local",
			},
			want: false,
		},
		{
			name: "no provider",
			fields: fields{
				ExpiresAt: "2001-01-01 08:15:39.137584 +0000 UTC",
				URL:       "https://controller.appgate.com",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				URL:                      tt.fields.URL,
				Provider:                 tt.fields.Provider,
				Insecure:                 tt.fields.Insecure,
				Debug:                    tt.fields.Debug,
				Version:                  tt.fields.Version,
				BearerToken:              tt.fields.BearerToken,
				ExpiresAt:                tt.fields.ExpiresAt,
				DeviceID:                 tt.fields.DeviceID,
				PemFilePath:              tt.fields.PemFilePath,
				PrimaryControllerVersion: tt.fields.PrimaryControllerVersion,
			}
			if got := c.CheckAuth(); got != tt.want {
				t.Errorf("Config.CheckAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigGetHost(t *testing.T) {
	type fields struct {
		URL string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "valid URL",
			fields: fields{
				URL: "http://controller.com/admin",
			},
			want:    "controller.com",
			wantErr: false,
		},
		{
			name: "ipv6 addr",
			fields: fields{
				URL: "http://[fd00:ffff:a:93:172:17:93:35]:666/admin",
			},
			want:    "fd00:ffff:a:93:172:17:93:35",
			wantErr: false,
		},
		{
			name: "empty URL",
			fields: fields{
				URL: "",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				URL: tt.fields.URL,
			}
			got, err := c.GetHost()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GetHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.GetHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		Name string
		URL  string
		want string
		err  bool
	}{
		{
			Name: "Full valid URL",
			URL:  "https://some.valid.url:8443/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No scheme",
			URL:  "some.valid.url:8443/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "HTTP scheme",
			URL:  "http://some.valid.url:8443/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No path",
			URL:  "https://some.valid.url:8443",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No port",
			URL:  "https://some.valid.url/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No port and path",
			URL:  "https://some.valid.url",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No port, path or scheme",
			URL:  "some.valid.url",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "No scheme or port",
			URL:  "some.valid.url/admin",
			want: "https://some.valid.url:8443/admin",
		},
		{
			Name: "Other port",
			URL:  "https://some.valid.url:443/admin",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "Other port, no path",
			URL:  "https://some.valid.url:443",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "Other port, no path, no scheme",
			URL:  "some.valid.url:443",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "Other port, no path, no scheme",
			URL:  "some.valid.url:443",
			want: "https://some.valid.url:443/admin",
		},
		{
			Name: "No URL",
			URL:  "",
			want: "",
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result, err := NormalizeURL(tt.URL)
			if err != nil && !tt.err {
				t.Fatalf("Test failed. Error: %v", err)
			}

			if result != tt.want {
				t.Fatalf("FAILED! EXPECTED: %s, GOT: %s", tt.want, result)
			}
		})
	}
}
