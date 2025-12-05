package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/appgate/sdp-api-client-go/api/v23/openapi"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/spf13/cobra"
)

type TokenType int

const (
	Administration TokenType = iota
	AdminClaims
	Entitlement
	Claims
	Unknown
)

func (t TokenType) String() string {
	switch t {
	case Administration:
		return "administration"
	case AdminClaims:
		return "adminclaims"
	case Entitlement:
		return "entitlement"
	case Claims:
		return "claims"
	}
	return "unknown"
}

func tokenType(t string) TokenType {
	switch strings.ToLower(t) {
	case "administration":
		return Administration
	case "adminclaims":
		return AdminClaims
	case "entitlement":
		return Entitlement
	case "claims":
		return Claims
	}
	return Unknown
}

type RevokeOptions struct {
	DeviceOptions              *DeviceOptions
	SiteID                     string
	RevocationReason           string
	DelayMinutes               int32
	TokensPerSecond            float32
	SpecificDistinguishedNames []string
	ByTokenType                string
	TokenType                  string
}

func NewDeviceRevokeCmd(parentOpts *DeviceOptions) *cobra.Command {
	opts := &RevokeOptions{
		DeviceOptions: parentOpts,
	}

	var revokeCmd = &cobra.Command{
		Use:     "revoke [<distinguished-name> | --by-token-type <type>]",
		Short:   docs.DeviceRevokeDoc.Short,
		Long:    docs.DeviceRevokeDoc.Long,
		Example: docs.DeviceRevokeDoc.ExampleString(),
		Args: func(cmd *cobra.Command, args []string) error {
			if (len(args) != 0 && len(args) != 1) || (len(args) == 0 && opts.ByTokenType == "") {
				return errors.New("Must set either <distinghuished-name> or --by-token-type <type>")
			}

			if len(args) > 0 && opts.ByTokenType != "" {
				return errors.New("Cannot set both <distinguished-name> and --by-token-type")
			}

			if opts.ByTokenType != "" && opts.TokenType != "" {
				return errors.New("Cannot set --token-type when using --by-token-type <type>")
			}

			if len(args) == 0 && tokenType(opts.ByTokenType) == Unknown {
				return fmt.Errorf("Unknown token type %s. valid types are { administration, adminclaims, entitlements, claims }", opts.ByTokenType)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return revokeByDistinguishedNameRun(args, opts)
		},
	}

	flags := revokeCmd.Flags()
	flags.StringVar(&opts.ByTokenType, "by-token-type", "", "Revoke all tokens of this type. { administration, adminclaims, entitlements, claims }")
	flags.StringVar(&opts.SiteID, "site-id", "", "Revoke only tokens for the given site ID")
	flags.StringVar(&opts.RevocationReason, "reason", "", "Reason for revocation")
	flags.Float32Var(&opts.TokensPerSecond, "per-second", 7, "Tokens are revoked in batches according to this value to spread load on the Controller. defaults to 7 token per second")
	flags.Int32Var(&opts.DelayMinutes, "delay-minutes", 5, "Delay time for token revocations in minutes. defaults to 5 minutes")
	flags.StringSliceVar(&opts.SpecificDistinguishedNames, "specific-distinguished-names", []string{}, "Comma-separated string of distinguished names to renew tokens in bulk for a specific list of devices")
	flags.StringVar(&opts.TokenType, "token-type", "", "Revoke only certain types of token when revoking by distinguished name")

	return revokeCmd
}

func revokeByDistinguishedNameRun(args []string, opts *RevokeOptions) error {
	t, err := opts.DeviceOptions.Device(opts.DeviceOptions.Config)
	if err != nil {
		return err
	}
	ctx := util.BaseAuthContext(t.Token)

	request := t.APIClient.RegisteredDevicesApi.OnBoardedDevicesRevokeTokensPost(ctx)

	body := openapi.DeviceRevocationRequest{
		DistinguishedNameFilter: "",
		DevicesPerSecond:        &opts.TokensPerSecond,
		DelayMinutes:            &opts.DelayMinutes,
	}

	if len(args) > 0 {
		body.DistinguishedNameFilter = args[0]
	}

	if opts.TokenType != "" {
		body.TokenType = &opts.TokenType
	}

	if opts.SiteID != "" {
		body.SiteId = &opts.SiteID
	}

	if opts.RevocationReason != "" {
		body.RevocationReason = &opts.RevocationReason
	}

	if len(opts.SpecificDistinguishedNames) > 0 {
		body.SpecificDistinguishedNames = opts.SpecificDistinguishedNames
	}

	response, err := t.RevokeByDistinguishedName(request, body)
	if err != nil {
		return err
	}

	err = PrintRevokedDevices(response, opts.DeviceOptions.Out, opts.DeviceOptions.useJSON)
	if err != nil {
		return err
	}

	return nil
}

func PrintRevokedDevices(response *http.Response, out io.Writer, printJSON bool) error {
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	result := &openapi.OnBoardedDeviceList{}
	err = json.Unmarshal(responseBody, result)
	if err != nil {
		return err
	}

	if printJSON {
		return util.PrintJSON(out, result.Data)
	}

	if len(result.GetData()) > 0 {
		p := util.NewPrinter(out, 2)
		p.AddHeader(
			"Distinguished Name",
			"Device ID",
			"Username",
			"Provider Name",
			"Device Type",
			"Hostname",
			"Onboarded At",
			"Last Seen At",
		)
		for _, t := range result.GetData() {
			p.AddLine(
				t.GetDistinguishedName(),
				t.GetDeviceId(),
				t.GetUsername(),
				t.GetProviderName(),
				t.GetDeviceType(),
				t.GetHostname(),
				t.GetOnBoardedAt(),
				t.GetLastSeenAt(),
			)
		}
		p.Print()
		return nil
	}
	_, err = fmt.Fprintln(out, "No tokens were revoked")
	if err != nil {
		return err
	}

	return nil
}
