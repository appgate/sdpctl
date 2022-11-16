## sdpctl appliance backup api

Controls the state of the backup API.

### Synopsis

This command controls the state of the backup API on the Appgate SDP Collective.
You will be prompted for a passphrase for the backups when enabling the backup API using this command.
The passphrase is required.

```
sdpctl appliance backup api [flags]
```

### Examples

```
  # enable the backup API
  > appgate appliance backup api

  # disable the backup API
  > sdpctl appliance backup api --disable
```

### Options

```
      --disable   Disable the backup API
  -h, --help      help for api
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

* [sdpctl appliance backup](sdpctl_appliance_backup.md)	 - Perform backup of the Appgate SDP Collective appliances

