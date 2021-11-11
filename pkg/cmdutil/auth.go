package cmdutil

import (
	"time"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/spf13/cobra"
)

func CheckAuth(c config.Config) bool {
	layout := "2006-01-02 15:04:05.999999999 -0700 MST"
	d, err := time.Parse(layout, c.ExpiresAt)
	if err != nil {
		return false
	}
	if len(c.BearerToken) < 1 {
		return false
	}
	if len(c.URL) < 1 {
		return false
	}
	if len(c.Provider) < 1 {
		return false
	}
	t1 := time.Now()
	return t1.Before(d)
}

func IsAuthCheckEnabled(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return false
	}
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations["skipAuthCheck"] == "true" {
			return false
		}
	}
	return true
}
