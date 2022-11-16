## sdpctl configure signin

Sign in and authenticate to Appgate SDP Collective

### Synopsis

Sign in to the Appgate SDP Collective using the configuration file created by the 'sdpctl configure' command.
This will fetch a token on valid authentication which will be valid for 24 hours and stored in the configuration.

```
sdpctl configure signin [flags]
```

### Examples

```
  # default sign in command
  > sdpctl configure signin
```

### Options

```
  -h, --help   help for signin
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

* [sdpctl configure](sdpctl_configure.md)	 - Configure your Appgate SDP Collective

