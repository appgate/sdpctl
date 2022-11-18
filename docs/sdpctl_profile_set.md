## sdpctl profile set

Set which admin profile to use

```
sdpctl profile set [flags]
```

### Examples

```
  # Set admin profile without any arguments
  > sdpctl profile set
  ? select profile:  [Use arrows to move, type to filter]
  > production
    staging
    testing

  # set production as your current admin profile
  > sdpctl profile set production
```

### Options

```
  -h, --help   help for set
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

* [sdpctl profile](sdpctl_profile.md)	 - Manage configuration for multiple admin profiles

