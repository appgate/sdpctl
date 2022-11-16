## sdpctl configure

Configure your Appgate SDP Collective

### Synopsis

Setup a configuration file towards your Appgate SDP Collective to be able to interact with the collective. By default, the configuration file
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
      --api-version int   peer API version override
      --ci-mode           log to stderr instead of file and disable progress-bars
      --debug             Enable debug logging
      --no-interactive    suppress interactive prompt with auto accept
      --no-verify         don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string    profile configuration to use
```

### SEE ALSO

* [sdpctl](sdpctl.md)	 - sdpctl is a command line tool to control and handle Appgate SDP using the CLI
* [sdpctl configure signin](sdpctl_configure_signin.md)	 - Sign in and authenticate to Appgate SDP Collective

