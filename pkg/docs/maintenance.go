package docs

var (
	MaintenanceToggle = CommandDoc{
		Short: "Toggle maintenance mode on a single controller",
		Long: `
This function is a advance function, and should be used with care,
enabling maintenance mode on the primary controller can leave you with a unreachable environment.

USE WITH CAUTION

arguments are Optional, if no arguments are provided, you will be prompted
with an interactive prompt.
`,

		Examples: []ExampleDoc{
			{
				Description: "Toggle maintenance mode to false on a fixed controller UUID",
				Command:     "sdpctl appliance maintenance-toggle 20e75a08-96c6-4ea3-833e-cdbac346e2ae false",
				Output:      "Change result: success\nChange Status: completed",
			},

			{
				Description: "Toggle maintenance mode interactive prompt",
				Command:     "sdpctl appliance maintenance-toggle",
				Output: `
? select appliance: controller-two - Default Site - []
? Toggle maintenance mode to: false

An appliance in maintenance mode won't allow to perform POST, PUT, PATCH or DELETE methods.
This is a superuser function and should only be used if you know what you are doing.

? Are you really sure you want to set maintenance mode to false on controller-two?

Do you want to continue? Yes
Change result: success
Change Status: completed
                `,
			},
		},
	}
)
