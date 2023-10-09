package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/google/go-cmp/cmp"
)

func TestNewListCmdNoProfiles(t *testing.T) {
	t.Setenv("SDPCTL_CONFIG_DIR", t.TempDir())
	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewListCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	gotStr := string(got)
	want := "no profiles added\n"
	if !cmp.Equal(want, gotStr) {
		t.Fatalf("\nGot: \n %q \n\n Want: \n %q \n", gotStr, want)
	}
}

var resetRead = func() {
	profiles.ReadProfiles = nil
}

func setupExistingProfiles(t *testing.T) (string, string) {
	t.Helper()
	t.Cleanup(resetRead)
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	logs := t.TempDir()
	t.Setenv("SDPCTL_DATA_DIR", logs)
	profileFile, err := os.Create(profiles.FilePath())
	if err != nil {
		t.Fatalf("failed to create testing profile %s", err)
	}
	defer profileFile.Close()
	if err := profiles.CreateConfigDirectory(); err != nil {
		t.Fatal(err)
	}

	p := profiles.Profiles{}
	names := []string{"staging", "production"}
	for _, name := range names {
		cfgDir, _ := profiles.Directories()
		directory := filepath.Join(cfgDir, name)
		if err := os.Mkdir(directory, os.ModePerm); err != nil {
			t.Fatalf("profile already exists with the name %s", name)
		}

		p.List = append(p.List, profiles.Profile{
			Name:      name,
			LogPath:   filepath.Join(logs, name+".log"),
			Directory: directory,
		})
		p.Current = &name
	}

	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profileFile.Write(bytes); err != nil {
		t.Fatalf("could not write testing profile %s", err)
	}
	return dir, logs
}

func Nprintf(format string, params map[string]interface{}) string {
	for key, val := range params {
		format = strings.Replace(format, "%{"+key+"}", fmt.Sprintf("%v", val), -1)
	}
	return format
}

// TODO: Fix this test, it's failing sporadically
// func TestNewListCmdTwoProfilesOneCurrent(t *testing.T) {
// 	dir, logs := setupExistingProfiles(t)

// 	stdout := &bytes.Buffer{}
// 	opts := &commandOpts{
// 		Out: stdout,
// 	}
// 	cmd := NewListCmd(opts)
// 	cmd.SetOut(io.Discard)
// 	cmd.SetErr(io.Discard)

// 	if _, err := cmd.ExecuteC(); err != nil {
// 		t.Fatalf("executeC %s", err)
// 	}

// 	got, err := io.ReadAll(stdout)
// 	if err != nil {
// 		t.Fatalf("unable to read stdout %s", err)
// 	}
// 	gotStr := string(got)
// 	params := map[string]interface{}{
// 		"dir":  dir,
// 		"logs": logs,
// 	}

// 	want := Nprintf(`Current profile production is not configured, run 'sdpctl configure'

// Available profiles
// Name          Config directory                                                              Log file path
// ----          ----------------                                                              -------------
// staging       %{dir}/profiles/staging       %{logs}/staging.log
// production    %{dir}/profiles/production    %{logs}/production.log
// `, params)

// 	if diff := cmp.Diff(want, gotStr); diff != "" {
// 		t.Errorf("List output mismatch (-want +got):\n%s", diff)
// 	}
// }
