package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/appgate/appgatectl/pkg/util"
	"github.com/appgate/sdp-api-client-go/api/v16/openapi"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
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
	TokenOptions               *TokenOptions
	SiteID                     string
	RevocationReason           string
	DelayMinutes               int32
	TokensPerSecond            float32
	SpecificDistinguishedNames []string
	ByTokenType                string
	TokenType                  string
}

func NewTokenRevokeCmd(parentOpts *TokenOptions) *cobra.Command {
	opts := &RevokeOptions{
		TokenOptions: parentOpts,
	}

	var revokeCmd = &cobra.Command{
		Use:   "revoke [<distinguished-name> | --by-token-type <type>]",
		Short: "revoke entitlement tokens by distinguished name or token-type",

		Args: func(cmd *cobra.Command, args []string) error {
			if (len(args) != 0 && len(args) != 1) || (len(args) == 0 && opts.ByTokenType == "") {
				return errors.New("must set either <distinghuished-name> or --by-token-type <type>")
			}

			if len(args) > 0 && opts.ByTokenType != "" {
				return errors.New("cannot set both <distinguished-name> and --by-token-type")
			}

			if opts.ByTokenType != "" && opts.TokenType != "" {
				return errors.New("cannot set --token-type when using --by-token-type <type>")
			}

			if len(args) == 0 && tokenType(opts.ByTokenType) == Unknown {
				return fmt.Errorf("unknown token type %s. valid types are { administration, adminclaims, entitlements, claims }", opts.ByTokenType)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ByTokenType != "" {
				return revokeByTokenTypeRun(args, opts)
			}

			return revokeByDistinguishedNameRun(args, opts)
		},
	}

	revokeCmd.Flags().StringVar(&opts.ByTokenType, "by-token-type", "", "revoke all tokens of this type. { administration, adminclaims, entitlements, claims }")
	revokeCmd.Flags().StringVar(&opts.SiteID, "site-id", "", "revoke only tokens for the given site ID")
	revokeCmd.Flags().StringVar(&opts.RevocationReason, "reason", "", "reason for revocation")
	revokeCmd.Flags().Float32Var(&opts.TokensPerSecond, "per-second", 7, "tokens are revoked in batches according to this value to spread load on the controller. defaults to 7 token per second")
	revokeCmd.Flags().Int32Var(&opts.DelayMinutes, "delay-minutes", 5, "delay time for token revocations in minutes. defaults to 5 minutes")
	revokeCmd.Flags().StringSliceVar(&opts.SpecificDistinguishedNames, "specific-distinguished-names", []string{}, "comma-separated string of distinguished names to renew tokens in bulk for a specific list of devices")
	revokeCmd.Flags().StringVar(&opts.TokenType, "token-type", "", "revoke only certain types of token when revoking by distinguished name")

	return revokeCmd
}

func revokeByDistinguishedNameRun(args []string, opts *RevokeOptions) error {
	ctx := context.Background()
	t, err := opts.TokenOptions.Token(opts.TokenOptions.Config)
	if err != nil {
		return err
	}

	request := t.APIClient.ActiveDevicesApi.TokenRecordsRevokedByDnDistinguishedNamePut(ctx, args[0])

	if opts.TokenType != "" {
		request.TokenType(opts.TokenType)
	}

	if opts.SiteID != "" {
		request.SiteId(opts.SiteID)
	}

	body := openapi.TokenRevocationRequest{
		TokensPerSecond: &opts.TokensPerSecond,
		DelayMinutes:    &opts.DelayMinutes,
	}

	if opts.RevocationReason != "" {
		body.RevocationReason = &opts.RevocationReason
	}

	if len(opts.SpecificDistinguishedNames) > 0 {
		body.SpecificDistinguishedNames = &opts.SpecificDistinguishedNames
	}

	response, err := t.RevokeByDistinguishedName(request, body)
	if err != nil {
		return err
	}

	err = PrintRevokedTokens(response, opts.TokenOptions.Out, opts.TokenOptions.useJSON)
	if err != nil {
		return err
	}

	return nil
}

func revokeByTokenTypeRun(args []string, opts *RevokeOptions) error {
	ctx := context.Background()
	t, err := opts.TokenOptions.Token(opts.TokenOptions.Config)
	if err != nil {
		return err
	}

	request := t.APIClient.ActiveDevicesApi.TokenRecordsRevokedByTypeTokenTypePut(ctx, opts.ByTokenType)

	if opts.SiteID != "" {
		request.SiteId(opts.SiteID)
	}

	body := openapi.TokenRevocationRequest{
		TokensPerSecond: &opts.TokensPerSecond,
		DelayMinutes:    &opts.DelayMinutes,
	}

	if opts.RevocationReason != "" {
		body.RevocationReason = &opts.RevocationReason
	}

	if len(opts.SpecificDistinguishedNames) > 0 {
		body.SpecificDistinguishedNames = &opts.SpecificDistinguishedNames
	}

	response, err := t.RevokeByTokenType(request, body)
	if err != nil {
		return err
	}

	err = PrintRevokedTokens(response, opts.TokenOptions.Out, opts.TokenOptions.useJSON)
	if err != nil {
		return err
	}

	return nil
}

func PrintRevokedTokens(response *http.Response, out io.Writer, printJSON bool) error {
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
    result := &openapi.TokenRevocationResponse{}
	err = json.Unmarshal(responseBody, result)
	if err != nil {
		return err
	}

	if printJSON {
		return util.PrintJSON(out, result.Data)
	}

	if len(result.GetData()) > 0 {
		p := util.NewPrinter(out)
		p.AddHeader("ID", "Type", "Distinguished Name", "Issued", "Expires", "Revoked", "Site ID", "Site Name", "Revocation Time", "Device ID", "Username", "Provider Name", "Controller Hostname")
		for _, t := range result.GetData() {
			p.AddLine(t.TokenId, t.TokenType, t.DistinguishedName, t.Issued, t.Expires, t.Revoked, t.SiteId, t.SiteName, t.RevocationTime, t.DeviceId, t.Username, t.ProviderName, t.ControllerHostname)
		}
		p.Print()
	} else {
		_, err = fmt.Fprintln(out, "No tokens were revoked")
		if err != nil {
			return err
		}
	}
	return nil
}
