package backup

import (
	"time"

	"github.com/appgate/appgatectl/pkg/appliance"
	"github.com/appgate/appgatectl/pkg/factory"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const (
	longDescription string = `The backup script will request a backup from the API and download them to a destination directory. The script requires the backup API to be enabled in
the Appgate SDP Collective. In case the backup API is not enabled when executing the backup command, you will be prompted to activate it.

There are multiple options for selecting which Appgate SDP Appliances to backup, using flags or optional arguments. The arguments are expected to be the name of
the Appgate SDP Appliance you want to take a backup of.

The default destination directory is set to be the users default downloads directory on the system. If the default destination is used, an 'appgate' directory
will be created there if it doesn't already exist and the backups will be downloaded to that. In case custom destination directory is specified by using the
'--destination' flag, the extra 'appgate' directory will not be created. The user also has to have write privileges on the specified directory.

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html`

	example string = `# backup with no arguments or flags will prompt for appliance
$ appgatectl appliance backup

# download backups to a custom directory
$ appgatectl appliance backup --destination=path/to/backup/destination

# backup only primary controller using flag
$ appgatectl appliance backup --primary

# backup all Appgate SDP Appliances
$ appgatectl appliance backup --all

# backup using '--filter' and '--exclude' flags
$ appgatectl appliance backup --filter=function=controller --exclude=tag=secondary
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
		Use:       "backup [flags] [appliance]",
		Short:     "Perform backup of the Appgate SDP Collective",
		Long:      longDescription,
		Example:   example,
		ValidArgs: []string{"appliance"},
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
	flags.StringSliceVarP(&opts.Include, "include", "i", []string{}, "include extra data in backup (audit,logs)")
	flags.DurationVarP(&opts.Timeout, "timeout", "t", 5*time.Minute, "time out for status check on the backups")

	cmd.AddCommand(NewBackupAPICmd(f))

	return cmd
}
