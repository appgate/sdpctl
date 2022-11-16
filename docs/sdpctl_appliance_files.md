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
* [sdpctl appliance files delete](sdpctl_appliance_files_delete.md)	 - delete files from the repository
* [sdpctl appliance files list](sdpctl_appliance_files_list.md)	 - lists the files in the controllers file repository

