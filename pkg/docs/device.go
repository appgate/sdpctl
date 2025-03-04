package docs

var (
	DeviceDoc = CommandDoc{
		Short: "Perform actions on registered devices",
		Long:  `The device command allows you to renew or revoke tokens used in the Collective.`,
	}
	DeviceListDoc = CommandDoc{
		Short: "List distinguished names of active devices",
		Long:  "List distinguished names of active devices, either in table format or JSON format using the '--json' flag",
		Examples: []ExampleDoc{
			{
				Description: "default list command",
				Command:     "sdpctl device list",
			},
			{
				Description: "print list in JSON format",
				Command:     "sdpctl device list --json",
			},
		},
	}
	DeviceRevokeDoc = CommandDoc{
		Short: "Revoke entitlement tokens by distinguished name or token-type",
		Long: `Revoke tokens by distinguished name or token type.

Valid token types are:
  - administration
  - adminclaims
  - entitlements
  - claims`,
		Examples: []ExampleDoc{
			{
				Description: "revoke by distinguished name",
				Command:     "sdpctl device revoke <distinguished-name>",
			},
			{
				Description: "revoke by token type",
				Command:     "sdpctl device revoke --token-type=claims",
			},
		},
	}
)
