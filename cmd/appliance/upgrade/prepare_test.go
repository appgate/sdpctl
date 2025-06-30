package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/appgate/sdp-api-client-go/api/v22/openapi"
	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/dns"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/foxcpp/go-mockdns"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func init() {}

type mockUpgradeStatus struct{}

func (u *mockUpgradeStatus) WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error {
	return nil
}

type errorUpgradeStatus struct{}

func (u *errorUpgradeStatus) WaitForUpgradeStatus(ctx context.Context, appliance openapi.Appliance, desiredStatuses []string, undesiredStatuses []string, tracker *tui.Tracker) error {
	return fmt.Errorf("gateway never reached %s, got failed", strings.Join(desiredStatuses, ", "))
}

func NewApplianceCmd(f *factory.Factory) *cobra.Command {
	// define prepare parent command flags so we can include these in the tests.
	cmd := &cobra.Command{
		Use:              "appliance",
		Short:            docs.ApplianceRootDoc.Short,
		Long:             docs.ApplianceRootDoc.Long,
		Aliases:          []string{"app", "a"},
		TraverseChildren: true,
	}
	pFlags := cmd.PersistentFlags()
	pFlags.StringToStringP("include", "i", map[string]string{}, "Include appliances. Adheres to the same syntax and key-value pairs as '--exclude'")
	pFlags.StringToStringP("exclude", "e", map[string]string{}, "")
	pFlags.StringSlice("order-by", []string{"name"}, "")
	pFlags.Bool("descending", false, "Change the direction of sort order when using the '--order-by' flag. Using this will reverse the sort order for all keywords specified in the '--order-by' flag.")
	return cmd
}

func TestUpgradePrepareCommand(t *testing.T) {
	os.Setenv("SDPCTL_DOCKER_REGISTRY", "https://localhost:5001")
	tests := []struct {
		name                string
		cli                 string
		askStubs            func(*prompt.PromptStubber)
		httpStubs           []httpmock.Stub
		tlsStubs            []httpmock.Stub // For HTTPS logserver bundle downloads
		upgradeStatusWorker appliancepkg.WaitForUpgradeStatus
		wantOut             *regexp.Regexp
		wantErr             bool
		wantErrOut          *regexp.Regexp
	}{
		{
			name:       "with args",
			cli:        "upgrade prepare some.invalid.arg.com",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`accepts 0 arg\(s\), received 1`),
		},
		{
			name: "with existing file",
			cli:  "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip'",
			askStubs: func(s *prompt.PromptStubber) {
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/admin/files/appgate-6.2.2-9876.img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with gateway filter",
			cli:  `upgrade prepare --include function=gateway --image './testdata/appgate-6.2.2-9876.img.zip'`,
			askStubs: func(s *prompt.PromptStubber) {
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/admin/files/appgate-6.2.2-9876.img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
			wantErr: false,
		},
		{
			name:                "error upgrade status",
			cli:                 "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip'",
			upgradeStatusWorker: &errorUpgradeStatus{},
			wantErrOut:          regexp.MustCompile(`gateway never reached verifying, ready, got failed`),
			askStubs: func(s *prompt.PromptStubber) {
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_upgrade_status_idle.json"),
				},
				{
					URL:       "/admin/files/appgate-6.2.2-9876.img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{
		                    "status": "idle",
		                    "details": "a reboot is required for the Upgrade to go into effect"
		                  }`))
					},
				},
			},
			wantErr: true,
		},
		{
			name:       "no image argument",
			cli:        "upgrade prepare",
			httpStubs:  []httpmock.Stub{},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`--image is mandatory`),
		},
		{
			name: "no prepare confirmation",
			cli:  "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip' --force",
			askStubs: func(s *prompt.PromptStubber) {
				s.StubOne(false) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
			wantErr: true,
		},
		{
			name:       "image file not found",
			cli:        "upgrade prepare --image 'abc123456.img.zip'",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`.+Image file not found ".+abc123456.img.zip"`),
		},
		{
			name:       "file name error",
			cli:        "upgrade prepare --image './testdata/appgate.img'",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Invalid name on image file. The format is expected to be a .img.zip archive`),
		},
		{
			name:       "invalid zip file error",
			cli:        "upgrade prepare --image './testdata/invalid.img.zip'",
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`zip: not a valid zip file`),
		},
		{
			name: "prepare same version",
			cli:  "upgrade prepare --image './testdata/appgate-6.2.2-12345.img.zip'",
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance_6.2.2.json"),
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`No appliances to prepare for upgrade. All appliances may have been filtered or are already prepared. See the log for more details`),
		},
		{
			name: "force prepare same version",
			cli:  "upgrade prepare --force --image './testdata/appgate-6.2.2-12345.img.zip'",
			askStubs: func(as *prompt.PromptStubber) {
				as.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance_6.2.2.json"),
				},
				{
					URL:       "/admin/files/appgate-6.2.2-12345.img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"ready","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
		},
		{
			name: "with remote logserver bundle success",
			cli:  "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip' --logserver-bundle '%s/logserver-6.5.image.zip'",
			askStubs: func(s *prompt.PromptStubber) {
				s.StubOne(true) // upgrade_confirm
			},
			tlsStubs: []httpmock.Stub{
				{
					URL: "/logserver-6.5.image.zip",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						file, _ := os.Open("./testdata/appgate-6.2.2-9876.img.zip")
						defer file.Close()

						rw.Header().Set("Content-Type", "application/zip")
						rw.Header().Set("Content-Disposition", "attachment; filename=logserver-6.5.image.zip")
						rw.WriteHeader(http.StatusOK)
						io.Copy(rw, file)
					},
				},
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/admin/files/appgate-6.2.2-9876.img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "logserver bundle HTTP validation",
			cli:        "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip' --logserver-bundle 'http://example.com/bundle.zip'",
			httpStubs:  []httpmock.Stub{},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`Plain HTTP URLs are not supported for LogServer bundle`),
		},
		{
			name: "remote logserver bundle download failure - HTTPS 404",
			cli:  "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip' --logserver-bundle '%s/nonexistent-bundle.zip'",
			tlsStubs: []httpmock.Stub{
				{
					URL: "/nonexistent-bundle.zip",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						http.Error(rw, "Not Found", http.StatusNotFound)
					},
				},
			},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`failed to download LogServer bundle: HTTP 404`),
		},
		{
			name:       "remote logserver bundle download failure - connection error",
			cli:        "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip' --logserver-bundle 'https://nonexistent.invalid.domain.test/bundle.zip'",
			httpStubs:  []httpmock.Stub{},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`failed to download LogServer bundle from URL`),
		},
		{
			name: "local logserver bundle file exists",
			cli:  "upgrade prepare --image './testdata/appgate-6.2.2-9876.img.zip' --logserver-bundle './testdata/appgate-6.2.2-9876.img.zip'",
			askStubs: func(s *prompt.PromptStubber) {
				s.StubOne(true) // upgrade_confirm
			},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/appliances",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/appliance_list.json"),
				},
				{
					URL:       "/admin/appliances/status",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/stats_appliance.json"),
				},
				{
					URL:       "/admin/files/appgate-6.2.2-9876.img.zip",
					Responder: httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json"),
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "37bdc593-df27-49f8-9852-cb302214ee1f" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade/prepare",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							httpmock.JSONResponse("../../../pkg/appliance/fixtures/upgrade_status_file.json")
							return
						}
						if r.Method == http.MethodPost {
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							fmt.Fprint(rw, string(`{"id": "493a0d78-772c-4a6d-a618-1fbfdf02ab68" }`))
						}
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/change/37bdc593-df27-49f8-9852-cb302214ee1f",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/change/493a0d78-772c-4a6d-a618-1fbfdf02ab68",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, string(`{"status": "completed", "result": "success"}`))
					},
				},
				{
					URL: "/admin/appliances/ee639d70-e075-4f01-596b-930d5f24f569/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
				{
					URL: "/admin/appliances/4c07bc67-57ea-42dd-b702-c2d6c45419fc/upgrade",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusOK)
						fmt.Fprint(rw, string(`{"status":"idle","details":"appgate-6.2.2-9876.img.zip"}`))
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, teardown := dns.RunMockDNSServer(map[string]mockdns.Zone{
				"appgate.test.": {
					A: []string{"127.0.0.1"},
				},
			})
			defer teardown()
			registry := httpmock.NewRegistry(t)
			for _, v := range tt.httpStubs {
				registry.Register(v.URL, v.Responder)
			}

			defer registry.Teardown()
			registry.Serve()

			// Set up TLS registry for HTTPS LogServer bundle downloads if needed
			var tlsRegistry *TLSRegistry
			if len(tt.tlsStubs) > 0 {
				tlsRegistry = newTLSRegistry(t)
				for _, v := range tt.tlsStubs {
					tlsRegistry.Register(v.URL, v.Responder)
				}
				defer tlsRegistry.Teardown()
				tlsRegistry.Serve()
			}
			stdout := &bytes.Buffer{}
			stdin := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			in := io.NopCloser(stdin)
			f := &factory.Factory{
				Config: &configuration.Config{
					Debug:   false,
					URL:     fmt.Sprintf("http://appgate.test:%d", registry.Port),
					Version: 16,
				},
				IOOutWriter: stdout,
				Stdin:       in,
				StdErr:      stderr,
			}
			f.APIClient = func(c *configuration.Config) (*openapi.APIClient, error) {
				return registry.Client, nil
			}
			f.HTTPClient = func() (*http.Client, error) {
				// Use TLS client if we have a TLS registry, otherwise use the regular client
				if tlsRegistry != nil {
					return tlsRegistry.server.Client(), nil
				}
				return registry.Client.GetConfig().HTTPClient, nil
			}
			f.SetSpinnerOutput(io.Discard) // Disable spinner output in tests
			f.Appliance = func(c *configuration.Config) (*appliancepkg.Appliance, error) {
				api, _ := f.APIClient(c)

				a := &appliancepkg.Appliance{
					APIClient:  api,
					HTTPClient: api.GetConfig().HTTPClient,
					Token:      "",
				}
				if tt.upgradeStatusWorker != nil {
					a.UpgradeStatusWorker = tt.upgradeStatusWorker
				} else {
					a.UpgradeStatusWorker = new(mockUpgradeStatus)
				}

				return a, nil
			}
			f.DockerRegistry = f.GetDockerRegistry
			// add parent command to allow us to include test with parent flags
			cmd := NewApplianceCmd(f)
			upgradeCmd := NewUpgradeCmd(f)
			cmd.AddCommand(upgradeCmd)
			upgradeCmd.AddCommand(NewPrepareUpgradeCmd(f))

			// cobra hack
			cmd.Flags().BoolP("help", "x", false, "")
			cmd.PersistentFlags().Bool("ci-mode", false, "")

			cli := tt.cli
			if strings.Contains(tt.cli, "%s") {
				// Use TLS registry URL for HTTPS tests, otherwise use HTTP registry URL
				if tlsRegistry != nil {
					cli = fmt.Sprintf(tt.cli, tlsRegistry.URL())
				} else {
					cli = fmt.Sprintf(tt.cli, fmt.Sprintf("http://appgate.test:%d", registry.Port))
				}
			}

			argv, err := shlex.Split(cli)
			if err != nil {
				panic("Internal testing error, failed to split args")
			}
			cmd.SetArgs(argv)

			out := &bytes.Buffer{}
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(out)
			cmd.SetErr(io.Discard)

			stubber, teardown := prompt.InitStubbers(t)
			defer teardown()

			if tt.askStubs != nil {
				tt.askStubs(stubber)
			}
			_, err = cmd.ExecuteC()
			if (err != nil) != tt.wantErr {
				t.Logf("Stdout: %s", stdout)
				t.Fatalf("TestUpgradePrepareCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErrOut != nil {
				if !tt.wantErrOut.MatchString(err.Error()) {
					t.Logf("Stdout: %s", stdout)
					t.Errorf("Expected output to match, got:\n%s\n expected: \n%s\n", err.Error(), tt.wantErrOut)
				}
			}
			if tt.wantOut != nil {
				got, err := io.ReadAll(out)
				if err != nil {
					t.Fatal("Test error: Failed to read output buffer")
				}
				if !tt.wantOut.Match(got) {
					t.Fatalf("WANT: %s\nGOT: %s", tt.wantOut.String(), string(got))
				}
			}
		})
	}
}

func TestCheckImageFilename(t *testing.T) {
	type args struct {
		i string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "s3 bucket url",
			args: args{
				i: "https://s3.us-central-1.amazonaws.com/bucket/appgate-5.5.99-123-release.img.zip",
			},
			wantErr: false,
		},

		{
			name: "localpath",
			args: args{
				i: "/tmp/artifacts/55/appgate-5.5.2-99999-release.img.zip",
			},
			wantErr: false,
		},
		{
			name: "test url with get variables",
			args: args{
				i: "https://download.com/release-5.5/artifact/appgate-5.5.3-27278-release.img.zip?is-build-type-id",
			},
			wantErr: false,
		},
		{
			name: "test url with get variables key value",
			args: args{
				i: "https://download.com/release-5.5/artifact/appgate-5.5.3-27278-release.img.zip?foo=bar",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkImageFilename(tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("checkImageFilename() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_showPrepareUpgradeMessage(t *testing.T) {
	type args struct {
		f                             string
		appliance                     []openapi.Appliance
		skip                          []appliancepkg.SkipUpgrade
		stats                         []openapi.ApplianceWithStatus
		multiControllerUpgradeWarning bool
		dockerBundleDownload          bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "prepare appliance default",
			args: args{
				f: "appgate-6.0.0-29426-release.img.zip",
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
				},
				stats: []openapi.ApplianceWithStatus{
					{
						Id:               openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:             "controller1",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:             "controller2",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:             "gateway",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("92a8ceed-a364-4e99-a2eb-0a8546bab48f"),
						Name:             "controller3",
						Status:           openapi.PtrString("offline"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("57a06ae4-8204-4780-a7c2-a9cdf03e5a0f"),
						Name:             "gateway2",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("6.0.0+29426"),
					},
				},
				skip: []appliancepkg.SkipUpgrade{
					{
						Appliance: openapi.Appliance{
							Id:   openapi.PtrString("92a8ceed-a364-4e99-a2eb-0a8546bab48f"),
							Name: "controller3",
						},
						Reason: appliancepkg.ErrSkipReasonOffline,
					},
					{
						Appliance: openapi.Appliance{
							Id:   openapi.PtrString("57a06ae4-8204-4780-a7c2-a9cdf03e5a0f"),
							Name: "gateway2",
						},
						Reason: appliancepkg.ErrSkipReasonAlreadySameVersion,
					},
				},
			},
			want: `PREPARE SUMMARY

1. Upload upgrade image appgate-6.0.0-29426-release.img.zip to Controller
2. Prepare upgrade on the following appliances:

  Appliance      Online    Current version    Prepare version
  ---------      ------    ---------------    ---------------
  controller1    ✓         5.5.7+28767        6.0.0+29426
  controller2    ✓         5.5.7+28767        6.0.0+29426
  gateway        ✓         5.5.7+28767        6.0.0+29426


The following appliances will be skipped:

  Appliance      Online    Current version    Reason
  ---------      ------    ---------------    ------
  controller3    ⨯         5.5.7+28767        appliance is offline
  gateway2       ✓         6.0.0+29426        appliance is already running a version higher or equal to the prepare version

`,
		},
		{
			name: "prepare appliance no-skipped",
			args: args{
				f: "appgate-6.0.0-29426-release.img.zip",
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
				},
				stats: []openapi.ApplianceWithStatus{
					{
						Id:               openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:             "controller1",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:             "controller2",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:             "gateway",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
				},
			},
			want: `PREPARE SUMMARY

1. Upload upgrade image appgate-6.0.0-29426-release.img.zip to Controller
2. Prepare upgrade on the following appliances:

  Appliance      Online    Current version    Prepare version
  ---------      ------    ---------------    ---------------
  controller1    ✓         5.5.7+28767        6.0.0+29426
  controller2    ✓         5.5.7+28767        6.0.0+29426
  gateway        ✓         5.5.7+28767        6.0.0+29426

`,
		},
		{
			name: "prepare appliance no-skipped",
			args: args{
				f:                             "appgate-6.0.0-29426-release.img.zip",
				multiControllerUpgradeWarning: true,
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
				},
				stats: []openapi.ApplianceWithStatus{
					{
						Id:               openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:             "controller1",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:             "controller2",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
					{
						Id:               openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:             "gateway",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("5.5.7+28767"),
					},
				},
			},
			want: `PREPARE SUMMARY

1. Upload upgrade image appgate-6.0.0-29426-release.img.zip to Controller
2. Prepare upgrade on the following appliances:

  Appliance      Online    Current version    Prepare version
  ---------      ------    ---------------    ---------------
  controller1    ✓         5.5.7+28767        6.0.0+29426
  controller2    ✓         5.5.7+28767        6.0.0+29426
  gateway        ✓         5.5.7+28767        6.0.0+29426


WARNING: This upgrade requires all controllers to be upgraded to the same version, but not all
controllers are being prepared for upgrade.
A partial major or minor controller upgrade is not supported. The upgrade will fail unless all
controllers are prepared for upgrade when running 'upgrade complete'.
`,
		},
		{
			name: "prepare >=6.2 appliance with log server",
			args: args{
				f:                    "appgate-6.2.0-89012-release.img.zip",
				dockerBundleDownload: true,
				appliance: []openapi.Appliance{
					{
						Id:   openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name: "controller1",
					},
					{
						Id:   openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name: "controller2",
					},
					{
						Id:   openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name: "gateway",
					},
					{
						Id:   openapi.PtrString("3ab2caf1-2a6a-4e2c-a848-268d402492a1"),
						Name: "logserver",
					},
				},
				stats: []openapi.ApplianceWithStatus{
					{
						Id:               openapi.PtrString("d4dc0b97-ef59-4431-871b-6b214099797a"),
						Name:             "controller1",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("6.1.0+56789"),
					},
					{
						Id:               openapi.PtrString("3f6f9e42-33c3-446c-9e0d-855c7d5b933b"),
						Name:             "controller2",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("6.1.0+56789"),
					},
					{
						Id:               openapi.PtrString("8a064b81-c692-46ae-b0fa-c4661a018f24"),
						Name:             "gateway",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("6.1.0+56789"),
					},
					{
						Id:               openapi.PtrString("3ab2caf1-2a6a-4e2c-a848-268d402492a1"),
						Name:             "logserver",
						Status:           openapi.PtrString("healthy"),
						ApplianceVersion: openapi.PtrString("6.1.0+56789"),
					},
				},
			},
			want: `PREPARE SUMMARY

1. Bundle and upload LogServer docker image
2. Upload upgrade image appgate-6.2.0-89012-release.img.zip to Controller
3. Prepare upgrade on the following appliances:

  Appliance      Online    Current version    Prepare version
  ---------      ------    ---------------    ---------------
  controller1    ✓         6.1.0+56789        6.2.0+89012
  controller2    ✓         6.1.0+56789        6.2.0+89012
  gateway        ✓         6.1.0+56789        6.2.0+89012
  logserver      ✓         6.1.0+56789        6.2.0+89012

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareVersion, err := appliancepkg.ParseVersionString(tt.args.f)
			if err != nil {
				t.Fatalf("internal test error: %v", err)
			}
			got, err := showPrepareUpgradeMessage(tt.args.f, prepareVersion, tt.args.appliance, tt.args.skip, tt.args.stats, tt.args.multiControllerUpgradeWarning, tt.args.dockerBundleDownload)
			if (err != nil) != tt.wantErr {
				t.Errorf("showPrepareUpgradeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

// TLSRegistry wraps an HTTP test server with HTTPS capabilities for testing LogServer bundle downloads
type TLSRegistry struct {
	Client   *openapi.APIClient
	cfg      *openapi.Configuration
	Mux      *http.ServeMux
	server   *httptest.Server
	Port     int
	Teardown func()
	stubs    []*httpmock.Stub
	mu       sync.Mutex
	notFound []string
}

// newTLSRegistry creates a new TLS-enabled registry for HTTPS testing
func newTLSRegistry(t *testing.T) *TLSRegistry {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	clientCfg := openapi.NewConfiguration()
	clientCfg.HTTPClient = server.Client()
	c := openapi.NewAPIClient(clientCfg)

	port := server.Listener.Addr().(*net.TCPAddr).Port

	r := &TLSRegistry{
		Client:   c,
		cfg:      clientCfg,
		Mux:      mux,
		server:   server,
		Port:     port,
		Teardown: server.Close,
	}

	return r
}

// Register adds a stub to the TLS registry
func (r *TLSRegistry) Register(url string, resp http.HandlerFunc) {
	r.stubs = append(r.stubs, &httpmock.Stub{
		URL:       url,
		Responder: resp,
	})
}

// stubMiddleware tracks which stubs were matched (similar to httpmock.Registry)
func (r *TLSRegistry) stubMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		r.mu.Lock()
		next.ServeHTTP(rw, request)
		r.mu.Unlock()
	})
}

// Serve sets up the TLS server handlers
func (r *TLSRegistry) Serve() {
	for _, stub := range r.stubs {
		r.Mux.Handle(stub.URL, r.stubMiddleware(stub.Responder))
	}
	r.Mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			rw.WriteHeader(http.StatusNotFound)
			r.notFound = append(r.notFound, req.Method+" "+html.EscapeString(req.URL.Path))
			return
		}
	})
}

// URL returns the HTTPS URL of the TLS server
func (r *TLSRegistry) URL() string {
	return r.server.URL
}
