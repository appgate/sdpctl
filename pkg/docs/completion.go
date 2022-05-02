package docs

var (
	CompletionDocs = CommandDoc{
		Short: "Generate shell completion scripts",
		Long:  `sdpctl provides a way to generate autocompletion scripts for your current shell. See examples for more details on your specific shell.`,
		Examples: []ExampleDoc{
			{
				Description: `Bash:
Note that if you installed sdpctl from deb or rpm package, bash completion is already included.`,
				Command: `mkdir -p ~/.local/share/bash-completion
sdpctl completion bash | tee ~/.local/share/bash-completion/sdpctl`,
			},
			{
				Description: `ZSH:
If shell completion is not already enabled in your environment,
you will need to enable it.  You can execute the following once:`,
				Command: `echo "autoload -U compinit; compinit" >> ~/.zshrc`,
			},
			{
				Description: "To load completions for each session, execute once:",
				Command:     `sdpctl completion zsh | tee --output-error=exit "/usr/share/zsh/vendor-completions/_sdpctl"`,
			},
			{
				Description: `Fish:
Load completion for the session:`,
				Command: "sdpctl completion fish | source",
			},
			{
				Description: "To load completions for each session, execute once:",
				Command:     "sdpctl completion fish | tee ~/.config/fish/completions/sdpctl.fish",
			},
			{
				Description: `PowerShell:
Load completion for the session`,
				Command: "sdpctl completion powershell | Out-String | Invoke-Expression",
			},
			{
				Description: `To load completions for every new session, run this command
and source this file from your PowerShell profile.`,
				Command: "sdpctl completion powershell > sdpctl.ps1",
			},
			{
				Description: `MacOS:
Load completion for session`,
				Command: "sdpctl completion bash | tee /usr/local/etc/bash_completion.d/sdpctl",
			},
		},
	}
)
