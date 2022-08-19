package docs

var (
	FilesDocs = CommandDoc{
		Short:    "The files command lets you manage the file repository on the connected Controller",
		Long:     `The files command lets you manage the file repository on the currently connected Controller.`,
		Examples: []ExampleDoc{},
	}
	FilesListDocs = CommandDoc{
		Short: "lists the files in the controllers file repository",
		Long: `Lists the files in the controllers file repository. Default output is in table format.
Optionally print the output in JSON format by using the "--json" flag`,
		Examples: []ExampleDoc{
			{
				Description: "list files table output",
				Command:     "sdctl files list",
				Output: `Name                                Status    Created                                 Modified                                Failure Reason
----                                ------    -------                                 --------                                --------------
appgate-6.0.1-29983-beta.img.zip    Ready     2022-08-19 08:06:20.909002 +0000 UTC    2022-08-19 08:06:20.909002 +0000 UTC`,
			},
			{
				Description: "list files using JSON output",
				Command:     "sdctl files list --json",
				Output: `[
  {
    "creationTime": "2022-08-19T08:06:20.909002Z",
    "lastModifiedTime": "2022-08-19T08:06:20.909002Z",
    "name": "appgate-6.0.1-29983-beta.img.zip",
    "status": "Ready"
  }
]`,
			},
		},
	}
)
