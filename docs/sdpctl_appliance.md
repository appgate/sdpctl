## sdpctl appliance

Manage the appliances and perform tasks such as backups, ugprades, metrics etc

### Synopsis

The base command to manage the appliances. This command does not do anything by itself, it is
used together with one of the available sub-commands listed below.

### Options

```
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -h, --help                     help for appliance
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
```

### Options inherited from parent commands

```
      --api-version int   Peer API version override
      --ci-mode           Log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --no-interactive    Suppress interactive prompt with auto accept
      --no-verify         Don't verify TLS on for the given command, overriding settings from config file
  -p, --profile string    Profile configuration to use
```

### SEE ALSO

* [sdpctl](sdpctl.md)	 - sdpctl is a command line tool to manage Appgate SDP Collectives
* [sdpctl appliance backup](sdpctl_appliance_backup.md)	 - Perform backup of the appliances
* [sdpctl appliance files](sdpctl_appliance_files.md)	 - The files command lets you manage the file repository on the connected Controller
* [sdpctl appliance list](sdpctl_appliance_list.md)	 - List all appliances
* [sdpctl appliance logs](sdpctl_appliance_logs.md)	 - Download zip bundle with logs
* [sdpctl appliance maintenance](sdpctl_appliance_maintenance.md)	 - Manually mange maintenance mode on Controllers
* [sdpctl appliance metric](sdpctl_appliance_metric.md)	 - Get all the Prometheus metrics for the given Appliance
* [sdpctl appliance resolve-name](sdpctl_appliance_resolve-name.md)	 - Test a resolver name on a Gateway
* [sdpctl appliance resolve-name-status](sdpctl_appliance_resolve-name-status.md)	 - Get the status of name resolution on a Gateway
* [sdpctl appliance stats](sdpctl_appliance_stats.md)	 - Show appliance stats
* [sdpctl appliance upgrade](sdpctl_appliance_upgrade.md)	 - Perform appliance upgrade on the Collective

