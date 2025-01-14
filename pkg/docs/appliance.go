package docs

var (
	ApplianceRootDoc = CommandDoc{
		Short: "Manage the appliances and perform tasks such as backups, upgrades, metrics etc",
		Long: `The base command to manage the appliances. This command does not do anything by itself, it is
used together with one of the available sub-commands listed below.`,
	}
	ApplianceListDoc = CommandDoc{
		Short: "List all appliances",
		Long: `List all appliances in the Collective. The appliances will be listed in no particular order. Using without arguments
will print a table view with a limited set of information. Using the command with the provided '--json' flag will print out a more detailed
list view in json format. The list command can also be combined with the global '--include' and '--exclude' flags`,
		Examples: []ExampleDoc{
			{
				Description: "Default list command",
				Command:     "sdpctl appliance list",
				Output: `Name                                                   ID                                    Hostname                 Site          Activated
----                                                   --                                    --------                 ----          ---------
controller                                             67f7ee0c-924c-4253-8b78-0882ff0665ab  controller.dev           Default Site  true
gateway                                                ec3b6270-ad7e-447a-a6e6-8f4ae816cab5  gateway.dev              Default Site  true`,
			},
			{
				Description: "Print list of appliances in json format",
				Command:     "sdpctl appliance list --json",
			},
			{
				Description: "Print a filtered list of appliances",
				Command:     "sdpctl appliance list --include=<key>=<value>",
			},
		},
	}
	ApplianceBackupDoc = CommandDoc{
		Short: "Perform backup of the appliances",
		Long: `The backup command will request a backup from the API and download them to a destination directory. The command requires the Backup API to be enabled in
the Collective. In case the Backup API is not enabled when executing the backup command, you will be prompted to activate it.

There are multiple options for selecting which Appliances to backup, using flags or optional arguments. The arguments are expected to be the name of
the appliance you want to take a backup of.

The default destination directory is set to be the users default downloads directory on the system. If the default destination is used, an 'appgate' directory
will be created there if it doesn't already exist and the backups will be downloaded to that. In case custom destination directory is specified by using the
'--destination' flag, the extra 'appgate' directory will not be created. The user also has to have write privileges on the specified directory.

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html`,
		Examples: []ExampleDoc{
			{
				Description: "Backup with no arguments or flags will prompt for appliance",
				Command:     "sdpctl appliance backup",
				Output: `? Backup API is disabled on the appliance. Do you want to enable it now? Yes
? The passphrase to encrypt the appliance backups when the Backup API is used: <password> # only shows if Backup API is not enabled
? Confirm your passphrase: <password> # only shows if Backup API is not enabled
? select appliances to backup:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
❯ [ ]  controller
  [ ]  gateway`,
			},
			{
				Description: "download backups to a custom directory",
				Command:     "sdpctl appliance backup --destination=path/to/backup/destination",
			},
			{
				Description: "backup only the primary Controller using flag",
				Command:     "sdpctl appliance backup --primary",
			},
			{
				Description: "backup all appliances",
				Command:     "sdpctl appliance backup --all",
			},
			{
				Description: "backup using '--include' and '--exclude' flags",
				Command:     "sdpctl appliance backup --include=function=controller --exclude=tag=secondary",
			},
		},
	}
	ApplianceBackupAPIDoc = CommandDoc{
		Short: "Controls the state of the Backup API",
		Long: `This command controls the state of the Backup API on the Collective.
You will be prompted for a passphrase for the backups when enabling the Backup API using this command.
The passphrase is required.`,
		Examples: []ExampleDoc{
			{
				Description: "enable the Backup API",
				Command:     "appgate appliance backup api",
			},
			{
				Description: "disable the Backup API",
				Command:     "sdpctl appliance backup api --disable",
			},
		},
	}
	ApplianceUpgradeDoc = CommandDoc{
		Short: "Perform appliance upgrade on the Collective",
		Long: `The upgrade procedure is divided into two parts,
  - prepare: Upload the image new appliance image to the Collective.
  - complete: Install a prepared upgrade on the secondary partition and perform a reboot to make the second partition the primary.

Additional subcommands included are:
  - status: view the current upgrade status on all appliances.
  - cancel: Cancel a prepared upgrade.
`,
	}
	ApplianceUpgradeStatusDoc = CommandDoc{
		Short: "Display the upgrade status of Appliances",
		Long: `Display the upgrade status of Appliances in either table or json format.
Upgrade statuses:
- idle:         No upgrade is initiated
- started:      Upgrade process has started
- downloading:  Appliance is downloading the upgrade image
- verifying:    Upgrade image download is completed and the image is being verified
- ready:        Image is verified and ready to be applied
- installing:   Appliance is installing the upgrade image
- success:      Upgrade successful
- failed:       Upgrade failed for some reason during the process`,
		Examples: []ExampleDoc{
			{
				Description: "view in table format",
				Command:     "sdpctl appliance upgrade status",
			},
			{
				Description: "view in JSON format",
				Command:     "sdpctl appliance upgrade status --json",
			},
			{
				Description: "filtered appliance status list",
				Command:     "sdpctl appliance upgrade status --include=name=controller",
			},
		},
	}
	ApplianceUpgradePrepareDoc = CommandDoc{
		Short: "Prepare the appliances for upgrade",
		Long: `Prepare an upgrade but do NOT install it.
This means the upgrade file will be downloaded/uploaded to all the appliances,
the signature verified as well as any other preconditions applicable at this point.

There are initial checks on the filename before attempting to upload it to the Appliances.
If a local upgrade image is uploaded to the Controller, the only pre-condition is that the filename ends with the file extension '.img.zip'.
If, however, the file is hosted on a server and a URL is provided to the prepare command, the filename should also contain a version, such as 6.0.3.
Otherwise the prepare will fail.

Note that the '--image' flag also accepts URL:s. The Appliances will then attempt to download
the upgrade image using the provided URL. It will fail if the Appliances cannot access the URL.`,
		Examples: []ExampleDoc{
			{
				Description: "prepare an upgrade from a local upgrade image",
				Command:     "sdpctl appliance upgrade prepare --image=/path/to/upgrade-5.5.3.img.zip",
			},
			{
				Description: "prepare an upgrade from remote upgrade image",
				Command:     "sdpctl appliance upgrade prepare --image=https://upgrade-host.com/upgrade-5.5.3.img.zip",
			},
			{
				Description: "use the primary Controller as an upgrade image host for the other appliances",
				Command:     "sdpctl appliance upgrade prepare --image=https://upgrade-host.com/upgrade-5.5.3.img.zip --host-on-controller",
			},
			{
				Description: "prepare only certain appliances based on a filter",
				Command:     "sdpctl appliance upgrade prepare --image=/path/to/image-5.5.3.img.zip --include function=controller",
			},
		},
	}
	ApplianceUpgradeCancelDoc = CommandDoc{
		Short: "Cancel a prepared upgrade on one or more appliances",
		Long: `Cancel a prepared upgrade. The command will attempt to cancel upgrades on
Appliances that are not in the 'idle' upgrade state. Cancelling will remove the
upgrade image from the Appliance, though it will not remove images hosted in the primary
controller file repository (such as when using the '--host-on-controller' flag) by default.
To remove them as well, you can use the '--delete' flag.

Note that you can cancel upgrades on specific appliances by using the '--include' and/or
'--exclude' flags in combination with this command.`,
		Examples: []ExampleDoc{
			{
				Description: "cancel upgrade on all Appliances",
				Command:     "sdpctl appliance upgrade cancel",
			},
			{
				Description: "cancel upgrade on selected Appliances",
				Command:     "sdpctl appliance upgrade cancel --include function=gateway",
			},
			{
				Description: "cancel upgrade and delete all dangling upgrade images",
				Command:     "sdpctl appliance upgrade cancel --delete",
			},
		},
	}
	ApplianceUpgradeCompleteDoc = CommandDoc{
		Short: "Complete the upgrade on prepared appliances",
		Long: `Complete a prepared upgrade.
Install a prepared upgrade on the secondary partition
and perform a reboot to make the second partition the primary.`,
		Examples: []ExampleDoc{
			{
				Description: "complete all pending upgrades",
				Command:     "sdpctl appliance upgrade complete",
			},
			{
				Description: "backup the primary Controller before completing",
				Command:     "sdpctl appliance upgrade complete --backup",
			},
			{
				Description: "backup to custom directory when completing pending upgrade",
				Command:     "sdpctl appliance upgrade complete --backup --backup-destination=/path/to/custom/destination",
			},
		},
	}
	ApplianceMetricsDoc = CommandDoc{
		Short: "Get all the Prometheus metrics for the given Appliance",
		Long: `The 'metric' command will return a list of all the available metrics provided by an appliance for use in Prometheus.
If no appliance ID is given as an argument, the command will prompt for which Appliance you want metrics for. A second argument can be used
to get a specific metric name. This needs to be an exact match.`,
		Examples: []ExampleDoc{
			{
				Description: "list all available appliance metrics",
				Command:     "sdpctl appliance metric",
			},
			{
				Description: "list metrics for a particular appliance",
				Command:     "sdpctl appliance metric <appliance-id>",
			},
			{
				Description: "get a particular metric from an appliance",
				Command:     "sdpctl appliance metric <appliance-id> <metric-name>",
			},
		},
	}
	ApplianceResolveNameDoc = CommandDoc{
		Short: "Test a resolver name on a Gateway",
		Long: `Test a resolver name on a Gateway. Name resolvers are used by the Gateways on a Site resolve
the IPs in the specific network or set of protected resources.`,
		Examples: []ExampleDoc{
			{
				Description: "with a specific gateway appliance id",
				Command:     "sdpctl appliance resolve-name d750ad44-7c6a-416d-773b-f805a2272418 dns://google.se",
			},
			{
				Description: "If you omit appliance id, you will be prompted with all online gateways, and you can select one to test on",
				Command:     "sdpctl appliance resolve-name dns://google.se",
				Output: `? select appliance: gateway-9a9b8b70-faaa-4059-a061-761ce13783ba-site1 - Default Site - []
142.251.36.3
2a00:1450:400e:80f::2003`,
			},
		},
	}
	ApplianceResolveNameStatusDoc = CommandDoc{
		Short: "Get the status of name resolution on a Gateway",
		Long: `Get the status of name resolution on a Gateway. It lists all the subscribed resource names from all the connected
Clients and shows the resolution results`,
		Examples: []ExampleDoc{
			{
				Description: "with a specific gateway appliance id",
				Command:     "sdpctl appliance resolve-name-status 7f340572-0cd3-416b-7755-9f5c4e546391 --json",
				Output: `{
    "resolutions": {
        "aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev": {
            "partial": false,
            "finals": [
                "3.120.51.78",
                "35.156.237.184"
            ],
            "partials": [
                "dns://all.GW-ELB-2001535196.eu-central-1.elb.amazonaws.com",
                "dns://all.purple-lb-1785267452.eu-central-1.elb.amazonaws.com"
            ],
            "errors": []
        }
    }
}`,
			},
		},
	}
	ApplianceStatsDocs = CommandDoc{
		Short: "Show appliance stats",
		Long: `Show current stats, such as current system resource consumption, appliance version etc, for the appliances.
Using the '--json' flag will return a more detailed list of stats in json format.

NOTE: Although the '--include' and '--exclude' flags are provided as options here, they don't have any actual effect on the command.`,
		Examples: []ExampleDoc{
			{
				Description: "default listing of stats",
				Command:     "sdpctl appliance stats",
			},
			{
				Description: "print stats in JSON format",
				Command:     "sdpctl appliance stats --json",
			},
		},
	}

	ApplianceLogsDoc = CommandDoc{
		Short: "Download zip bundle with logs",
		Long:  `Download a zip bundle with all logs from a appliance`,
		Examples: []ExampleDoc{
			{
				Description: "Default logs command",
				Command:     "sdpctl appliance logs",
			},
			{
				Description: "Default logs to specific path",
				Command:     "sdpctl appliance logs a68c1f69-534d-4060-a052-b223d42bac2c --path /tmp",
			},
		},
	}

	ApplianceExtractLogsDoc = CommandDoc{
		Short: "Unpacks binary log files from a log bundle",
		Long:  `Unpacks journald binary log files from a log bundle .zip file, creating log files grouped by daemon`,
		Examples: []ExampleDoc{
			{
				Description: "Extract logs command",
				Command:     "sdpctl extract-logs controller.zip",
			},
			{
				Description: "Extract logs to specific path",
				Command:     "sdpctl extract-logs controller.zip --path /tmp",
			},
		},
	}

	ApplianceSeedDocs = CommandDoc{
		Short: "Export seed for an inactive Appliance",
		Long: `Generate a seed file in JSON (or iso format)


for More information, see: https://sdphelp.appgate.com/adminguide/new-appliance.html
        `,
		Examples: []ExampleDoc{
			{
				Description: "export seed file in JSON format with cloud authentication",
				Command:     "sdpctl appliance export-seed 08cd20c0-f175-4503-96f7-c5b429c19236 --provide-cloud-ssh-key",
			},
			{
				Description: "export seed file in iso format with passphrase",
				Command:     `echo "YourSuperSecretPassword" | sdpctl appliance export-seed 08cd20c0-f175-4503-96f7-c5b429c19236 --iso-format`,
			},
			{
				Description: "Interactive prompt to configure the seed file",
				Command:     "sdpctl appliance export-seed",
			},
		},
	}

	ApplianceForceDisableControllerDocs = CommandDoc{
		Short: "Force disable misbehaving Controllers using this command. USE WITH CAUTION!",
		Long: `Force disable Controllers that are misbehaving in any way. This will send a disable command to the primary Controller, which will notify the remaining Controllers of the change.
The command will accept one or more hostnames or ID:s of Controllers that will be disabled as an argument. You can get the hostnames by running 'sdpctl appliance stats'.`,
		Examples: []ExampleDoc{
			{
				Description: "force disable a Controller with the hostname 'failedcontroller.example.com'",
				Command:     "sdpctl appliance force-disable-controller failedcontroller.example.com",
			},
			{
				Description: "force disable multiple controllers",
				Command:     "sdpctl appliance force-disable-controller failed1.example.com failed2.example.com",
			},
			{
				Description: "force disable using ID:s",
				Command:     "sdpctl appliance force-disable-controller f905ff0b-91a6-4d12-afbe-f9a9506f02da",
			},
			{
				Description: "using the command without arguments will prompt for which controllers to disable",
				Command:     "sdpctl appliance force-disable-controller",
				Output: `? Select Controllers to force disable  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
[ ] failed-controller-1 (failed1.example.com)
[ ] failed-controller-2 (failed2.example.com)
[ ] offline-controller (offline.example.com) [OFFLINE]`,
			},
		},
	}
	ApplianceFunctionsDownloadDocs = CommandDoc{
		Short: "Download functions as container bundles",
		Long:  "Download appliance functions as container bundles. The container bundles can then be uploaded to the SDP Collective to enable the function contained in the bunde. Note that currently only the LogServer function is available for container bundle download.",
		Examples: []ExampleDoc{
			{
				Command:     "sdpctl appliance functions download LogServer",
				Description: "download the LogServer function as a bundle",
			},
			{
				Command:     "sdpctl appliance functions download LogServer --save-path=<download-path>",
				Description: "Save the container bundles in a custom path. This is expected to be a directory. If the directory does not exist, sdpctl will try to create it.",
			},
			{
				Command:     "sdpctl appliance functions download LogServer --docker-registry=<path-to-custom-docker-registry>",
				Description: "Download the functions from a custom docker registry.",
			},
		},
	}
	ApplianceSwitchPartitionDocs = CommandDoc{
		Short: "Initiate a partition switch on an appliance regardless of upgrade status",
		Long:  "",
		Examples: []ExampleDoc{
			{
				Command:     "sdpctl appliance switch-partition <appliance-id>",
				Description: "executing the command with a valid appliance id will initiate the partition switch on the appliance with the matching id",
				Output:      ``,
			},
			{
				Command:     "sdpctl appliance switch-partition",
				Description: "executing the command with no argument will prompt for the appliance to make the partition switch on",
				Output: `? select appliance:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
[ ] controller
[ ] gateway`,
			},
		},
	}
)
