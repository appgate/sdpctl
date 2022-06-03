package util

import (
	"reflect"
	"testing"
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
