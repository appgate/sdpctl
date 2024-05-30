package serviceusers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/appgate/sdpctl/pkg/prompt"
)

func TestServiceUsersCreate(t *testing.T) {
	testCases := []serviceUsersTestStub{
		{
			desc: "create using prompt",
			args: []string{"create"},
			askStubs: func(as *prompt.AskStubber) {
				as.StubPrompt("Name for service user:").AnswerWith("test-service-user")
				as.StubPrompt("Passphrase for service user:").AnswerWith("password")
				as.StubPrompt("Confirm your passphrase:").AnswerWith("password")
			},
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/service-users",
					Responder: func(rw http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							filename := "../../pkg/serviceusers/fixtures/service-user-create.json"
							f, err := os.Open(filename)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not open %q", filename))
							}
							defer f.Close()
							rw.Header().Set("Content-Type", "application/json")
							rw.WriteHeader(http.StatusOK)
							reader := bufio.NewReader(f)
							content, err := io.ReadAll(reader)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not read %q", filename))
							}
							fmt.Fprint(rw, string(content))
						}
					},
				},
			},
			wantOut: regexp.MustCompile(`"name": "test-service-user"`),
		},
		{
			desc:       "no arguments no-interactive",
			args:       []string{"create", "--no-interactive"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`failed to create user: missing data`),
		},
		{
			desc:    "create with flags",
			args:    []string{"create", "--name=test-service-user", "--tags=one,two,three"},
			wantOut: regexp.MustCompile(`"name": "test-service-user"`),
			askStubs: func(as *prompt.AskStubber) {
				as.StubPrompt("Passphrase for service user:").AnswerWith("passphrase")
				as.StubPrompt("Confirm your passphrase:").AnswerWith("passphrase")
			},
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/service-users",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							filename := "../../pkg/serviceusers/fixtures/service-user-create.json"
							f, err := os.Open(filename)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not open %q", filename))
							}
							defer f.Close()
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							reader := bufio.NewReader(f)
							content, err := io.ReadAll(reader)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not read %q", filename))
							}
							fmt.Fprint(w, string(content))
						}
					},
				},
			},
		},
		{
			desc:    "create with only username flag",
			args:    []string{"create", "--name=test-service-user"},
			wantOut: regexp.MustCompile(`"name": "test-service-user"`),
			askStubs: func(as *prompt.AskStubber) {
				as.StubPrompt("Passphrase for service user:").AnswerWith("password")
				as.StubPrompt("Confirm your passphrase:").AnswerWith("password")
			},
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/service-users",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							filename := "../../pkg/serviceusers/fixtures/service-user-create.json"
							f, err := os.Open(filename)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not open %q", filename))
							}
							defer f.Close()
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							reader := bufio.NewReader(f)
							content, err := io.ReadAll(reader)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not read %q", filename))
							}
							fmt.Fprint(w, string(content))
						}
					},
				},
			},
		},
		{
			desc:    "create with only tags flag",
			args:    []string{"create", "--tags=one,two,three"},
			wantOut: regexp.MustCompile(`"name": "test-service-user"`),
			askStubs: func(as *prompt.AskStubber) {
				as.StubPrompt("Name for service user:").AnswerWith("test-service-user")
				as.StubPrompt("Passphrase for service user:").AnswerWith("password")
				as.StubPrompt("Confirm your passphrase:").AnswerWith("password")
			},
			httpStubs: []httpmock.Stub{
				{
					URL: "/admin/service-users",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							filename := "../../pkg/serviceusers/fixtures/service-user-create.json"
							f, err := os.Open(filename)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not open %q", filename))
							}
							defer f.Close()
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							reader := bufio.NewReader(f)
							content, err := io.ReadAll(reader)
							if err != nil {
								panic(fmt.Sprintf("Internal testing error: could not read %q", filename))
							}
							fmt.Fprint(w, string(content))
						}
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cmd, registry, stdout, teardown := setupServiceUsersTest(t, &tC)
			defer func() {
				registry.Teardown()
				teardown()
			}()

			_, err := cmd.ExecuteC()
			if (err != nil) != tC.wantErr {
				t.Logf("Stdout: %s", stdout)
				t.Fatalf("TestServiceUsersCreate() error = %v, wantErr %v", err, tC.wantErr)
			}
			if (err != nil) == tC.wantErr && tC.wantErrOut != nil {
				errString := err.Error()
				matchString := tC.wantErrOut.String()
				if !tC.wantErrOut.MatchString(errString) {
					t.Fatalf("\n%s\n != \n%s", errString, matchString)
				}
			}
			if tC.wantOut != nil {
				parsedOut := stdout.String()
				if !tC.wantOut.MatchString(parsedOut) {
					t.Fatalf("\n%s\n != \n%s", tC.wantOut.String(), parsedOut)
				}
			}
		})
	}
}
