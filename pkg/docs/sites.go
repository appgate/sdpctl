package docs

var (
	SitesDocRoot = CommandDoc{
		Short:    "Root command for listing and checking status of sites in a collective",
		Examples: []ExampleDoc{},
	}
	SitesListDocs = CommandDoc{
		Short: "List and see status of sites of a collective",
		Long: `List and see status of sites of a collective. By default, configured sites will be shown in a table, but viewing sites in json format is supported using the '--json' flag.
Text lines that contain multiple lines will be abbreviated to only showing the first line in the table view. In that case, the abbreviated lines will be indicated with '[...]'.
To view the full output, use the '--json' flag.`,
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
