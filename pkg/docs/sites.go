package docs

var (
	SitesDocRoot = CommandDoc{
		Short:    "Root command for listing and checking status of sites in a collective",
		Examples: []ExampleDoc{},
	}
	SitesListDocs = CommandDoc{
		Short: "List and see status of sites of a collective",
		Examples: []ExampleDoc{
			{
				Command:     "sdpctl sites list",
				Description: "will list all sites configured in a collective if no arguments are used.",
				Output: `Site Name       Short Name    ID                                      Tags         Description    Status
---------       ----------    --                                      ----         -----------    ------
Default Site                  8a4add9e-0e99-4bb1-949c-c9faf9a49ad4    [builtin]                   healthy`,
			},
			{
				Command:     "sdpctl sites list --json",
				Description: "list sites with json output",
			},
		},
	}
	SitesResourcesDocsRoot = CommandDoc{
		Short:    "Command for resources",
		Long:     "",
		Examples: []ExampleDoc{},
	}
	SitesResourcesDocsList = CommandDoc{
		Short:    "Command for listing resources",
		Long:     "",
		Examples: []ExampleDoc{},
	}
)
