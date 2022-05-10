package backup

import (
	"time"

	"github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func NewCmdBackup(f *factory.Factory) *cobra.Command {
	var backupIDs map[string]string
	opts := appliance.BackupOpts{
		Config:      f.Config,
		Out:         f.IOOutWriter,
		SpinnerOut:  f.SpinnerOut,
		Appliance:   f.Appliance,
		Destination: appliance.DefaultBackupDestination,
	}
	cmd := &cobra.Command{
		Use:     "backup",
		Short:   docs.ApplianceBackupDoc.Short,
		Long:    docs.ApplianceBackupDoc.Long,
		Example: docs.ApplianceBackupDoc.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if opts.NoInteractive, err = cmd.Flags().GetBool("no-interactive"); err != nil {
				return err
			}
			return appliance.PrepareBackup(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			backupIDs, err = appliance.PerformBackup(cmd, args, &opts)
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
	flags.StringVarP(&opts.Destination, "destination", "d", appliance.DefaultBackupDestination, "backup destination directory")
	flags.BoolVar(&opts.AllFlag, "all", false, "backup all Appliances in the Appgate SDP Collective")
	flags.BoolVar(&opts.PrimaryFlag, "primary", false, "backup primary controller")
	flags.BoolVar(&opts.CurrentFlag, "current", false, "backup current peer controller")
	flags.StringSliceVar(&opts.With, "with", []string{}, "include extra data in backup (audit,logs)")
	flags.DurationVarP(&opts.Timeout, "timeout", "t", 5*time.Minute, "time out for status check on the backups")
	flags.BoolVar(&opts.Quiet, "quiet", false, "backup summary will not be printed if setting this flag")

	cmd.AddCommand(NewBackupAPICmd(f))

	return cmd
}
