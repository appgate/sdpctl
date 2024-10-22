package util

import "testing"

func TestSmallestGroupIndex(t *testing.T) {
	tests := []struct {
		name   string
		groups [][]string
		want   int
	}{
		{
			name: "should be 0",
			groups: [][]string{
				{"value"},
				{"value", "value"},
				{"value", "value", "value"},
			},
			want: 0,
		},
		{
			name: "should be 1",
			groups: [][]string{
				{"value", "value"},
				{"value"},
				{"value", "value", "value"},
			},
			want: 1,
		},
		{
			name: "should be 2",
			groups: [][]string{
				{"value", "value"},
				{"value", "value", "value"},
				{"value"},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SmallestGroupIndex(tt.groups); got != tt.want {
				t.Errorf("SmallestGroupIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
