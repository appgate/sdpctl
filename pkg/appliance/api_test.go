package appliance

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func TestGetPeerAPIVersion(t *testing.T) {
	tests := map[string]struct {
		testVersion   string
		expectVersion int
	}{
		"test 5.1": {
			testVersion:   "5.1",
			expectVersion: 12,
		},
		"test 5.2": {
			testVersion:   "5.2",
			expectVersion: 13,
		},
		"test 5.3": {
			testVersion:   "5.3",
			expectVersion: 14,
		},
		"test 5.3.4+24950": {
			testVersion:   "5.3.4+24950",
			expectVersion: 14,
		},
		"test 5.4": {
			testVersion:   "5.4",
			expectVersion: 15,
		},
		"test 5.4.7+27205": {
			testVersion:   "5.4.7+27205",
			expectVersion: 15,
		},
		"test 5.5": {
			testVersion:   "5.5",
			expectVersion: 16,
		},
		"test 6.0": {
			testVersion:   "6.0",
			expectVersion: 17,
		},
		"test 6.1": {
			testVersion:   "6.1",
			expectVersion: 18,
		},
	}

	for k, tt := range tests {
		t.Run(k, func(t *testing.T) {
			tv, _ := version.NewVersion(tt.testVersion)
			a := Appliance{}
			result := a.GetPeerAPIVersion(tv)
			if tt.expectVersion != result {
				t.Errorf("peer version test failed:\nWANT: %d\nGOT: %d", tt.expectVersion, result)
			}
		})
	}
}
