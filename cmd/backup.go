package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/appgate/appgatectl/internal/config"
	"github.com/appgate/appgatectl/pkg/cmd/backup"
    log "github.com/sirupsen/logrus"

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
		ValidArgs: []string{"controller"},
		RunE:      runBackup(c),
	}

	homedirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to determine user home directory:", err)
	}
	defaultBackupDir := filepath.FromSlash(fmt.Sprintf("%s/appgate/appgate_backup_yyyymmdd_hhMMss", homedirname))
	cmd.PersistentFlags().StringVarP(&destinationFlag, "destination", "d", defaultBackupDir, "backup destination")
	cmd.PersistentFlags().BoolVar(&allFlag, "all", false, "backup the entire Appgate SDP Collective")
	cmd.PersistentFlags().BoolVar(&allControllersFlag, "controllers", false, "backup all controllers")

	return cmd
}

func runBackup(c *config.Config) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
        log.Debug(args)
		err := backup.Prepare(c, destinationFlag)
		if err != nil {
			return fmt.Errorf("Backup preperation failed: %s", err)
		}

		err = backup.Perform(c, cmd, args)
		if err != nil {
			return fmt.Errorf("Failed to perform backup: %s", err)
		}

		err = backup.Cleanup(c, cmd, args)
		if err != nil {
			return fmt.Errorf("Failed to cleanup after backup: %s", err)
		}
		return nil
	}
}
