package docs

var (
	ProfileRootDoc = CommandDoc{
		Short: "Manage configuration for multiple admin profiles",
		Long:  `Manage the local configuration of sdpctl for multiple admin profiles`,
	}
	ProfileAddDoc = CommandDoc{
		Short: "Add another admin profile configuration",
		Long:  `Add creates a new admin profile configuration directory`,
	}
	ProfileDeleteDoc = CommandDoc{
		Short: "Remove an existing admin profile",
		Long:  `Remove an existing admin profile from your local configuration`,
	}
	ProfileListDoc = CommandDoc{
		Short: "List all existing admin profiles",
		Long:  ``,
	}
	ProfileSetDoc = CommandDoc{
		Short: "Set which admin profile to use",
		Long:  ``,

		Examples: []ExampleDoc{
			{
				Description: "Set admin profile without any arguments",
				Command:     "sdpctl profile set",
				Output: `? select profile:  [Use arrows to move, type to filter]
❯ production
  staging
  testing`,
			},
			{
				Description: "set production as your current admin profile",
				Command:     "sdpctl profile set production",
			},
		},
	}
)
