package docs

var (
	TokenDoc = CommandDoc{
		Short: "Perform actions related to token on the Appgate SDP Collective",
		Long:  `The token command allows you to renew or revoke device tokens used in the Appgate SDP Collective.`,
	}
	TokenListDoc = CommandDoc{
		Short: "list distinguished names of active devices",
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
		Short: "revoke entitlement tokens by distinguished name or token-type",
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
