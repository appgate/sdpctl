package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	version       string
	commit        string
	buildDate     string
	debug         bool
	versionOutput string
)

func init() {
	versionOutput = fmt.Sprintf(`%s
commit: %s
build date: %s`, version, commit, buildDate)

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}

var rootCmd = &cobra.Command{
	PreRun:  preRunFunc,
	Use:     "appgatectl [COMMAND]",
	Short:   "appgatectl is a command line tool to control and handle Appgate SDP using the CLI",
	Version: versionOutput,
	Aliases: []string{
		"agctl",
		"ag",
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func preRunFunc(cmd *cobra.Command, args []string) {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}
