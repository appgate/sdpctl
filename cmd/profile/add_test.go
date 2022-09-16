package profile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/appgate/sdpctl/pkg/util"
	"github.com/google/go-cmp/cmp"
)

func TestNewAddCmdWithExistingProfiles(t *testing.T) {
	setupExistingProfiles(t)

	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewAddCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"testing"})
	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	gotStr := string(got)

	want := `Created profile testing, run 'sdpctl profile list' to see all available profiles
run 'sdpctl profile set testing' to select the new profile profile
`

	if diff := cmp.Diff(want, gotStr); diff != "" {
		t.Errorf("List output mismatch (-want +got):\n%s", diff)
	}
}

func TestNewAddCmdDuplicateName(t *testing.T) {
	setupExistingProfiles(t)

	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewAddCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"production"})

	_, err := cmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error, got none.")
	}
	want := "profile already exists with the name production"
	if err.Error() != want {
		t.Fatalf("Want: %s - got %s", want, err.Error())
	}
}

func TestNewAddCmdMigrateExistingRootConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	// create a root level config, to make sure we can migrate this to new default profile
	rootConfigFile := filepath.Join(dir, "config.json")
	profileFile, err := os.Create(rootConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	defer profileFile.Close()
	data := `{ "url": "https://appgate.controller.com:444/admin" }`
	if _, err := profileFile.WriteString(data); err != nil {
		t.Fatal(err)
	}
	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewAddCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"europe"})
	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	gotStr := string(got)

	want := `Created profile europe, run 'sdpctl profile list' to see all available profiles
run 'sdpctl profile set europe' to select the new profile profile
`

	if diff := cmp.Diff(want, gotStr); diff != "" {
		t.Errorf("List output mismatch (-want +got):\n%s", diff)
	}
	if !profiles.FileExists() {
		t.Fatal("expected profile file to exists, found none")
	}
	p, err := profiles.Read()
	if err != nil {
		t.Fatalf("could not read profiles.json %s", err)
	}
	if len(p.List) != 2 {
		t.Fatalf("expect 2 profiles, 'default' and 'testing' got %d - %+v", len(p.List), p.List)
	}

	ok, err := util.FileExists(rootConfigFile)
	if err == nil && ok {
		t.Fatal("Expect no config.json in root level, found one")
	}
}
