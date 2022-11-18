## sdpctl service-users update

Update a service user

```
sdpctl service-users update [id] [args...] [flags]
```

### Examples

```
  # update the name of a service user with the id of <id>
  > sdpctl service-users update <id> name <new-name>

  # set a new passphrase for service user with id of <id>
  > sdpctl service-users update <id> passphrase <new-passphrase>

  # disable a service user with id of <id>
  > sdpctl service-users update <id> disable

  # enable a service user with id of <id>
  > sdpctl service-users update <id> enable

  # add a tag for a service user
  > sdpctl service-users update <id> add tag <new-tag>

  # add a label for a service user
  > sdpctl service-users update <id> add label <key>=<value>

  # remove a tag for a service user
  > sdpctl service-users update <id> remove tag <tag>

  # remove a label for a service user
  > sdpctl service-users update <id> remove label <key>

  # update a service user using a predefined JSON file
  > sdpctl service-users update <id> --from-file=<path-to-json-file>

  # update multiple values of a service user
  > sdpctl service-users update <id> '{"name": "<new-name>", "disabled": true}'
```

### Options

```
  -f, --from-file string   update service user with values using a valid json file
  -h, --help               help for update
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

