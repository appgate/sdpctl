package cmdutil

import "github.com/spf13/cobra"

var HideIncludeExcludeFlags = func(command *cobra.Command, strings []string) {
	// Hide flag for this command
	command.Flags().MarkHidden("exclude")
	command.Flags().MarkHidden("include")
	// Call parent help func
	command.Parent().HelpFunc()(command, strings)
}
