## sdpctl appliance upgrade status

Display the upgrade status of Appliances

### Synopsis

Display the upgrade status of Appliances in either table or json format.
Upgrade statuses:
- idle:         No upgrade is initiated
- started:      Upgrade process has started
- downloading:  Appliance is downloading the upgrade image
- verifying:    Upgrade image download is completed and the image is being verified
- ready:        Image is verified and ready to be applied
- installing:   Appliance is installing the upgrade image
- success:      Upgrade successful
- failed:       Upgrade failed for some reason during the process

```
sdpctl appliance upgrade status [flags]
```

### Examples

```
  # view in table format
  > sdpctl appliance upgrade status

  # view in JSON format
  > sdpctl appliance upgrade status --json

  # filtered appliance status list
  > sdpctl appliance upgrade status --include=name=controller
```

### Options

```
  -h, --help   help for status
      --json   Display in JSON format
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

