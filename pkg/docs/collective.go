package docs

var (
	CollectiveRootDoc = CommandDoc{
		Short: "Handle configuration for more then one appgate sdp collective",
		Long:  `Mange local configuration files for sdpctl towards more then one appgate sdp collective.`,
	}
	CollectiveAddDoc = CommandDoc{
		Short: "Add another appgate sdp collective configuration",
		Long:  `Add creates a new appgate sdp collective configuration directory`,
	}
	CollectiveDeleteDoc = CommandDoc{
		Short: "Remove an existing collective profile",
		Long:  `Remove an existing collective profile from your local configuration settings`,
	}
	CollectiveListDoc = CommandDoc{
		Short: "List all existing collective profiles",
		Long:  ``,
	}
	CollectiveSetDoc = CommandDoc{
		Short: "Set which collective profile to use",
		Long:  ``,

		Examples: []ExampleDoc{
			{
				Description: "Set collective without any arguments",
				Command:     "sdpctl collective set",
				Output: `? select collective:  [Use arrows to move, type to filter]
> production
  staging
  testing`,
			},
			{
				Description: "set production as your current configuration",
				Command:     "sdpctl collective set production",
			},
		},
	}
)
