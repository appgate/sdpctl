## sdpctl appliance maintenance status



```
sdpctl appliance maintenance status [flags]
```

### Options

```
  -h, --help   help for status
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

