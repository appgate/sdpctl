package docs

var (
	FilesDocs = CommandDoc{
		Short:    "The files command lets you manage the file repository on the connected Controller",
		Long:     `The files command lets you manage the file repository on the currently connected Controller.`,
		Examples: []ExampleDoc{},
	}
	FilesListDocs = CommandDoc{
		Short: "Lists the files in the Controllers file repository",
		Long: `Lists the files in the Controllers file repository. Default output is in table format.
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
	FilesDeleteDocs = CommandDoc{
		Short: "Delete files from the repository",
		Long:  `Delete files from the repository with this command. There are multiple options on which file(s) should be deleted.`,
		Examples: []ExampleDoc{
			{
				Description: "delete a single file using the filename as a parameter",
				Command:     "sdpctl appliance files delete file-to-delete.img.zip",
				Output:      "file-to-delete.img.zip: deleted",
			},
			{
				Description: "delete all files in the repository",
				Command:     "sdpctl appliance files delete --all",
				Output: `deleted1.img.zip: deleted
deleted2.img.zip: deleted`,
			},
			{
				Description: "no arguments will prompt for which files to delete",
				Command:     "sdpctl appliance files delete",
				Output: `? select files to delete:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
❯ [ ]  file1.img.zip
  [ ]  file2.img.zip
  [ ]  file3.img.zip
`,
			},
		},
	}
	FilesUploadDocs = CommandDoc{
		Short: "upload a file to the primary controllers file repository",
		Long:  `upload one or multiple files to the primary controller file repository. Optionally you can give the file a new name on the server using the '=' with the filepath (see examples for more details)`,
		Examples: []ExampleDoc{
			{
				Description: "upload one file",
				Command:     "sdpctl appliance files upload <path-to-file>",
			},
			{
				Description: "upload multiple files",
				Command:     "sdpctl appliance files upload <path1> <path2>",
			},
			{
				Description: "rename a file when uploading",
				Command:     "sdpctl appliance files upload <path-to-file>=<new-file-name>",
			},
		},
	}
)
