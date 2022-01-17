package util

import "testing"

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
			name: "valid url",
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
