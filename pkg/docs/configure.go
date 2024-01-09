package docs

var (
	ConfigureDocs = CommandDoc{
		Short: "Configure your Collective",
		Long: `Setup a configuration file towards your Collective to be able to interact with the collective. By default, the configuration file
will be created in a default directory in depending on your system. This can be overridden by setting the 'SDPCTL_CONFIG_DIR' environment variable.
See 'sdpctl help environment' for more information on using environment variables.`,
		Examples: []ExampleDoc{
			{
				Description: "basic configuration command",
				Command:     "sdpctl configure",
			},
			{
				Description: "configuration, no interactive",
				Command:     "sdpctl configure company.controller.com",
			},
			{
				Description: "configure sdpctl using a custom certificate file",
				Command:     "sdpctl configure --pem=/path/to/pem",
			},
			{
				Description: "configure using a custom confiuration directory",
				Command:     "SDPCTL_CONFIG_DIR=/path/config/dir sdpctl configure",
			},
		},
	}
	ConfigureSigninDocs = CommandDoc{
		Short: "Sign in and authenticate to Collective",
		Long: `Sign in to the Collective using the configuration file created by the 'sdpctl configure' command.
This will fetch a token on valid authentication which will be valid for 24 hours and stored in the configuration.`,
		Examples: []ExampleDoc{
			{
				Description: "default sign in command",
				Command:     "sdpctl configure signin",
			},
		},
	}
)
