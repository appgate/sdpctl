## sdpctl appliance list

List all appliances

### Synopsis

List all appliances

```
sdpctl appliance list [flags]
```

### Examples

```
  # Default list command
  > sdpctl appliance list
  Name                                                   ID                                    Hostname                 Site          Activated
  ----                                                   --                                    --------                 ----          ---------
  controller                                             67f7ee0c-924c-4253-8b78-0882ff0665ab  controller.dev           Default Site  true
  gateway                                                ec3b6270-ad7e-447a-a6e6-8f4ae816cab5  gateway.dev              Default Site  true

  # Print list of appliances in json format
  > sdpctl appliance list --json

  # Print a filtered list of appliances
  > sdpctl appliance list --include=<key>=<value>
```

### Options

```
  -h, --help   help for list
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
```

### SEE ALSO

* [sdpctl appliance](sdpctl_appliance.md)	 - Manage the appliances and perform tasks such as backups, ugprades, metrics etc

