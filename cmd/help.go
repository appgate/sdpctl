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

APPGATECTL_URL: URL to the controller API endpoint, for example https://appgate.acme.com:8443/admin

APPGATECTL_PROVIDER: Display name of the Identity Provider name. Used during signin (default to local)

APPGATECTL_INSECURE: Whether server should be accessed without verifying the TLS certificate.

APPGATECTL_PEM_FILEPATH: If appgatectl is configured insecure:false, you need to append this configuration and point to a valid PEM
file used by the controller.

APPGATECTL_VERSION: Client peer version used to communicate with the controller API,
default value will be computed based on the primary controller appliance version.

APPGATECTL_BEARER: The Bearer authentication, computed from 'appgatectl configure signin'

APPGATECTL_USERNAME: username for local identity provider, can be used instead of APPGATECTL_BEARER in combination with APPGATECTL_PASSWORD.
APPGATECTL_PASSWORD: password for local identity provider, can be used instead of APPGATECTL_BEARER in combination with APPGATECTL_USERNAME.
    Example usage:
        APPGATECTL_USERNAME=admin \
        APPGATECTL_PASSWORD=admin \
        APPGATECTL_URL=https://controller.appgate.com/admin \
        APPGATECTL_INSECURE=true \
        APPGATECTL_API_VERSION=15 \
        appgatectl appliance list

APPGATECTL_DEVICE_ID: UUID to distinguish the Client device making the request. It is supposed to be same for every signin request from the same server.

    Defaults to:
        /etc/machine-id on Linux
        /etc/hostid on BSD
        ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID on OSX
        reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid on Windows

APPGATECTL_CREDENTIALS_FILE: The filepath to optional credentials file, generated from 'appgatectl configure signin --remember-me'

APPGATECTL_CONFIG_DIR: the directory where appgatectl will store configuration files. Default:
"$XDG_CONFIG_HOME/appgatectl" or "$HOME/.config/appgatectl on UNIX".

APPGATECTL_LOG_LEVEL: application log level, default to INFO

HTTP_PROXY: HTTP Proxy for the client
    Example:  HTTP_PROXY="http://proxyIp:proxyPort"

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
