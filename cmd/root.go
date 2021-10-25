package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "appgatectl [COMMAND]",
	Short: "appgatectl is a command line tool to control and handle Appgate SDP using the CLI",
	Aliases: []string{
		"agctl",
		"ag",
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
