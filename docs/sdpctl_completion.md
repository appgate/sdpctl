## sdpctl completion

Generate shell completion scripts

### Synopsis

sdpctl provides a way to generate autocompletion scripts for your current shell. See examples for more details on your specific shell.

```
sdpctl completion [bash|zsh|fish|powershell] [flags]
```

### Examples

```
  # Bash:
  # Note that if you installed sdpctl from deb or rpm package, bash completion is already included.
  > mkdir -p ~/.local/share/bash-completion
  > sdpctl completion bash | tee ~/.local/share/bash-completion/sdpctl

  # ZSH:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  > echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  > sdpctl completion zsh | tee --output-error=exit "/usr/share/zsh/vendor-completions/_sdpctl"

  # Fish:
  # Load completion for the session:
  > sdpctl completion fish | source

  # To load completions for each session, execute once:
  > sdpctl completion fish | tee ~/.config/fish/completions/sdpctl.fish

  # PowerShell:
  # Load completion for the session
  > sdpctl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run this command
  # and source this file from your PowerShell profile.
  > sdpctl completion powershell > sdpctl.ps1

  # MacOS:
  # Load completion for session
  > sdpctl completion bash | tee /usr/local/etc/bash_completion.d/sdpctl
```

### Options

```
  -h, --help   help for completion
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

