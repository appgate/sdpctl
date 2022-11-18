## sdpctl token list

List distinguished names of active devices

### Synopsis

List distinguished names of active tokens, either in table format or JSON format using the '--json' flag

```
sdpctl token list [flags]
```

### Examples

```
  # default list command
  > sdpctl token list

  # print list in JSON format
  > sdpctl token list --json
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
      --json              Display in JSON format
      --no-interactive    Suppress interactive prompt with auto accept
      --no-verify         Don't verify TLS on for the given command, overriding settings from config file
  -p, --profile string    Profile configuration to use
```

### SEE ALSO

* [sdpctl token](sdpctl_token.md)	 - Perform actions on Admin, Claims and Entitlement tokens

