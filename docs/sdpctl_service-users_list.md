## sdpctl service-users list

List service users in the Collective

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
      --api-version int   Peer API version override
      --ci-mode           Log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --json              output in json format
      --no-interactive    Suppress interactive prompt with auto accept
      --no-verify         Don't verify TLS on for the given command, overriding settings from config file
  -p, --profile string    Profile configuration to use
```

### SEE ALSO

* [sdpctl service-users](sdpctl_service-users.md)	 - Manage Service Users

