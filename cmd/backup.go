package cmd

import (
	"path/filepath"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/cmd/backup"

	"github.com/spf13/cobra"
)

var (
	destinationFlag    string
	allFlag            bool
	allControllersFlag bool
	longDescription    string = `Appgate backup script.

© 2021 Appgate Cybersecurity, Inc.
All rights reserved. Appgate is a trademark of Appgate Cybersecurity, Inc.
htts://www.appgate.com

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html
`
)

func NewCmdBackup(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup [flags] CONTROLLER",
		Short: "Perform backup of the Appgate SDP Collective",
		Long:  longDescription,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return c.Validate()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return backup.Prepare(c, destinationFlag)
		},
		ValidArgs: []string{"controller"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return backup.Perform(c)
		},
	}

	defaultBackupDir := filepath.FromSlash("$HOME/appgate/appgate_backup_yyyymmdd_hhMMss")
	cmd.PersistentFlags().StringVarP(&destinationFlag, "destination", "d", defaultBackupDir, "backup destination")
	cmd.PersistentFlags().BoolVar(&allFlag, "all", false, "backup the entire Appgate SDP Collective")
	cmd.PersistentFlags().BoolVar(&allControllersFlag, "controllers", false, "backup all controllers")

	return cmd
}
