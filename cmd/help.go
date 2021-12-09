package cmd

import (
	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

func HelpTemplate() string {
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{end}}


ENVIRONMENT VARIABLES
  See 'appgatectl help environment' for the list of supported environment variables.


{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

const environmentHelpLong = `

All environment variables are Optional, for initial configuration of appgatectl, run 'appgatectl configure'

The environment variables will take precedence over the values in the config file,
The default path to the config file is "$XDG_CONFIG_HOME/appgatectl" or "$HOME/.config/appgatectl on UNIX

APPGATECTL_URL: (Optional) URL to the controller API endpoint, for example https://appgate.acme.com:8443/admin

APPGATECTL_PROVIDER: (Optional) Display name of the Identity Provider name. Used during login (default to local)

APPGATECTL_INSECURE: (Optional) Whether server should be accessed without verifying the TLS certificate.

APPGATECTL_PEM_FILEPATH: (Optional) If appgatectl is configured insecure:false, you need to append this configuration and point to a valid PEM
file used by the controller.

APPGATECTL_VERSION: (Optional) Client peer version used to communicate with the controller API,
default value will be computed based on the primary controller appliance version.

APPGATECTL_BEARER: (Optional) The Bearer authentication, computed from 'appgatectl configure login'

APPGATECTL_DEVICE_ID: (Optional) UUID to distinguish the Client device making the request. It is supposed to be same for every login request from the same server.

    Defaults to:
        /etc/machine-id on Linux
        /etc/hostid on BSD
        ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID on OSX
        reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid on Windows

APPGATECTL_CREDENTIALS_FILE: (Optional) The filepath to optional credentials file, generated from 'appgatectl configure login --remember-me'

APPGATECTL_CONFIG_DIR: (Optional) the directory where appgatectl will store configuration files. Default:
"$XDG_CONFIG_HOME/appgatectl" or "$HOME/.config/appgatectl on UNIX".

APPGATECTL_LOG_LEVEL: application log level, default to INFO

`

func NewHelpCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "environment",
		Short:  "Environment variables that can be used with appgatectl",
		Long:   environmentHelpLong,
		Hidden: true,
	}
	cmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		command.Println(command.Long)
	})
	return cmd
}
