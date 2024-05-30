package serviceusers

import (
	"regexp"
	"testing"

	"github.com/appgate/sdpctl/pkg/httpmock"
)

func TestServiceUsersGet(t *testing.T) {
	testCases := []serviceUsersTestStub{
		{
			desc: "get with id arg",
			args: []string{"get", "068d9e30-7847-48c3-a88e-5fa7c0964288"},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/admin/service-users/068d9e30-7847-48c3-a88e-5fa7c0964288",
					Responder: httpmock.JSONResponse("../../pkg/serviceusers/fixtures/service-user-create.json"),
				},
			},
			wantOut: regexp.MustCompile(`"id": "068d9e30-7847-48c3-a88e-5fa7c0964288"`),
		},
		{
			desc:       "no id arg",
			args:       []string{"get"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(`accepts 1 arg\(s\), received 0`),
		},
		{
			desc:       "invalid uuid arg",
			args:       []string{"get", "sakdslkjfalfja"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile(InvalidUUIDError),
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
				t.Fatalf("TestServiceUsersGet() error = %v, wantErr %v", err, tC.wantErr)
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
