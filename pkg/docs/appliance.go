package docs

var (
	ApplianceRootDoc = CommandDoc{
		Short: "interact with Appgate SDP Appliances",
		Long: `The base command to access and interact with your Appgate SDP Appliances. This command does not do anything by itself, it is
used together with one of the available sub-commands listed below.`,
	}
	ApplianceListDoc = CommandDoc{
		Short: "List all Appgate SDP Appliances",
		Long: `List all Appliances in the Appgate SDP Collective. The appliances will be listed in no particular order. Using without arguments
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
		Short: "Perform backup of the Appgate SDP Collective appliances",
		Long: `The backup command will request a backup from the API and download them to a destination directory. The command requires the backup API to be enabled in
the Appgate SDP Collective. In case the backup API is not enabled when executing the backup command, you will be prompted to activate it.

There are multiple options for selecting which Appgate SDP Appliances to backup, using flags or optional arguments. The arguments are expected to be the name of
the Appgate SDP Appliance you want to take a backup of.

The default destination directory is set to be the users default downloads directory on the system. If the default destination is used, an 'appgate' directory
will be created there if it doesn't already exist and the backups will be downloaded to that. In case custom destination directory is specified by using the
'--destination' flag, the extra 'appgate' directory will not be created. The user also has to have write privileges on the specified directory.

For more information on the backup process, go to: https://sdphelp.appgate.com/adminguide/v5.5/backup-script.html`,
		Examples: []ExampleDoc{
			{
				Description: "Backup with no arguments or flags will prompt for appliance",
				Command:     "sdpctl appliance backup",
				Output: `? Backup API is disabled on the appliance. Do you want to enable it now? Yes
? The passphrase to encrypt Appliance Backups when backup API is used: <password> # only shows if backup API is not enabled
? Confirm your passphrase: <password> # only shows if backup API is not enabled
? select appliances to backup:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [ ]  controller
  [ ]  gateway`,
			},
			{
				Description: "download backups to a custom directory",
				Command:     "sdpctl appliance backup --destination=path/to/backup/destination",
			},
			{
				Description: "backup only primary controller using flag",
				Command:     "sdpctl appliance backup --primary",
			},
			{
				Description: "backup all Appgate SDP Appliances",
				Command:     "sdpctl appliance backup --all",
			},
			{
				Description: "backup using '--include' and '--exclude' flags",
				Command:     "sdpctl appliance backup --include=function=controller --exclude=tag=secondary",
			},
		},
	}
	ApplianceBackupAPIDoc = CommandDoc{
		Short: "Controls the state of the backup API.",
		Long: `This command controls the state of the backup API on the Appgate SDP Collective.
You will be prompted for a passphrase for the backups when enabling the backup API using this command.
The passphrase is required.`,
		Examples: []ExampleDoc{
			{
				Description: "enable the backup API",
				Command:     "appgate appliance backup api",
			},
			{
				Description: "disable the backup API",
				Command:     "sdpctl appliance backup api --disable",
			},
		},
	}
	ApplianceUpgradeDoc = CommandDoc{
		Short: "Perform appliance upgrade on the Appgate SDP Collective",
		Long: `The upgrade procedure is divided into two parts,
  - prepare: Upload the image new appliance image to the Appgate SDP Collective.
  - complete: Install a prepared upgrade on the secondary partition and perform a reboot to make the second partition the primary.

Additional subcommands included are:
  - status: view the current upgrade status on all appliances.
  - cancel: Cancel a prepared upgrade.
`,
	}
	ApplianceUpgradeStatusDoc = CommandDoc{
		Short: "Display the upgrade status of Appgate SDP Appliances",
		Long: `Display the upgrade status of Appgate SDP Appliances in either table or json format.
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
		Short: "prepare appliances for upgrade",
		Long: `Prepare an upgrade but do NOT install it.
This means the upgrade file will be downloaded/uploaded to all the appliances,
the signature verified as well as any other preconditions applicable at this point.

There are initial checks on the filename before attempting to upload it to the Appliances.
A valid filename ends with '.img.zip' and also needs to have a semver included somewhere
in the name, eg. 'upgrade.img.zip' will not not be valid, but 'upgrade5.5.3.img.zip' is
considered valid.

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
				Description: "use primary controller as an upgrade image host for the other appliances",
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
				Description: "cancel upgrade on all Appgate SDP Appliances",
				Command:     "sdpctl appliance upgrade cancel",
			},
			{
				Description: "cancel upgrade on selected Appgate SDP Appliances",
				Command:     "sdpctl appliance upgrade cancel --include function=gateway",
			},
			{
				Description: "cancel upgrade and delete all dangling upgrade images",
				Command:     "sdpctl appliance upgrade cancel --delete",
			},
		},
	}
	ApplianceUpgradeCompleteDoc = CommandDoc{
		Short: "complete the upgrade on prepared appliances",
		Long: `Complete a prepared upgrade.
Install a prepared upgrade on the secondary partition
and perform a reboot to make the second partition the primary.`,
		Examples: []ExampleDoc{
			{
				Description: "complete all pending upgrades",
				Command:     "sdpctl appliance upgrade complete",
			},
			{
				Description: "backup primary controller before completing",
				Command:     "sdpctl appliance upgrade complete --backup",
			},
			{
				Description: "backup to custom directory when completing pending upgrade",
				Command:     "sdpctl appliance upgrade complete --backup --backup-destination=/path/to/custom/destination",
			},
		},
	}
	ApplianceMetricsDoc = CommandDoc{
		Short: "Get all the Prometheus metrics for the given Appgate SDP Appliance",
		Long: `The 'metric' command will return a list of all the available metrics provided by an Appgate SDP Appliance for use in Prometheus.
If no Appliance ID is given as an argument, the command will prompt for which Appliance you want metrics for. The '--metric-name' flag can be used
to get a specific metric name. This needs to be an exact match.

NOTE: Although the '--include' and '--exclude' flags are provided as options here, they don't have any actual effect on the command.`,
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
				Command:     "sdpctl appliance metric <appliance-id> --metric-name=<some_metric_name>",
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
				Command:     "sdpctl appliance resolve-name d750ad44-7c6a-416d-773b-f805a2272418 --resource-name dns://google.se",
			},
			{
				Description: "If you omit appliance id, you will be prompted with all online gateways, and you can select one to test on.",
				Command:     "sdpctl appliance resolve-name --resource-name dns://google.se",
				Output: `? select appliance: gateway-9a9b8b70-faaa-4059-a061-761ce13783ba-site1 - Default Site - []
142.251.36.3
2a00:1450:400e:80f::2003`,
			},
		},
	}
	ApplianceResolveNameStatusDoc = CommandDoc{
		Short: "Get the status of name resolution on a Gateway.",
		Long: `Get the status of name resolution on a Gateway. It lists all the subscribed resource names from all the connected
Clients and shows the resolution results.`,
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
		Short: "show Appgate SDP Appliance stats",
		Long: `Show current stats, such as current system resource consumption, Appliance version etc, for the Appgate SDP Appliances.
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
)
