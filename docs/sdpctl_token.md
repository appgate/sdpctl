## sdpctl token

Perform actions related to token on the Appgate SDP Collective

### Synopsis

The token command allows you to renew or revoke device tokens used in the Appgate SDP Collective.

### Options

```
  -h, --help   help for token
      --json   Display in JSON format
```

### Options inherited from parent commands

```
      --api-version int   peer API version override
      --ci-mode           log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --no-interactive    suppress interactive prompt with auto accept
      --no-verify         don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string    profile configuration to use
```

### SEE ALSO

* [sdpctl](sdpctl.md)	 - sdpctl is a command line tool to control and handle Appgate SDP using the CLI
* [sdpctl token list](sdpctl_token_list.md)	 - list distinguished names of active devices
* [sdpctl token revoke](sdpctl_token_revoke.md)	 - revoke entitlement tokens by distinguished name or token-type

