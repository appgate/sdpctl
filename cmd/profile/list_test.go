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

func setupExistingProfiles(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	profileFile, err := os.Create(profiles.FilePath())
	if err != nil {
		t.Fatalf("could not create testing profile %s", err)
	}
	defer profileFile.Close()
	if err := profiles.CreateDirectories(); err != nil {
		t.Fatal(err)
	}

	p := profiles.Profiles{}
	names := []string{"staging", "production"}
	for _, name := range names {
		directory := filepath.Join(profiles.Directories(), name)
		if err := os.Mkdir(directory, os.ModePerm); err != nil {
			t.Fatalf("profile already exists with the name %s", name)
		}

		p.List = append(p.List, profiles.Profile{
			Name:      name,
			Directory: directory,
		})
		p.Current = &directory
	}

	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profileFile.Write(bytes); err != nil {
		t.Fatalf("could not write testing profile %s", err)
	}
	return dir

}

func Nprintf(format string, params map[string]interface{}) string {
	for key, val := range params {
		format = strings.Replace(format, "%{"+key+"}", fmt.Sprintf("%v", val), -1)
	}
	return format
}

func TestNewListCmdTwoProfilesOneCurrent(t *testing.T) {
	dir := setupExistingProfiles(t)

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
	params := map[string]interface{}{
		"dir": dir,
	}

	want := Nprintf(`Current profile production is not configure, run 'sdpctl configure'

Available profiles
Name          Config directory
----          ----------------
staging       %{dir}/profiles/staging
production    %{dir}/profiles/production
`, params)

	if diff := cmp.Diff(want, gotStr); diff != "" {
		t.Errorf("List output mismatch (-want +got):\n%s", diff)
	}
}
