## sdpctl appliance upgrade

Perform appliance upgrade on the Collective

### Synopsis

The upgrade procedure is divided into two parts,
  - prepare: Upload the image new appliance image to the Collective.
  - complete: Install a prepared upgrade on the secondary partition and perform a reboot to make the second partition the primary.

Additional subcommands included are:
  - status: view the current upgrade status on all appliances.
  - cancel: Cancel a prepared upgrade.


### Options

```
  -h, --help               help for upgrade
  -t, --timeout duration   Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on (default 30m0s)
```

### Options inherited from parent commands

```
      --api-version int          Peer API version override
      --ci-mode                  Log to stderr instead of file and disable progress-bars
      --debug                    Enable debug logging
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
      --no-interactive           Suppress interactive prompt with auto accept
      --no-verify                Don't verify TLS on for the given command, overriding settings from config file
  -p, --profile string           Profile configuration to use
```

### SEE ALSO

* [sdpctl appliance](sdpctl_appliance.md)	 - Manage the appliances and perform tasks such as backups, ugprades, metrics etc
* [sdpctl appliance upgrade cancel](sdpctl_appliance_upgrade_cancel.md)	 - Cancel a prepared upgrade on one or more appliances
* [sdpctl appliance upgrade complete](sdpctl_appliance_upgrade_complete.md)	 - Complete the upgrade on prepared appliances
* [sdpctl appliance upgrade prepare](sdpctl_appliance_upgrade_prepare.md)	 - Prepare the appliances for upgrade
* [sdpctl appliance upgrade status](sdpctl_appliance_upgrade_status.md)	 - Display the upgrade status of Appliances

