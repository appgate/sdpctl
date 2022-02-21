# The `token` command

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
| Flag | Shorthand | Description | Default |
|---|---|---|---|
| `--by-token-type` | string | Add description | null |
| `--reason` | string | Add description | null |
| `--delay-minutes` | int32 | Add description | null |
| `--per-second` | float32 | Add description | null |
| `--site-id` | string | Add description | null |
| `--specific-distinguished-names` | strings | Add description | null |
| `--token-type` | string | Add description | null |
