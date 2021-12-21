package token

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/appgate/appgatectl/pkg/util"
    "github.com/appgate/sdp-api-client-go/api/v16/openapi"
    "github.com/spf13/cobra"
    "io"
    "io/ioutil"
    "net/http"
    "time"
)

type RevokeOptions struct {
    TokenOptions *TokenOptions
	SiteID                     string
	RevocationReason           string
	DelayMinutes               int32
	TokensPerSecond            float32
	SpecificDistinguishedNames []string
}

func NewTokenRevokeCmd(parentOpts *TokenOptions) *cobra.Command {
	opts := &RevokeOptions{
        TokenOptions: parentOpts,
	}

	var revokeCmd = &cobra.Command{
		Use:   "revoke [by-distinguished-name | by-token-type]",
		Short: "revoke entitlement tokens by distinguished name or token-type",
	}

	revokeCmd.PersistentFlags().StringVar(&opts.SiteID, "site-id", "", "revoke only tokens for the given site ID")
	revokeCmd.PersistentFlags().StringVar(&opts.RevocationReason, "reason", "", "reason for revocation")
	revokeCmd.PersistentFlags().Float32Var(&opts.TokensPerSecond, "per-second", 7, "tokens are revoked in batches according to this value to spread load on the controller. defaults to 7 token per second")
	revokeCmd.PersistentFlags().Int32Var(&opts.DelayMinutes, "delay-minutes", 5, "delay time for token revocations in minutes. defaults to 5 minutes")
	revokeCmd.PersistentFlags().StringSliceVar(&opts.SpecificDistinguishedNames, "specific-distinguished-names", []string{}, "comma-separated string of distinguished names to renew tokens in bulk for a specific list of devices")

	revokeCmd.AddCommand(NewTokenRevokeByTokenTypeCmd(opts))
	revokeCmd.AddCommand(NewTokenRevokeByDistinguishedNameCmd(opts))

	return revokeCmd
}

type RevokeByDistinguishedNameOptions struct {
	RevokeOptions *RevokeOptions
	TokenType     string
}

func NewTokenRevokeByDistinguishedNameCmd(parentOpts *RevokeOptions) *cobra.Command {
	opts := RevokeByDistinguishedNameOptions{
		RevokeOptions: parentOpts,
	}

	var revokeByDistinguishedNameCmd = &cobra.Command{
		Use:   "by-distinguished-name [distinguished-name]",
		Short: "revoke entitlement tokens by distinguished name",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return revokeByDistinguishedNameRun(args, &opts)
		},
	}

	revokeByDistinguishedNameCmd.Flags().StringVar(&opts.TokenType, "type", "", "revoke only certain type of token")

	return revokeByDistinguishedNameCmd
}

func revokeByDistinguishedNameRun(args []string, opts *RevokeByDistinguishedNameOptions) error {
	ctx := context.Background()
	t, err := opts.RevokeOptions.TokenOptions.Token(opts.RevokeOptions.TokenOptions.Config)
	if err != nil {
		return err
	}

	request := t.APIClient.ActiveDevicesApi.TokenRecordsRevokedByDnDistinguishedNamePut(ctx, args[0])

	if opts.TokenType != "" {
		request.TokenType(opts.TokenType)
	}

	if opts.RevokeOptions.SiteID != "" {
		request.SiteId(opts.RevokeOptions.SiteID)
	}

	body := openapi.TokenRevocationRequest{
		TokensPerSecond: &opts.RevokeOptions.TokensPerSecond,
		DelayMinutes:    &opts.RevokeOptions.DelayMinutes,
	}

	if opts.RevokeOptions.RevocationReason != "" {
		body.RevocationReason = &opts.RevokeOptions.RevocationReason
	}

	if len(opts.RevokeOptions.SpecificDistinguishedNames) > 0 {
		body.SpecificDistinguishedNames = &opts.RevokeOptions.SpecificDistinguishedNames
	}

	response, err := t.RevokeByDistinguishedName(request, body)
	if err != nil {
		return err
	}

	err = PrintRevokedTokens(response, opts.RevokeOptions.TokenOptions.Out, opts.RevokeOptions.TokenOptions.useJSON)
	if err != nil {
		return err
	}

	return nil
}

func NewTokenRevokeByTokenTypeCmd(opts *RevokeOptions) *cobra.Command {
	var revokeByTokenCmd = &cobra.Command{
		Use:   "by-token-type [token-type]",
		Short: "revoke entitlement tokens by token type",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return revokeByTokenTypeRun(args, opts)
		},
	}

	return revokeByTokenCmd
}

func revokeByTokenTypeRun(args []string, opts *RevokeOptions) error {
	ctx := context.Background()
	t, err := opts.TokenOptions.Token(opts.TokenOptions.Config)
	if err != nil {
		return err
	}

	request := t.APIClient.ActiveDevicesApi.TokenRecordsRevokedByTypeTokenTypePut(ctx, args[0])

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
	result := &RevokeTokenResponse{}
	err = json.Unmarshal(responseBody, result)
	if err != nil {
		return err
	}

	if printJSON {
		return util.PrintJSON(out, result)
	}

	if len(result.Data) > 0 {
		p := util.NewPrinter(out)
		p.AddHeader("ID", "Type", "Distinguished Name", "Issued", "Expires", "Revoked", "Site ID", "Site Name", "Revocation Time", "Device ID", "Username", "Provider Name", "Controller Hostname")
		for _, t := range result.Data {
			p.AddLine(t.TokenID, t.TokenType, t.DistinguishedName, t.Issued, t.Expires, t.Revoked, t.Site, t.SiteName, t.RevocationTime, t.DeviceID, t.Username, t.ProviderName, t.ControllerHostname)
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

//TODO: Fix OpenAPI spec for /token-records/revoked/by-type and /token-records/revoked/by-dn

type Token struct {
	TokenID            string     `json:"tokenId,omitempty"`
	TokenType          string     `json:"tokenType,omitempty"`
	DistinguishedName  string     `json:"distinguishedName,omitempty"`
	Issued             *time.Time `json:"issued,omitempty"`
	Expires            *time.Time `json:"expires,omitempty"`
	Revoked            bool       `json:"revoked,omitempty"`
	Site               string     `json:"siteId,omitempty"`
	SiteName           string     `json:"siteName,omitempty"`
	RevocationTime     *time.Time `json:"revocationTime,omitempty"`
	DeviceID           string     `json:"deviceId,omitempty"`
	Username           string     `json:"username,omitempty"`
	ProviderName       string     `json:"providerName,omitempty"`
	ControllerHostname string     `json:"controllerHostname"`
}

type RevokeTokenResponse struct {
	Data       []Token  `json:"data,omitempty"`
	Query      string   `json:"query,omitempty"`
	Range      string   `json:"range,omitempty"`
	OrderBy    string   `json:"orderBy,omitempty"`
	Issued     bool     `json:"issued,omitempty"`
	Descending bool     `json:"descending,omitempty"`
	FilterBy   []string `json:"filterBy,omitempty"`
}
