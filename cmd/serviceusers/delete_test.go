package serviceusers

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/appgate/sdpctl/pkg/httpmock"
)

func TestServiceUsersDeleteCMD(t *testing.T) {
	testCases := []serviceUsersTestStub{
		{
			desc:    "delete single user with argument",
			args:    []string{"delete", "e857d763-37d6-4476-a8ce-092bf9ac8537"},
			wantOut: regexp.MustCompile(`deleted: e857d763-37d6-4476-a8ce-092bf9ac8537`),
			httpStubs: []httpmock.Stub{
				{
					URL: "/service-users/e857d763-37d6-4476-a8ce-092bf9ac8537",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							w.WriteHeader(http.StatusAccepted)
						}
					},
				},
			},
		},
		{
			desc: "delete multiple users with arguments",
			args: []string{"delete", "e857d763-37d6-4476-a8ce-092bf9ac8537", "ff3d10ab-2474-4193-b670-86e230495188"},
			wantOut: regexp.MustCompile(`deleted: e857d763-37d6-4476-a8ce-092bf9ac8537
deleted: ff3d10ab-2474-4193-b670-86e230495188`),
			httpStubs: []httpmock.Stub{
				{
					URL: "/service-users/e857d763-37d6-4476-a8ce-092bf9ac8537",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							w.WriteHeader(http.StatusAccepted)
						}
					},
				},
				{
					URL: "/service-users/ff3d10ab-2474-4193-b670-86e230495188",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							w.WriteHeader(http.StatusAccepted)
						}
					},
				},
			},
		},
		{
			desc:       "delete non-existing user",
			args:       []string{"delete", "ee3d10ab-2474-4193-b670-86e230495188"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile("404 Not Found"),
			httpStubs: []httpmock.Stub{
				{
					URL: "/service-users/ee3d10ab-2474-4193-b670-86e230495188",
					Responder: func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							w.WriteHeader(http.StatusNotFound)
						}
					},
				},
			},
		},
		{
			desc:       "invalid UUID",
			args:       []string{"delete", "ldskfjslkdjgfls"},
			wantErr:    true,
			wantErrOut: regexp.MustCompile("argument is not a valid UUID"),
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
				t.Fatalf("TestServiceUsersDelete() error = %v, wantErr %v", err, tC.wantErr)
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
