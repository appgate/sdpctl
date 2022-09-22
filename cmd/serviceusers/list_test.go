package serviceusers

import (
	"regexp"
	"testing"

	"github.com/appgate/sdpctl/pkg/httpmock"
	"github.com/stretchr/testify/assert"
)

var expectedTableView = `Name                 ID                                      Disabled    Tags    Modified
----                 --                                      --------    ----    --------
a-user               1767c791-4001-429b-82cf-0a471cc2f5d2    false               2022-09-16 08:19:06.516115 \+0000 UTC
some-other-user      9b3d3652-f36f-46fe-8065-4a3d3cee6ec4    false               2022-09-16 08:20:17.242287 \+0000 UTC
test-service-user    ff3d10ab-2474-4193-b670-86e230495188    false               2022-09-16 06:07:02.600422 \+0000 UTC
disabled-user        e857d763-37d6-4476-a8ce-092bf9ac8537    true                2022-09-16 06:07:02.600422 \+0000 UTC
`

var expectedJSONView = `[
  {
    "created": "2022-09-16T08:19:06.516115Z",
    "disabled": false,
    "failedLoginAttempts": 0,
    "id": "1767c791-4001-429b-82cf-0a471cc2f5d2",
    "labels": {},
    "name": "a-user",
    "notes": "",
    "tags": [],
    "updated": "2022-09-16T08:19:06.516115Z"
  },
  {
    "created": "2022-09-16T08:20:17.242287Z",
    "disabled": false,
    "failedLoginAttempts": 0,
    "id": "9b3d3652-f36f-46fe-8065-4a3d3cee6ec4",
    "labels": {},
    "name": "some-other-user",
    "notes": "",
    "tags": [],
    "updated": "2022-09-16T08:20:17.242287Z"
  },
  {
    "created": "2022-09-14T11:28:37.547326Z",
    "disabled": false,
    "failedLoginAttempts": 0,
    "id": "ff3d10ab-2474-4193-b670-86e230495188",
    "labels": {},
    "name": "test-service-user",
    "notes": "",
    "tags": [],
    "updated": "2022-09-16T06:07:02.600422Z"
  },
  {
    "created": "2022-09-14T11:28:37.547326Z",
    "disabled": true,
    "failedLoginAttempts": 0,
    "id": "e857d763-37d6-4476-a8ce-092bf9ac8537",
    "labels": {},
    "name": "disabled-user",
    "notes": "",
    "tags": [],
    "updated": "2022-09-16T06:07:02.600422Z"
  }
]
`

func TestServiceUsersList(t *testing.T) {
	testCases := []serviceUsersTestStub{
		{
			desc: "list table view",
			args: []string{"list"},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users",
					Responder: httpmock.JSONResponse("../../pkg/serviceusers/fixtures/service-users-list.json"),
				},
			},
			wantOut: regexp.MustCompile(expectedTableView),
		},
		{
			desc: "list json view",
			args: []string{"list", "--json"},
			httpStubs: []httpmock.Stub{
				{
					URL:       "/service-users",
					Responder: httpmock.JSONResponse("../../pkg/serviceusers/fixtures/service-users-list.json"),
				},
			},
			wantExactMatch: expectedJSONView,
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
			parsedOut := stdout.String()
			if tC.wantOut != nil {
				if !tC.wantOut.MatchString(parsedOut) {
					t.Fatalf("\n%s\n != \n%s", tC.wantOut.String(), parsedOut)
				}
			}
			if len(tC.wantExactMatch) > 0 {
				assert.Equal(t, tC.wantExactMatch, parsedOut)
			}
		})
	}
}
