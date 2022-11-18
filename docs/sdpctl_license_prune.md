## sdpctl license prune

clear the license back (from 30) to 1 day

### Synopsis

clear the license back (from 30) to 1 day
This command only works on appliances higher or equal to 6.1 (API Version 18)

```
sdpctl license prune [flags]
```

### Examples

```
  > sdpctl license prune
```

### Options

```
  -h, --help   help for prune
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

* [sdpctl license](sdpctl_license.md)	 - Manage assigned User/Portal/Service licenses

