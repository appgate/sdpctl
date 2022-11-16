## sdpctl license prune

clear the license back (from 30) to 1 day.

### Synopsis

clear the license back (from 30) to 1 day.
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
      --api-version int   peer API version override
      --ci-mode           log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --no-interactive    suppress interactive prompt with auto accept
      --no-verify         don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string    profile configuration to use
```

### SEE ALSO

* [sdpctl license](sdpctl_license.md)	 - interact with Appgate SDP License

