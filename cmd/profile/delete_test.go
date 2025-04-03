package profile

import (
	"bytes"
	"io"
	"testing"

	"github.com/appgate/sdpctl/pkg/profiles"
	"github.com/google/go-cmp/cmp"
)

func TestNewDeleteCmdNoProfilesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewDeleteCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"invalid_key"})

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	gotStr := string(got)
	want := "no profiles added\n"
	if gotStr != want {
		t.Fatal(cmp.Diff(want, gotStr))
	}
}

func TestNewDeleteCmdNotFound(t *testing.T) {
	setupExistingProfiles(t)

	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewDeleteCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"invalid_key"})

	_, err := cmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error, got none.")
	}
	want := "Did not find \"invalid_key\" as a existing profile"
	if err.Error() != want {
		t.Fatalf("Want: %s - got %s", want, err.Error())
	}
}

func TestDeleteProfile(t *testing.T) {
	setupExistingProfiles(t)
	if !profiles.FileExists() {
		t.Fatal("expected profile file to exists, found none")
	}
	// count number of profiles before deleting
	profilesPre, err := profiles.Read()
	if err != nil {
		t.Fatalf("could not read profiles.json %s", err)
	}
	countPre := len(profilesPre.List)
	if countPre != 2 {
		t.Fatalf("expect 2 profiles, 'default' and 'testing' got %d - %+v", len(profilesPre.List), profilesPre.List)
	}

	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewDeleteCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"staging"})

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("executeC %s", err)
	}

	post, err := profiles.Read()
	if err != nil {
		t.Fatalf("could not read profiles.json POST deletion %s", err)
	}
	if len(post.List) != countPre-1 {
		t.Error("expect 1 less profile in list, got same as pre deletion")
	}
}
