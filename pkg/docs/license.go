package docs

var (
	LicenseRootDoc = CommandDoc{
		Short: "interact with Appgate SDP License",
		Long:  ``,
	}

	LicensePruneDoc = CommandDoc{
		Short: "clear the license back (from 30) to 1 day.",
		Long: `clear the license back (from 30) to 1 day.
This command only works on appliances >= 6.1`,
		Examples: []ExampleDoc{
			{
				Description: "",
				Command:     "sdpctl license prune",
			},
		},
	}
)
