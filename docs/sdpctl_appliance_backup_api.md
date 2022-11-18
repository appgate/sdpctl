## sdpctl appliance backup api

Controls the state of the Backup API

### Synopsis

This command controls the state of the Backup API on the Collective.
You will be prompted for a passphrase for the backups when enabling the Backup API using this command.
The passphrase is required.

```
sdpctl appliance backup api [flags]
```

### Examples

```
  # enable the Backup API
  > appgate appliance backup api

  # disable the Backup API
  > sdpctl appliance backup api --disable
```

### Options

```
      --disable   Disable the Backup API
  -h, --help      help for api
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

* [sdpctl appliance backup](sdpctl_appliance_backup.md)	 - Perform backup of the appliances

