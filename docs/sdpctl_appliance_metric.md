## sdpctl appliance metric

Get all the Prometheus metrics for the given Appgate SDP Appliance

### Synopsis

The 'metric' command will return a list of all the available metrics provided by an Appgate SDP Appliance for use in Prometheus.
If no Appliance ID is given as an argument, the command will prompt for which Appliance you want metrics for. A second argument can be used
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

