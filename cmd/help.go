package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/spf13/cobra"
)

func HelpTemplate() string {
	cobra.AddTemplateFunc("now", time.Now)
	cobra.AddTemplateFunc("caller", getCaller)
	return `© {{ now.Year }} Appgate Cybersecurity, Inc.
All rights reserved. Appgate is a trademark of Appgate Cybersecurity, Inc.
https://www.appgate.com

{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{end}}

Environment Variables:
  See '{{caller}} help environment' for the list of supported environment variables.

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

func UsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{with .ValidArgs}}

Valid arguments:{{range $arg := .}}
  {{. | trimTrailingWhitespaces}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

const environmentHelpLong = `
All environment variables are Optional, for initial configuration of sdpctl, run 'sdpctl configure'

The environment variables will take precedence over the values in the config file,
The default path to the config file is "$XDG_CONFIG_HOME/sdpctl" or "$HOME/.config/sdpctl on UNIX

Available Variables:
  SDPCTL_URL:
    Description: URL to the controller API endpoint, for example https://appgate.acme.com:8443/admin
  SDPCTL_PROVIDER:
    Description: Display name of the Identity Provider name. Used during sign in
    Default: local
  SDPCTL_INSECURE:
    Description: Whether server should be accessed without verifying the TLS certificate.
                 WARNING! Setting this to 'true' is strongly disadvised in a production environment.
    Default: false
  SDPCTL_PEM_FILEPATH:
    Description: If sdpctl is configured insecure:false, you need to append this configuration and point
                 to a valid PEM file used by the controller.
  SDPCTL_VERSION:
    Description: Client peer version used to communicate with the controller API, default value will be computed based on the
                 primary controller appliance version.
  SDPCTL_BEARER:
    Description: The Bearer authentication, computed from 'sdpctl configure signin'
  SDPCTL_USERNAME:
    Description: username for local identity provider, can be used instead of SDPCTL_BEARER in combination with SDPCTL_PASSWORD.
  SDPCTL_PASSWORD:
    Description: password for local identity provider, can be used instead of SDPCTL_BEARER in combination with SDPCTL_USERNAME.
  SDPCTL_DEVICE_ID:
    Description: UUID to distinguish the Client device making the request. It is supposed to be same for every sign in request from the same server.
    Default: /etc/machine-id on Linux
             /etc/hostid on BSD
             ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID on OSX
             reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid on Windows
  SDPCTL_CONFIG_DIR:
    Description: the directory where sdpctl will store configuration files.
    Default: "$XDG_CONFIG_HOME/sdpctl" or "$HOME/.config/sdpctl on UNIX and %APPDATA%\Local\sdpctl on Windows".
  SDPCTL_LOG_LEVEL:
    Description: application log level
    Default: INFO
  HTTP_PROXY:
    Description: HTTP Proxy for the client

Example Usage:
  SDPCTL_USERNAME=admin \
  SDPCTL_PASSWORD=admin \
  SDPCTL_URL=https://controller.appgate.com/admin \
  SDPCTL_INSECURE=true \
  SDPCTL_API_VERSION=15 \
  sdpctl appliance list
`

func NewHelpCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "environment",
		Short:  "Environment variables that can be used with sdpctl",
		Long:   environmentHelpLong,
		Hidden: true,
	}
	cmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		command.Println(command.Long)
	})
	return cmd
}

func getCaller() string {
	binary := "sdpctl"
	raw := os.Args[0]
	regex := regexp.MustCompile(`sdpctl`)
	if bin := filepath.Base(raw); regex.MatchString(bin) {
		binary = bin
	}
	return binary
}
