package cmdutil

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/appgate/sdpctl/pkg/profiles"
)

func GetCaller() string {
	binary := "sdpctl"
	raw := os.Args[0]
	regex := regexp.MustCompile(`sdpctl`)
	if bin := filepath.Base(raw); regex.MatchString(bin) {
		binary = bin
	}
	return binary
}

func GetConfigPath() string {
	return profiles.GetConfigPath()
}

func GetLogPath() string {
	return profiles.GetLogPath()
}

func CurrentProfile() string {
	return profiles.GetCurrentProfile()
}
