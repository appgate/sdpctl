package factory

import (
	"net/http"
	"testing"

	"github.com/appgate/appgatectl/internal/config"
)

func TestHttpClientTransportTLSFromConfig(t *testing.T) {
	type args struct {
		f *Factory
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "tls insecure transport",
			args: args{
				&Factory{
					Config: &config.Config{
						Insecure: true,
					},
				},
			},
			want: true,
		},
		{
			name: "tls secure transport",
			args: args{
				&Factory{
					Config: &config.Config{
						Insecure: false,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := httpClientFunc(tt.args.f)()
			if err != nil {
				t.Errorf("got error %v", err)
				return
			}
			tr := c.Transport.(*http.Transport)
			if tr.TLSClientConfig.InsecureSkipVerify != tt.want {
				t.Fatalf("got %v expected %v", tr.TLSClientConfig.InsecureSkipVerify, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		appVersion string
		config     *config.Config
	}
	tests := []struct {
		name string
		args args
		want *Factory
	}{
		{
			name: "test basic API client",
			args: args{
				appVersion: "1.1.1.",
				config: &config.Config{
					Insecure: true,
					Version:  15,
					Debug:    false,
					URL:      "https://appgate.controller.com/admin",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.appVersion, tt.args.config)
			inputCfg := tt.args.config
			c, err := got.APIClient(inputCfg)
			if err != nil {
				t.Fatalf("got err %s", err)
			}
			clientConfig := c.GetConfig()
			if clientConfig.Servers[0].URL != inputCfg.URL {
				t.Errorf("Got %s expected %s", clientConfig.Host, inputCfg.URL)
			}
			if clientConfig.UserAgent != "appgatectl/1.1.1./go" {
				t.Errorf("Got %s expected %s", clientConfig.UserAgent, "appgatectl/1.1.1./go")
			}
			if clientConfig.Debug != inputCfg.Debug {
				t.Errorf("Got %v expected %v", clientConfig.Debug, inputCfg.Debug)
			}
			if clientConfig.DefaultHeader["Accept"] != "application/vnd.appgate.peer-v15+json" {
				t.Errorf("Got %s expected %s", clientConfig.DefaultHeader["Accept"], "application/vnd.appgate.peer-v15+json")
			}
		})
	}
}
