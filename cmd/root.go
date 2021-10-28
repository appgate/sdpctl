package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var debug bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}

var rootCmd = &cobra.Command{
	PreRun: preRunFunc,
	Use:   "appgatectl [COMMAND]",
	Short: "appgatectl is a command line tool to control and handle Appgate SDP using the CLI",
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
