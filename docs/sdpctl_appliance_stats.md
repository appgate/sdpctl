## sdpctl appliance stats

show Appgate SDP Appliance stats

### Synopsis

Show current stats, such as current system resource consumption, Appliance version etc, for the Appgate SDP Appliances.
Using the '--json' flag will return a more detailed list of stats in json format.

NOTE: Although the '--include' and '--exclude' flags are provided as options here, they don't have any actual effect on the command.

```
sdpctl appliance stats [flags]
```

### Examples

```
  # default listing of stats
  > sdpctl appliance stats

  # print stats in JSON format
  > sdpctl appliance stats --json
```

### Options

```
  -h, --help   help for stats
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

* [sdpctl appliance](sdpctl_appliance.md)	 - interact with Appgate SDP Appliances

