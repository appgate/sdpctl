## sdpctl appliance maintenance

Manually mange maintenance mode on Controllers

### Options

```
  -h, --help   help for maintenance
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
* [sdpctl appliance maintenance disable](sdpctl_appliance_maintenance_disable.md)	 - Disable maintenance mode on a single Controller
* [sdpctl appliance maintenance enable](sdpctl_appliance_maintenance_enable.md)	 - Enable maintenance mode on a single Controller
* [sdpctl appliance maintenance status](sdpctl_appliance_maintenance_status.md)	 - 

