## sdpctl appliance backup

Perform backup of the Appgate SDP Collective appliances

### Synopsis

The backup command will request a backup from the API and download them to a destination directory. The command requires the backup API to be enabled in
the Appgate SDP Collective. In case the backup API is not enabled when executing the backup command, you will be prompted to activate it.

There are multiple options for selecting which Appgate SDP Appliances to backup, using flags or optional arguments. The arguments are expected to be the name of
the Appgate SDP Appliance you want to take a backup of.

The default destination directory is set to be the users default downloads directory on the system. If the default destination is used, an 'appgate' directory
will be created there if it doesn't already exist and the backups will be downloaded to that. In case custom destination directory is specified by using the
'--destination' flag, the extra 'appgate' directory will not be created. The user also has to have write privileges on the specified directory.

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html

```
sdpctl appliance backup [flags]
```

### Examples

```
  # Backup with no arguments or flags will prompt for appliance
  > sdpctl appliance backup
  ? Backup API is disabled on the appliance. Do you want to enable it now? Yes
  ? The passphrase to encrypt the appliance backups when the Backup API is used: <password> # only shows if backup API is not enabled
  ? Confirm your passphrase: <password> # only shows if backup API is not enabled
  ? select appliances to backup:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
  > [ ]  controller
    [ ]  gateway

  # download backups to a custom directory
  > sdpctl appliance backup --destination=path/to/backup/destination

  # backup only primary controller using flag
  > sdpctl appliance backup --primary

  # backup all Appgate SDP Appliances
  > sdpctl appliance backup --all

  # backup using '--include' and '--exclude' flags
  > sdpctl appliance backup --include=function=controller --exclude=tag=secondary
```

### Options

```
      --all                  backup all Appliances in the Appgate SDP Collective
      --current              backup current peer controller
  -d, --destination string   backup destination directory (default "/home/larskajes/Downloads/appgate/backup")
  -h, --help                 help for backup
      --primary              backup primary controller
      --quiet                backup summary will not be printed if setting this flag
      --with strings         include extra data in backup (audit,logs)
```

### Options inherited from parent commands

```
      --api-version int          peer API version override
      --ci-mode                  log to stderr instead of file and disable progress-bars
      --debug                    Enable debug logging
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
      --no-interactive           suppress interactive prompt with auto accept
      --no-verify                don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string           profile configuration to use
```

### SEE ALSO

* [sdpctl appliance](sdpctl_appliance.md)	 - interact with Appgate SDP Appliances
* [sdpctl appliance backup api](sdpctl_appliance_backup_api.md)	 - Controls the state of the backup API.

