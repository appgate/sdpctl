## sdpctl configure

Configure your Collective

### Synopsis

Setup a configuration file towards your Collective to be able to interact with the collective. By default, the configuration file
will be created in a default directory in depending on your system. This can be overridden by setting the 'SDPCTL_CONFIG_DIR' environment variable.
See 'sdpctl help environment' for more information on using environment variables.

```
sdpctl configure [flags]
```

### Examples

```
  # basic configuration command
  > sdpctl configure

  # configuration, no interactive
  > sdpctl configure appgate.controller.com

  # configure sdpctl using a custom certificate file
  > sdpctl configure --pem=/path/to/pem

  # configure using a custom confiuration directory
  > SDPCTL_CONFIG_DIR=/path/config/dir sdpctl configure
```

### Options

```
  -h, --help         help for configure
      --pem string   Path to PEM file to use for request certificate validation
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
* [sdpctl configure signin](sdpctl_configure_signin.md)	 - Sign in and authenticate to Collective

