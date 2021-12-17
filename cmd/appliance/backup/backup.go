package backup

import (
	"time"

	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/factory"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const (
	longDescription string = `Appgate backup script.

© 2021 Appgate Cybersecurity, Inc.
All rights reserved. Appgate is a trademark of Appgate Cybersecurity, Inc.
htts://www.appgate.com

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html
`
)

func NewCmdBackup(f *factory.Factory) *cobra.Command {
	var backupIDs map[string]string
	opts := appliance.BackupOpts{
		Config:      f.Config,
		Out:         f.IOOutWriter,
		Appliance:   f.Appliance,
		Destination: appliance.DefaultBackupDestination,
	}
	cmd := &cobra.Command{
		Use:       "backup [flags]",
		Short:     "Perform backup of the Appgate SDP Collective",
		Long:      longDescription,
		ValidArgs: []string{"controller"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return appliance.PrepareBackup(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			backupIDs, err = appliance.PerformBackup(cmd, &opts)
			if err != nil {
				return err
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return appliance.CleanupBackup(&opts, backupIDs)
		},
	}

	log.SetOutput(opts.Out)
	flags := cmd.Flags()
	flags.StringVarP(&opts.Destination, "destination", "d", appliance.DefaultBackupDestination, "backup destination")
	flags.BoolVar(&opts.AllFlag, "all", false, "backup the entire Appgate SDP Collective")
	flags.BoolVar(&opts.AllControllersFlag, "controllers", false, "backup all controllers")
	flags.StringSliceVarP(&opts.Include, "include", "i", []string{}, "include extra data in backup (audit,logs)")
	flags.DurationVarP(&opts.Timeout, "timeout", "t", 5*time.Minute, "time out for status check on the backups")

	return cmd
}
