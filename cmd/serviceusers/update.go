package serviceusers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/filesystem"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

func NewServiceUsersUpdateCMD(f *factory.Factory) *cobra.Command {
	opts := ServiceUsersOptions{
		Config: f.Config,
		API:    f.ServiceUsers,
		Out:    f.IOOutWriter,
	}
	cmd := &cobra.Command{
		Use:     "update [id] [args...]",
		Short:   docs.ServiceUsersUpdate.Short,
		Long:    docs.ServiceUsersUpdate.Long,
		Example: docs.ServiceUsersUpdate.ExampleString(),
		Aliases: []string{"edit", "set"},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("Not enough arguments")
			}
			if !util.IsUUID(args[0]) {
				return fmt.Errorf("%s: %s", InvalidUUIDError, args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceUserUpdateRun(cmd, args, opts)
		},
	}

	cmd.Flags().StringP("from-file", "f", "", "update service user with values using a valid json file")

	return cmd
}

func serviceUserUpdateRun(cmd *cobra.Command, args []string, opts ServiceUsersOptions) error {
	ctx := context.Background()
	api, err := opts.API(opts.Config)
	if err != nil {
		return err
	}

	id := args[0]
	toUpdate, err := api.Read(ctx, id)
	if err != nil {
		return err
	}

	fromFile, err := cmd.Flags().GetString("from-file")
	if err != nil {
		return err
	}
	dto := ServiceUserDTO{}
	if len(fromFile) > 0 {
		path := filesystem.AbsolutePath(fromFile)
		ok, err := util.FileExists(path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("File not found: %s", path)
		}
		file, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(file, &dto); err != nil {
			return err
		}
	} else {
		if len(args) < 2 {
			return fmt.Errorf("Not enough arguments")
		}
		arg := args[1]

		dto.Labels = toUpdate.GetLabels()
		dto.Tags = toUpdate.GetTags()

		var value string
		argv := strings.Split(arg, "=")
		if len(argv) > 1 {
			arg = argv[0]
			value = argv[1]
		}

		switch arg {
		case "passphrase", "password":
			hasStdin := false
			stat, err := os.Stdin.Stat()
			if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
				hasStdin = true
			}
			password, err := prompt.GetPassphrase(opts.In, false, hasStdin, "")
			if err != nil {
				return errors.New("You need to provide passphrase through stdin")
			}
			dto.Password = password
		case "name", "username":
			if len(value) > 0 {
				dto.Name = value
			} else if len(args) == 3 {
				dto.Name = args[2]
			} else {
				return fmt.Errorf("no value for argument %s", arg)
			}
		case "disable", "lock":
			dto.Disabled = true
		case "enable", "unlock":
			dto.Disabled = false
		case "add", "append":
			// adding tag or label requires at least four arguments
			if len(args) < 4 {
				return fmt.Errorf("Not enough arguments")
			}
			noun := args[2]
			value := args[3]
			switch noun {
			case "label":
				keyValue := strings.Split(value, "=")
				if len(keyValue) < 2 {
					return fmt.Errorf("No key or value provided for label")
				}
				dto.Labels[keyValue[0]] = keyValue[1]
			case "tag":
				dto.Tags = append(toUpdate.GetTags(), value)
			default:
				return fmt.Errorf("Unknown argument %s", noun)
			}
		case "remove", "rm":
			if len(args) < 4 {
				return fmt.Errorf("Not enough arguments")
			}
			noun := args[2]
			value := args[3]
			switch noun {
			case "label":
				_, ok := dto.Labels[value]
				if !ok {
					return fmt.Errorf("Failed to remove label %s: label does not exist", value)
				}
				delete(dto.Labels, value)
			case "tag":
				newTags := []string{}
				for _, t := range dto.Tags {
					if value != t {
						newTags = append(newTags, t)
					}
				}
				dto.Tags = newTags
			default:
				return fmt.Errorf("unknown argument %s", noun)
			}
		default:
			// If no noun is given as an argument, we expect the second argument to be a JSON parsable string
			if err := json.Unmarshal([]byte(arg), &dto); err != nil {
				return err
			}
		}
	}

	if len(dto.Name) > 0 && dto.Name != toUpdate.GetName() {
		toUpdate.SetName(dto.Name)
	}
	if len(dto.Password) > 0 {
		toUpdate.SetPassword(dto.Password)
	}
	if dto.Disabled != toUpdate.GetDisabled() {
		toUpdate.SetDisabled(dto.Disabled)
	}
	if len(dto.Notes) > 0 && dto.Notes != toUpdate.GetNotes() {
		toUpdate.SetNotes(dto.Notes)
	}
	toUpdate.SetTags(dto.Tags)
	toUpdate.SetLabels(dto.Labels)

	updated, err := api.Update(ctx, *toUpdate)
	if err != nil {
		return err
	}

	return util.PrintJSON(opts.Out, updated)
}
