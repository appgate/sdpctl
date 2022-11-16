## sdpctl appliance upgrade complete

complete the upgrade on prepared appliances

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

  # backup primary controller before completing
  > sdpctl appliance upgrade complete --backup

  # backup to custom directory when completing pending upgrade
  > sdpctl appliance upgrade complete --backup --backup-destination=/path/to/custom/destination
```

### Options

```
      --actual-hostname string      If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname.
  -b, --backup                      backup primary controller before completing upgrade (default true)
      --backup-destination string   specify path to download backup (default "/home/larskajes/Downloads/appgate/backup")
      --batch-size int              number of batch groups (default 2)
  -h, --help                        help for complete
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
  -t, --timeout duration         Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on. (default 30m0s)
```

### SEE ALSO

* [sdpctl appliance upgrade](sdpctl_appliance_upgrade.md)	 - Perform appliance upgrade on the Appgate SDP Collective

