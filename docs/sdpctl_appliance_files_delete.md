## sdpctl appliance files delete

Delete files from the repository

### Synopsis

Delete files from the repository with this command. There are multiple options on which file(s) should be deleted.

```
sdpctl appliance files delete [flags]
```

### Examples

```
  # delete a single file using the filename as a parameter
  > sdpctl appliance files delete file-to-delete.img.zip
  file-to-delete.img.zip: deleted

  # delete all files in the repository
  > sdpctl appliance files delete --all
  deleted1.img.zip: deleted
  deleted2.img.zip: deleted

  # no arguments will prompt for which files to delete
  > sdpctl appliance files delete
  ? select files to delete:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
  > [ ]  file1.img.zip
    [ ]  file2.img.zip
    [ ]  file3.img.zip
  
```

### Options

```
      --all    delete all files from repository
  -h, --help   help for delete
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

* [sdpctl appliance files](sdpctl_appliance_files.md)	 - The files command lets you manage the file repository on the connected Controller

