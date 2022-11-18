## sdpctl token

Perform actions on Admin, Claims and Entitlement tokens

### Synopsis

The token command allows you to renew or revoke tokens used in the Collective.

### Options

```
  -h, --help   help for token
      --json   Display in JSON format
```

### Options inherited from parent commands

```
      --api-version int   Peer API version override
      --ci-mode           Log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --no-interactive    Suppress interactive prompt with auto accept
      --no-verify         Don't verify TLS on for the given command, overriding settings from config file
  -p, --profile string    Profile configuration to use
```

### SEE ALSO

* [sdpctl](sdpctl.md)	 - sdpctl is a command line tool to manage Appgate SDP Collectives
* [sdpctl token list](sdpctl_token_list.md)	 - List distinguished names of active devices
* [sdpctl token revoke](sdpctl_token_revoke.md)	 - Revoke entitlement tokens by distinguished name or token-type

