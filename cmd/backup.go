package cmd

import (
	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/cmd/factory"

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

func NewCmdBackup(f *factory.Factory) *cobra.Command {
	opts := appliance.BackupOpts{
		Config:      f.Config,
		Out:         f.IOOutWriter,
		Appliance:   f.Appliance,
		Destination: appliance.DefaultBackupDestination,
		Audit:       true,
		Logs:        true,
	}
	cmd := &cobra.Command{
		Use:       "backup [flags] CONTROLLER",
		Short:     "Perform backup of the Appgate SDP Collective",
		Long:      longDescription,
		ValidArgs: []string{"controller"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return f.Config.Validate()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return appliance.PrepareBackup(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return appliance.PerformBackup(&opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&destinationFlag, "destination", "d", appliance.DefaultBackupDestination, "backup destination")
	cmd.PersistentFlags().BoolVar(&allFlag, "all", false, "backup the entire Appgate SDP Collective")
	cmd.PersistentFlags().BoolVar(&allControllersFlag, "controllers", false, "backup all controllers")

	return cmd
}
