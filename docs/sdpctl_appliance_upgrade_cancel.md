## sdpctl appliance upgrade cancel

Cancel a prepared upgrade on one or more appliances

### Synopsis

Cancel a prepared upgrade. The command will attempt to cancel upgrades on
Appliances that are not in the 'idle' upgrade state. Cancelling will remove the
upgrade image from the Appliance, though it will not remove images hosted in the primary
controller file repository (such as when using the '--host-on-controller' flag) by default.
To remove them as well, you can use the '--delete' flag.

Note that you can cancel upgrades on specific appliances by using the '--include' and/or
'--exclude' flags in combination with this command.

```
sdpctl appliance upgrade cancel [flags]
```

### Examples

```
  # cancel upgrade on all Appgate SDP Appliances
  > sdpctl appliance upgrade cancel

  # cancel upgrade on selected Appgate SDP Appliances
  > sdpctl appliance upgrade cancel --include function=gateway

  # cancel upgrade and delete all dangling upgrade images
  > sdpctl appliance upgrade cancel --delete
```

### Options

```
      --delete           Delete all upgrade files from the controller
  -h, --help             help for cancel
      --no-interactive   suppress interactive prompt with auto accept
```

### Options inherited from parent commands

```
      --api-version int          peer API version override
      --ci-mode                  log to stderr instead of file and disable progress-bars
      --debug                    Enable debug logging
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
      --no-verify                don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string           profile configuration to use
  -t, --timeout duration         Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on. (default 30m0s)
```

### SEE ALSO

* [sdpctl appliance upgrade](sdpctl_appliance_upgrade.md)	 - Perform appliance upgrade on the Appgate SDP Collective

