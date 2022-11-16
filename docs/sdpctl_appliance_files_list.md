## sdpctl appliance files list

lists the files in the controllers file repository

### Synopsis

Lists the files in the controllers file repository. Default output is in table format.
Optionally print the output in JSON format by using the "--json" flag

```
sdpctl appliance files list [flags]
```

### Examples

```
  # list files table output
  > sdctl files list
  Name                                Status    Created                                 Modified                                Failure Reason
  ----                                ------    -------                                 --------                                --------------
  appgate-6.0.1-29983-beta.img.zip    Ready     2022-08-19 08:06:20.909002 +0000 UTC    2022-08-19 08:06:20.909002 +0000 UTC

  # list files using JSON output
  > sdctl files list --json
  [
    {
      "creationTime": "2022-08-19T08:06:20.909002Z",
      "lastModifiedTime": "2022-08-19T08:06:20.909002Z",
      "name": "appgate-6.0.1-29983-beta.img.zip",
      "status": "Ready"
    }
  ]
```

### Options

```
  -h, --help   help for list
      --json   output in json format
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

* [sdpctl appliance files](sdpctl_appliance_files.md)	 - The files command lets you manage the file repository on the connected Controller

