package cmd

import (
	"github.com/appgate/appgatectl/pkg/cmd/factory"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCmdBackup(f *factory.Factory) *cobra.Command {
	opts := configureOptions{
		Config: f.Config,
	}

	return &cobra.Command{
		Use:   "backup [flags] CONTROLLER",
		Short: "Perform backup of the Appgate SDP Collective",
		Long: `Appgate backup script.

© 2021 Appgate Cybersecurity, Inc.
All rights reserved. Appgate is a trademark of Appgate Cybersecurity, Inc.
htts://www.appgate.com

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html
    `,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            return opts.Config.Validate()
        },
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("Running backup script")
			return nil
		},
	}
}
