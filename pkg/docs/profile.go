package docs

var (
	ProfileRootDoc = CommandDoc{
		Short: "Handle configuration for more then one appgate sdp profile",
		Long:  `Mange local configuration files for sdpctl towards more then one appgate sdp profile.`,
	}
	ProfileAddDoc = CommandDoc{
		Short: "Add another appgate sdp profile configuration",
		Long:  `Add creates a new appgate sdp profile configuration directory`,
	}
	ProfileDeleteDoc = CommandDoc{
		Short: "Remove an existing profile profile",
		Long:  `Remove an existing profile profile from your local configuration settings`,
	}
	ProfileListDoc = CommandDoc{
		Short: "List all existing profile profiles",
		Long:  ``,
	}
	ProfileSetDoc = CommandDoc{
		Short: "Set which profile profile to use",
		Long:  ``,

		Examples: []ExampleDoc{
			{
				Description: "Set profile without any arguments",
				Command:     "sdpctl profile set",
				Output: `? select profile:  [Use arrows to move, type to filter]
> production
  staging
  testing`,
			},
			{
				Description: "set production as your current configuration",
				Command:     "sdpctl profile set production",
			},
		},
	}
)
