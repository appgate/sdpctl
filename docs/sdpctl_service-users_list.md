## sdpctl service-users list

list service users in the Appgate SDP Collective

```
sdpctl service-users list [flags]
```

### Examples

```
  # list service users
  > sdpctl service-users list

  # output in JSON format
  > sdpctl service-users list --json
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --api-version int   peer API version override
      --ci-mode           log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --json              output in json format
      --no-interactive    suppress interactive prompt with auto accept
      --no-verify         don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string    profile configuration to use
```

### SEE ALSO

* [sdpctl service-users](sdpctl_service-users.md)	 - root command for managing service users in the Appgate SDP Collective

