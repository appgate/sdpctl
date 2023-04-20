package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/go-version"
)

func TestIsValidURL(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid url",
			args: args{
				addr: "https://appgate.com",
			},
			want: true,
		},
		{
			name: "invalid url",
			args: args{
				addr: "appgate.com",
			},
			want: false,
		},
		{
			name: "empty string",
			args: args{
				addr: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidURL(tt.args.addr); got != tt.want {
				t.Errorf("IsValidURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsJSON(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid json",
			args: args{
				`
            {
                "foo": "bar"
            }
            `,
			},
			want: true,
		},
		{
			name: "incomplete json",
			args: args{
				`
            {
                "foo": "bar
            `,
			},
			want: false,
		},
		{
			name: "empty string",
			args: args{""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSON(tt.args.str); got != tt.want {
				t.Errorf("IsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchSlice(t *testing.T) {
	type args struct {
		needle          string
		haystack        []string
		caseInsensitive bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "case insensitive search match",
			args: args{
				needle:          "controller",
				caseInsensitive: true,
			},
			want: []string{"controller", "Controller"},
		},
		{
			name: "case sensitive search match",
			args: args{
				needle:          "controller",
				caseInsensitive: false,
			},
			want: []string{"controller"},
		},
		{
			name: "case insensitive no match",
			args: args{
				needle:          "randomTerm",
				caseInsensitive: true,
			},
			want: []string{},
		},
		{
			name: "case sensitive no match",
			args: args{
				needle:          "portal",
				caseInsensitive: false,
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.haystack = []string{
				"controller",
				"Controller",
				"Connector",
				"Gateway",
				"LogServer",
				"LogForwarder",
				"Portal",
			}
			if got := SearchSlice(tt.args.needle, tt.args.haystack, tt.args.caseInsensitive); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SearchSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDockerTagVersion(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		v       string
		want    string
		wantErr bool
	}{
		{
			name: "version 6.2.0",
			v:    "6.2.0",
			want: "6.2",
		},
		{
			name: "version 6.2.1",
			v:    "6.2.1",
			want: "6.2",
		},
		{
			name: "version 6.0.0",
			v:    "6.0.0",
			want: "6.0",
		},
		{
			name: "override with env variable",
			v:    "6.2.1",
			env:  "latest",
			want: "latest",
		},
		{
			name:    "version is nil",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v *version.Version
			var err error
			if len(tt.v) > 0 {
				v, err = version.NewVersion(tt.v)
				if err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("SDPCTL_DOCKER_TAG", tt.env)
			got, err := DockerTagVersion(v)
			if err != nil && !tt.wantErr {
				t.Errorf("DockerTagVersion() = %v, want no err", err)
			}
			if got != tt.want {
				t.Errorf("DockerTagVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
