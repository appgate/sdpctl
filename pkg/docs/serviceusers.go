package docs

var (
	ServiceUsersRoot = &CommandDoc{
		Short:    "root command for managing service users in the Appgate SDP Collective",
		Long:     "",
		Examples: []ExampleDoc{},
	}
	ServiceUsersList = &CommandDoc{
		Short: "list service users in the Appgate SDP Collective",
		Long:  "",
		Examples: []ExampleDoc{
			{
				Description: "list service users",
				Command:     "sdpctl service-users list",
			},
			{
				Description: "output in JSON format",
				Command:     "sdpctl service-users list --json",
			},
		},
	}
	ServiceUsersGet = &CommandDoc{
		Short: "get detailed information about a service user",
		Long:  "",
		Examples: []ExampleDoc{
			{
				Description: "get user with id of <id>",
				Command:     "sdpctl service-users get <id>",
			},
		},
	}
	ServiceUsersCreate = &CommandDoc{
		Short: "create a new service user",
		Long:  "",
		Examples: []ExampleDoc{
			{
				Description: "create a new service user",
				Command:     "sdpctl service-users create",
				Output: `? Name for service user: <service-user-name>
? Passphrase for service user: <service-user-passphrase>
? Confirm your passphrase: <confirm-passphrase>`,
			},
			{
				Description: "create service user with flag input",
				Command:     `echo "<passphrase>" | sdpctl service-users create --name=<service-user-name>`,
			},
			{
				Description: "create a service user from a valid JSON file",
				Command:     "sdpctl service-users create --from-file=<path-to-json-file>",
			},
		},
	}
	ServiceUsersUpdate = &CommandDoc{
		Short: "update a service user",
		Long:  "",
		Examples: []ExampleDoc{
			{
				Description: "update the name of a service user with the id of <id>",
				Command:     "sdpctl service-users update <id> name <new-name>",
			},
			{
				Description: "set a new passphrase for service user with id of <id>",
				Command:     "sdpctl service-users update <id> passphrase <new-passphrase>",
			},
			{
				Description: "disable a service user with id of <id>",
				Command:     "sdpctl service-users update <id> disable",
			},
			{
				Description: "enable a service user with id of <id>",
				Command:     "sdpctl service-users update <id> enable",
			},
			{
				Description: "add a tag for a service user",
				Command:     "sdpctl service-users update <id> add tag <new-tag>",
			},
			{
				Description: "add a label for a service user",
				Command:     "sdpctl service-users update <id> add label <key>=<value>",
			},
			{
				Description: "remove a tag for a service user",
				Command:     "sdpctl service-users update <id> remove tag <tag>",
			},
			{
				Description: "remove a label for a service user",
				Command:     "sdpctl service-users update <id> remove label <key>",
			},
			{
				Description: "update a service user using a predefined JSON file",
				Command:     "sdpctl service-users update <id> --from-file=<path-to-json-file>",
			},
			{
				Description: "update multiple values of a service user",
				Command:     `sdpctl service-users update <id> '{"name": "<new-name>", "disabled": true}'`,
			},
		},
	}
	ServiceUsersDelete = &CommandDoc{
		Short: "delete one or more service user(s)",
		Long:  "",
		Examples: []ExampleDoc{
			{
				Description: "delete a service user with the id of <id>",
				Command:     "sdpctl service-users delete <id>",
			},
			{
				Description: "delete multiple service users by providing multiple id:s",
				Command:     "sdpctl service-users delete <id1> id2>",
			},
			{
				Description: "delete service user(s) using prompt",
				Command:     "sdpctl service-users delete",
			},
		},
	}
)
