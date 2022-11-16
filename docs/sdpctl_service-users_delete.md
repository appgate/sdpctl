## sdpctl service-users delete

delete one or more service user(s)

```
sdpctl service-users delete [id...] [flags]
```

### Examples

```
  # delete a service user with the id of <id>
  > sdpctl service-users delete <id>

  # delete multiple service users by providing multiple id:s
  > sdpctl service-users delete <id1> <id2>

  # delete service user(s) using prompt
  > sdpctl service-users delete
```

### Options

```
  -h, --help   help for delete
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

