## sdpctl appliance logs

download zip bundle with logs

### Synopsis

download zip bundle with logs

```
sdpctl appliance logs [flags]
```

### Examples

```
  # Default logs command
  > sdpctl appliance logs

  # Default logs to specific path
  > sdpctl appliance logs a68c1f69-534d-4060-a052-b223d42bac2c --path /tmp
```

### Options

```
  -h, --help          help for logs
      --json          Display in JSON format
      --path string   Optional path to write to
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

