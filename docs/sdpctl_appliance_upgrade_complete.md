## sdpctl appliance upgrade complete

Complete the upgrade on prepared appliances

### Synopsis

Complete a prepared upgrade.
Install a prepared upgrade on the secondary partition
and perform a reboot to make the second partition the primary.

```
sdpctl appliance upgrade complete [flags]
```

### Examples

```
  # complete all pending upgrades
  > sdpctl appliance upgrade complete

  # backup the primary Controller before completing
  > sdpctl appliance upgrade complete --backup

  # backup to custom directory when completing pending upgrade
  > sdpctl appliance upgrade complete --backup --backup-destination=/path/to/custom/destination
```

### Options

```
      --actual-hostname string      If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname
  -b, --backup                      Backup primary Controller before completing the upgrade (default true)
      --backup-destination string   Specify path to download backup (default "$HOME/Downloads/appgate/backup")
      --batch-size int              Number of batch groups (default 2)
  -h, --help                        help for complete
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
  -t, --timeout duration         Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on (default 30m0s)
```

### SEE ALSO

* [sdpctl appliance upgrade](sdpctl_appliance_upgrade.md)	 - Perform appliance upgrade on the Collective

