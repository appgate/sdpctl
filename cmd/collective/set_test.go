package collective

import (
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewSetCmdNoProfilesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDPCTL_CONFIG_DIR", dir)
	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewSetCmd(opts)
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
	want := `no profiles added
run 'sdpctl collective add' first
`
	if gotStr != want {
		t.Fatalf(cmp.Diff(want, gotStr))
	}
}

func TestNewSetCmdSetValid(t *testing.T) {
	dir := setupExistingProfiles(t)

	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewSetCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"staging"})
	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("executeC %s", err)
	}

	got, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("unable to read stdout %s", err)
	}
	gotStr := string(got)
	want := Nprintf(`staging (%{dir}/profiles/staging) is selected as current sdp collective profile
staging is not configured yet, run 'sdpctl configure'
`, map[string]interface{}{"dir": dir})

	if diff := cmp.Diff(want, gotStr); diff != "" {
		t.Errorf("List output mismatch (-want +got):\n%s", diff)
	}
}

func TestNewSetCmdSetNotFound(t *testing.T) {
	setupExistingProfiles(t)

	stdout := &bytes.Buffer{}
	opts := &commandOpts{
		Out: stdout,
	}
	cmd := NewSetCmd(opts)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"pink"})
	_, err := cmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error, got none")
	}

	want := "Profile pink not found in [staging production]"
	if err.Error() != want {
		t.Errorf("want %s, got %s", want, err.Error())
	}
}
