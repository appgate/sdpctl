package factory

import (
	"testing"

	"github.com/appgate/sdpctl/pkg/configuration"
)

func TestHttpTransportTLSFromConfig(t *testing.T) {
	type args struct {
		f *Factory
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		wantInsecure bool
	}{
		{
			name: "tls insecure transport",
			args: args{
				&Factory{
					Config: &configuration.Config{
						Insecure: true,
					},
				},
			},
			wantErr:      false,
			wantInsecure: true,
		},
		{
			name: "tls secure transport",
			args: args{
				&Factory{
					Config: &configuration.Config{
						Insecure: false,
					},
				},
			},
			wantErr:      false,
			wantInsecure: false,
		},
		{
			name: "test with invalid pem file",
			args: args{
				&Factory{
					Config: &configuration.Config{
						Insecure:    false,
						PemFilePath: "testdata/invalid_cert.pem",
					},
				},
			},
			wantErr:      true,
			wantInsecure: false,
		},
		{
			name: "test with valid pem file",
			args: args{
				&Factory{
					Config: &configuration.Config{
						Insecure:    false,
						PemFilePath: "testdata/cert.pem",
					},
				},
			},
			wantErr:      false,
			wantInsecure: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := httpTransport(tt.args.f)()
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v", err)
				return
			}

			if tr != nil {
				if tr.TLSClientConfig.InsecureSkipVerify != tt.wantInsecure {
					t.Fatalf("got %v expected %v", tr.TLSClientConfig.InsecureSkipVerify, tt.wantInsecure)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		appVersion string
		config     *configuration.Config
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
				config: &configuration.Config{
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
			if clientConfig.UserAgent != "sdpctl/1.1.1." {
				t.Errorf("Got %s expected %s", clientConfig.UserAgent, "sdpctl/1.1.1.")
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
