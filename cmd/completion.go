package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCmdCompletion represents the completion command
func NewCmdCompletion() *cobra.Command {
	var completionCmd = &cobra.Command{
		Use: "completion [bash|zsh|fish|powershell]",
		Annotations: map[string]string{
			"skipAuthCheck": "true",
		},
		Short:                 "Generate shell completion scripts",
		Long:                  `sdpctl provides a way to generate autocompletion scripts for your current shell. See examples for more details on your specific shell.`,
		DisableFlagsInUseLine: true,
		DisableFlagParsing:    true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Example: `Bash:
    # Note that if you installed sdpctl from deb or rpm package, bash completion is already included.
    mkdir -p ~/.local/share/bash-completion
    sdpctl completion bash | tee ~/.local/share/bash-completion/sdpctl

ZSH:
    # If shell completion is not already enabled in your environment,
    # you will need to enable it.  You can execute the following once:
    echo "autoload -U compinit; compinit" >> ~/.zshrc

    # To load completions for each session, execute once:
    sdpctl completion zsh | tee --output-error=exit "/usr/share/zsh/vendor-completions/_sdpctl"

Fish:
    sdpctl completion fish | source

    # To load completions for each session, execute once:
    sdpctl completion fish | tee ~/.config/fish/completions/sdpctl.fish

PowerShell:
    sdpctl completion powershell | Out-String | Invoke-Expression

    # To load completions for every new session, run:
    # and source this file from your PowerShell profile.
    sdpctl completion powershell > sdpctl.ps1

MacOS:
    sdpctl completion bash | tee /usr/local/etc/bash_completion.d/sdpctl`,
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	return completionCmd
}
