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
)

func TestServiceUsersUpdate(t *testing.T) {
	testCases := []serviceUsersTestStub{
		{
			desc: "update using file",
			args: []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", "--from-file=../../pkg/serviceusers/fixtures/service-user-update.json"},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
				},
			},
			wantOut: regexp.MustCompile(`"disabled": true`),
		},
		{
			desc:    "update using JSON formatted argument",
			args:    []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", `{"disabled": true}`},
			wantOut: regexp.MustCompile(`"disabled": true`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
				},
			},
		},
		{
			desc:    "disable user with argument",
			args:    []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", "disable"},
			wantOut: regexp.MustCompile(`"disabled": true`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
				},
			},
		},
		{
			desc:    "add user label",
			args:    []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", "add", "label", "somelabelkey=testlabel"},
			wantOut: regexp.MustCompile(`"somelabelkey": "testlabel"`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
				},
			},
		},
		{
			desc:    "add user tag",
			args:    []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", "add", "tag", "test-tag"},
			wantOut: regexp.MustCompile(`"test-tag"`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
				},
			},
		},
		{
			desc:       "invalid label arguments",
			args:       []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", "add", "label", "badlabel"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`no key or value provided for label`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
				},
			},
		},
		{
			desc:       "invalid tag arguments",
			args:       []string{"update", "1767c791-4001-429b-82cf-0a471cc2f5d2", "add", "tag"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`not enough arguments`),
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users/1767c791-4001-429b-82cf-0a471cc2f5d2",
					Responder: defaultUpdateResponseHandler,
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
				t.Fatalf("TestServiceUsersUpdate() error = %v, wantErr %v", err, tC.wantErr)
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

func defaultUpdateResponseHandler(w http.ResponseWriter, r *http.Request) {
	var filename string
	if r.Method == http.MethodGet {
		filename = "../../pkg/serviceusers/fixtures/service-user-get.json"
	}
	if r.Method == http.MethodPut {
		filename = "../../pkg/serviceusers/fixtures/service-user-update.json"
	}
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
