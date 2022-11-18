## sdpctl token revoke

Revoke entitlement tokens by distinguished name or token-type

### Synopsis

Revoke tokens by distinguished name or token type.

Valid token types are:
  - administration
  - adminclaims
  - entitlements
  - claims

```
sdpctl token revoke [<distinguished-name> | --by-token-type <type>] [flags]
```

### Examples

```
  # revoke by distinguished name
  > sdpctl token revoke <distinguished-name>

  # revoke by token type
  > sdpctl token revoke --token-type=claims
```

### Options

```
      --by-token-type string                   Revoke all tokens of this type. { administration, adminclaims, entitlements, claims }
      --delay-minutes int32                    Delay time for token revocations in minutes. defaults to 5 minutes (default 5)
  -h, --help                                   help for revoke
      --per-second float32                     Tokens are revoked in batches according to this value to spread load on the Controller. defaults to 7 token per second (default 7)
      --reason string                          Reason for revocation
      --site-id string                         Revoke only tokens for the given site ID
      --specific-distinguished-names strings   Comma-separated string of distinguished names to renew tokens in bulk for a specific list of devices
      --token-type string                      Revoke only certain types of token when revoking by distinguished name
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

