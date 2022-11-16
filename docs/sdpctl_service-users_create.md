## sdpctl service-users create

create a new service user

```
sdpctl service-users create [flags]
```

### Examples

```
  # create a new service user
  > sdpctl service-users create
  ? Name for service user: <service-user-name>
  ? Passphrase for service user: <service-user-passphrase>
  ? Confirm your passphrase: <confirm-passphrase>

  # create service user with flag input
  > echo "<passphrase>" | sdpctl service-users create --name=<service-user-name>

  # create a service user from a valid JSON file
  > sdpctl service-users create --from-file=<path-to-json-file>
```

### Options

```
  -f, --from-file string   create a user from a valid json file
  -h, --help               help for create
      --name string        name for service user
      --tags strings       tags for service user
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

