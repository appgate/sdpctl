## sdpctl token list

list distinguished names of active devices

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
      --api-version int   peer API version override
      --ci-mode           log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --json              Display in JSON format
      --no-interactive    suppress interactive prompt with auto accept
      --no-verify         don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string    profile configuration to use
```

### SEE ALSO

* [sdpctl token](sdpctl_token.md)	 - Perform actions related to token on the Appgate SDP Collective

