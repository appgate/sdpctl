## sdpctl appliance maintenance disable

Disable maintenance mode on a single controller

### Synopsis


    For advanced users only.
    enabling maintenance mode on the primary controller can leave you with a unreachable environment.

    USE WITH CAUTION

    arguments are Optional, if no arguments are provided, you will be prompted
    with an interactive prompt.
    

```
sdpctl appliance maintenance disable <applianceUUID> [flags]
```

### Examples

```
  # Disable maintenance mode on a fixed controller UUID
  > sdpctl appliance maintenance disable 20e75a08-96c6-4ea3-833e-cdbac346e2ae
  Change result: success
  Change Status: completed

  # Disable maintenance mode interactive prompt
  > sdpctl appliance maintenance disable
  
  ? select appliance: controller-two - Default Site - []
  
  A Controller in maintenance mode will not accept any API calls besides disabling maintenance mode. Starting in version 6.0, clients will still function as usual while a Controller is in maintenance mode.
  
  ? Are you really sure you want to disable maintenance mode on controller-two?
  
  Do you want to continue? Yes
  Change result: success
  Change Status: completed
                  
```

### Options

```
  -h, --help   help for disable
      --json   Display in JSON format
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

* [sdpctl appliance maintenance](sdpctl_appliance_maintenance.md)	 - Manually mange maintenance mode on controllers

