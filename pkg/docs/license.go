package docs

var (
	LicenseRootDoc = CommandDoc{
		Short: "interact with Appgate SDP License",
		Long:  ``,
	}

	LicensePruneDoc = CommandDoc{
		Short: "clear the license back (from 30) to 1 day.",
		Long: `clear the license back (from 30) to 1 day.
This command only works on appliances higher or equal to 6.1 (API Version 18)`,
		Examples: []ExampleDoc{
			{
				Description: "",
				Command:     "sdpctl license prune",
			},
		},
	}
)
