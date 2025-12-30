package docs

var (
	NamesMigrationsDocsList = CommandDoc{
		Short: "Migrate entitlement names to new format",
		Long: `Migrate entitlement names to new format. By default, the command will perform a dry run showing which entitlements would be updated.
To perform the actual migration, use the '--dryRun=false' flag.`,
		Examples: []ExampleDoc{
			{
				Command:     "sdpctl entitlements names-migration",
				Description: "will show which entitlements need migration (dry run)",
				Output: `Name                ID                                      Original Value                Updated Value
----                --                                      --------------                -------------
VPN Entitlement     8a4add9e-0e99-4bb1-949c-c9faf9a49ad4    azure://lb:lbFoo             azure://{"lbs":"lbFoo"}`,
			},
			{
				Command:     "sdpctl entitlements names-migration --dryRun=false",
				Description: "perform the actual migration",
			},
		},
	}
)
