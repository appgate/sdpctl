## sdpctl appliance files

The files command lets you manage the file repository on the connected Controller

### Synopsis

The files command lets you manage the file repository on the currently connected Controller.

### Options

```
  -h, --help   help for files
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
* [sdpctl appliance files delete](sdpctl_appliance_files_delete.md)	 - Delete files from the repository
* [sdpctl appliance files list](sdpctl_appliance_files_list.md)	 - Lists the files in the Controllers file repository

