package cmdutil

import (
	"os"
	"path/filepath"
	"regexp"
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
