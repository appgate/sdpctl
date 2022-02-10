package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCmdCompletion represents the completion command
func NewCmdCompletion() *cobra.Command {
	var completionCmd = &cobra.Command{
		Use: "completion",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Short: "Generate shell completion scripts",
		Long: `To load completions:

Bash:

  $ source <(sdpctl completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ sdpctl completion bash > /etc/bash_completion.d/sdpctl
  # macOS:
  $ sdpctl completion bash > /usr/local/etc/bash_completion.d/sdpctl

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once (you may need to execute as 'sudo' user):
  $ sdpctl completion zsh > "/usr/share/zsh/vendor-completions/_sdpctl"

  # You will need to start a new shell for this setup to take effect.


PowerShell:

  PS> sdpctl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> sdpctl completion powershell > sdpctl.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Example:               "sdpctl completion bash",
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	return completionCmd
}
