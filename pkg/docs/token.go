package docs

var (
	TokenDoc = CommandDoc{
		Short: "Perform actions on Admin, Claims and Entitlement tokens",
		Long:  `The token command allows you to renew or revoke tokens used in the Collective.`,
	}
	TokenListDoc = CommandDoc{
		Short: "List distinguished names of active devices",
		Long:  "List distinguished names of active tokens, either in table format or JSON format using the '--json' flag",
		Examples: []ExampleDoc{
			{
				Description: "default list command",
				Command:     "sdpctl token list",
			},
			{
				Description: "print list in JSON format",
				Command:     "sdpctl token list --json",
			},
		},
	}
	TokenRevokeDoc = CommandDoc{
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
				Command:     "sdpctl token revoke <distinguished-name>",
			},
			{
				Description: "revoke by token type",
				Command:     "sdpctl token revoke --token-type=claims",
			},
		},
	}
)
