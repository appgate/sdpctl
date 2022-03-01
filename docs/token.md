# The `token` command
The token command let's you list and revoke device tokens registered in the Appgate SDP collective. You can revoke single tokens or batches of tokens using the provided flags (see below).

## Flags
| Flag | Type | Description | Default |
|---|---|---|---|
| `--json` | none | Will display output in JSON format | false |

## Actions
- [list](#action-list)
- [revoke](#action-revoke)

## Action: list
```bash
$ sdpctl token list
```

## Action: revoke
### Flags
| Flag | Type | Description | Default |
|---|---|---|---|
| `--by-token-type` | string | Revoke all tokens by type { administration, adminclaims, entitlements, claims } | null |
| `--reason` | string | Add a reason for revoking token(s) | null |
| `--delay-minutes` | int32 | Delay time for revoking token(s) in minutes | 5 |
| `--per-second` | float32 | Tokens are revoked in batches according to this value to spread load on the controller | 7 |
| `--site-id` | string | revoke only tokens for the given site ID | null |
| `--specific-distinguished-names` | strings | comma-separated string of distinguished names to renew tokens in bulk for a specific list of devices | null |
| `--token-type` | string | Revoke only certain types of token when revoking by distinguished name { administration, adminclaims, entitlements, claims } | null |
