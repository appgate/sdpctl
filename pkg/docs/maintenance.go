package docs

var (
	MaintenanceModeLongDescription = `
    For advanced users only.
    enabling maintenance mode on the primary Controller can leave you with a unreachable environment.

    USE WITH CAUTION

    arguments are Optional, if no arguments are provided, you will be prompted
    with an interactive prompt.
    `
	MaintenanceRootDoc = CommandDoc{
		Short: "Manually manage maintenance mode on Controllers",
		Long:  ``,
	}

	MaintenanceDisable = CommandDoc{
		Short: "Disable maintenance mode on a single Controller",
		Long:  MaintenanceModeLongDescription,

		Examples: []ExampleDoc{
			{
				Description: "Disable maintenance mode on a fixed Controller UUID",
				Command:     "sdpctl appliance maintenance disable 20e75a08-96c6-4ea3-833e-cdbac346e2ae",
				Output:      "Change result: success\nChange Status: completed",
			},

			{
				Description: "Disable maintenance mode interactive prompt",
				Command:     "sdpctl appliance maintenance disable",
				Output: `
? select appliance: Controller-two - Default Site - []

A Controller in maintenance mode will not accept any API calls besides disabling maintenance mode. Starting in version 6.0, clients will still function as usual while a Controller is in maintenance mode.

? Are you really sure you want to disable maintenance mode on Controller-two?

Do you want to continue? Yes
Change result: success
Change Status: completed
                `,
			},
		},
	}

	MaintenanceEnable = CommandDoc{
		Short: "Enable maintenance mode on a single Controller",
		Long:  MaintenanceModeLongDescription,

		Examples: []ExampleDoc{
			{
				Description: "Toggle maintenance mode to false on a fixed Controller UUID",
				Command:     "sdpctl appliance maintenance enable 20e75a08-96c6-4ea3-833e-cdbac346e2ae",
				Output:      "Change result: success\nChange Status: completed",
			},

			{
				Description: "Enable maintenance mode interactive prompt",
				Command:     "sdpctl appliance maintenance enable",
				Output: `
? select appliance: Controller-two - Default Site - []

This is a superuser function and should only be used if you know what you are doing.

? Are you really sure you want to enable maintenance mode on Controller-two?

Do you want to continue? Yes
Change result: success
Change Status: completed
                `,
			},
		},
	}
)
