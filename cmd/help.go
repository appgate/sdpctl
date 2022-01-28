package cmd

import (
	"time"

	"github.com/appgate/appgatectl/pkg/factory"
	"github.com/spf13/cobra"
)

func HelpTemplate() string {
	cobra.AddTemplateFunc("now", time.Now)
	return `© {{ now.Year }} Appgate Cybersecurity, Inc.
All rights reserved. Appgate is a trademark of Appgate Cybersecurity, Inc.
https://www.appgate.com

{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{end}}

Environment Variables:
  See 'appgatectl help environment' for the list of supported environment variables.

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

const environmentHelpLong = `
All environment variables are Optional, for initial configuration of appgatectl, run 'appgatectl configure'

The environment variables will take precedence over the values in the config file,
The default path to the config file is "$XDG_CONFIG_HOME/appgatectl" or "$HOME/.config/appgatectl on UNIX

Available Variables:
  APPGATECTL_URL:
    Description: URL to the controller API endpoint, for example https://appgate.acme.com:8443/admin
  APPGATECTL_PROVIDER:
    Description: Display name of the Identity Provider name. Used during sign in
    Default: local
  APPGATECTL_INSECURE:
    Description: Whether server should be accessed without verifying the TLS certificate.
                 WARNING! Setting this to 'true' is strongly disadvised in a production environment.
    Default: false
  APPGATECTL_PEM_FILEPATH:
    Description: If appgatectl is configured insecure:false, you need to append this configuration and point
                 to a valid PEM file used by the controller.
  APPGATECTL_VERSION:
    Description: Client peer version used to communicate with the controller API, default value will be computed based on the
                 primary controller appliance version.
  APPGATECTL_BEARER:
    Description: The Bearer authentication, computed from 'appgatectl configure signin'
  APPGATECTL_USERNAME:
    Description: username for local identity provider, can be used instead of APPGATECTL_BEARER in combination with APPGATECTL_PASSWORD.
  APPGATECTL_PASSWORD:
    Description: password for local identity provider, can be used instead of APPGATECTL_BEARER in combination with APPGATECTL_USERNAME.
  APPGATECTL_DEVICE_ID:
    Description: UUID to distinguish the Client device making the request. It is supposed to be same for every sign in request from the same server.
    Default: /etc/machine-id on Linux
             /etc/hostid on BSD
             ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID on OSX
             reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid on Windows
  APPGATECTL_CREDENTIALS_FILE:
    Description: The filepath to optional credentials file, generated from 'appgatectl configure signin --remember-me'
  APPGATECTL_CONFIG_DIR:
    Description: the directory where appgatectl will store configuration files.
    Default: "$XDG_CONFIG_HOME/appgatectl" or "$HOME/.config/appgatectl on UNIX".
  APPGATECTL_LOG_LEVEL:
    Description: application log level
    Default: INFO
  HTTP_PROXY:
    Description: HTTP Proxy for the client

Example Usage:
  APPGATECTL_USERNAME=admin \
  APPGATECTL_PASSWORD=admin \
  APPGATECTL_URL=https://controller.appgate.com/admin \
  APPGATECTL_INSECURE=true \
  APPGATECTL_API_VERSION=15 \
  appgatectl appliance list
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
