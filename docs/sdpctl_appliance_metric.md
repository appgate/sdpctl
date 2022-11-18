## sdpctl appliance metric

Get all the Prometheus metrics for the given Appliance

### Synopsis

The 'metric' command will return a list of all the available metrics provided by an appliance for use in Prometheus.
If no appliance ID is given as an argument, the command will prompt for which Appliance you want metrics for. A second argument can be used
to get a specific metric name. This needs to be an exact match.

```
sdpctl appliance metric [<appliance-id>] [<metric-name>] [flags]
```

### Examples

```
  # list all available appliance metrics
  > sdpctl appliance metric

  # list metrics for a particular appliance
  > sdpctl appliance metric <appliance-id>

  # get a particular metric from an appliance
  > sdpctl appliance metric <appliance-id> <metric-name>
```

### Options

```
  -h, --help   help for metric
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

