package appliance

import (
	"archive/zip"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// TestExtractLogsPutsDaemonLogsInSubdir verifies that processJournalFile writes
// per-daemon log files into a logs_by_daemon/ subdirectory instead of dumping
// them flat into the output root.
func TestExtractLogsPutsDaemonLogsInSubdir(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	zipPath := filepath.Join(src, "bundle.zip")

	// Build a minimal systemd journal binary containing one entry with
	// SYSLOG_IDENTIFIER=sshd and MESSAGE=accepted publickey.
	journalPath := filepath.Join(src, "test.journal")
	buildMinimalJournal(t, journalPath, "sshd", "accepted publickey")

	journalBytes, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create a zip with one plain file and one .journal file.
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)

	pw, err := zw.Create("system-info.txt")
	if err != nil {
		t.Fatal(err)
	}
	pw.Write([]byte("info"))

	jw, err := zw.Create("abc123/system.journal")
	if err != nil {
		t.Fatal(err)
	}
	jw.Write(journalBytes)

	zw.Close()
	zf.Close()

	if err := processJournalFile(zipPath, out, false); err != nil {
		t.Fatalf("processJournalFile: %v", err)
	}

	// system-info.txt must be at the output root.
	if _, err := os.Stat(filepath.Join(out, "system-info.txt")); err != nil {
		t.Errorf("expected system-info.txt at root: %v", err)
	}

	// Per-daemon logs must be inside logs_by_daemon/, not at root.
	byDaemon := filepath.Join(out, "logs_by_daemon")
	if _, err := os.Stat(byDaemon); err != nil {
		t.Fatalf("expected logs_by_daemon/ directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(byDaemon, "sshd.log")); err != nil {
		t.Errorf("expected logs_by_daemon/sshd.log: %v", err)
	}

	// sshd.log must NOT exist at root — that was the old broken behaviour.
	if _, err := os.Stat(filepath.Join(out, "sshd.log")); err == nil {
		t.Errorf("sshd.log should not be at root, expected it under logs_by_daemon/")
	}
}

// buildMinimalJournal creates a minimal valid systemd journal binary file
// containing a single entry with two fields: SYSLOG_IDENTIFIER and MESSAGE.
//
// The binary layout follows the systemd journal format used by journaldreader:
//
//	[  0, 208)  Header
//	[208, 296)  DataObject  "SYSLOG_IDENTIFIER=<identifier>"
//	[296, 392)  DataObject  "MESSAGE=<message>"
//	[392, 488)  EntryObject (2 data items, non-compact 16-byte refs)
//	[488, 520)  EntryArrayObject (1 entry item, non-compact 8-byte ref)
func buildMinimalJournal(t *testing.T, path, identifier, message string) {
	t.Helper()

	field1 := []byte("SYSLOG_IDENTIFIER=" + identifier)
	field2 := []byte("MESSAGE=" + message)

	align8 := func(n int) int { return (n + 7) &^ 7 }

	const (
		headerSize        = 208
		dataObjectSize    = 64
		entryObjectSize   = 64
		entryArrayObjSize = 24
		typeData          = 1
		typeEntry         = 3
		typeEntryArray    = 6
	)

	// Calculate sizes and offsets (all 8-byte aligned).
	data1Off := headerSize
	data1Padded := align8(dataObjectSize + len(field1))

	data2Off := data1Off + data1Padded
	data2Padded := align8(dataObjectSize + len(field2))

	entryOff := data2Off + data2Padded
	entrySize := entryObjectSize + 2*16 // 2 data refs, 16 bytes each (non-compact)

	eaOff := entryOff + entrySize
	eaSize := entryArrayObjSize + 8 // 1 entry ref, 8 bytes (non-compact)

	totalSize := eaOff + align8(eaSize)
	buf := make([]byte, totalSize)

	// bytecopy is used instead of the builtin copy, which is shadowed by a
	// package-level function in logs.go.
	bytecopy := func(dst, src []byte) {
		for i, b := range src {
			dst[i] = b
		}
	}

	// ---- Header ----
	bytecopy(buf[0:8], []byte("LPKSHHRH"))
	// incompatible_flags = 0  → non-compact, no compression
	binary.LittleEndian.PutUint64(buf[88:], headerSize)                   // header_size
	binary.LittleEndian.PutUint64(buf[96:], uint64(totalSize-headerSize)) // arena_size
	binary.LittleEndian.PutUint64(buf[136:], uint64(eaOff))               // tail_object_offset
	binary.LittleEndian.PutUint64(buf[144:], 4)                           // n_objects
	binary.LittleEndian.PutUint64(buf[152:], 1)                           // n_entries
	binary.LittleEndian.PutUint64(buf[160:], 1)                           // tail_entry_seqnum
	binary.LittleEndian.PutUint64(buf[168:], 1)                           // head_entry_seqnum
	binary.LittleEndian.PutUint64(buf[176:], uint64(eaOff))               // entry_array_offset

	// ---- DataObject 1: SYSLOG_IDENTIFIER ----
	buf[data1Off] = typeData
	binary.LittleEndian.PutUint64(buf[data1Off+8:], uint64(dataObjectSize+len(field1)))
	bytecopy(buf[data1Off+dataObjectSize:], field1)

	// ---- DataObject 2: MESSAGE ----
	buf[data2Off] = typeData
	binary.LittleEndian.PutUint64(buf[data2Off+8:], uint64(dataObjectSize+len(field2)))
	bytecopy(buf[data2Off+dataObjectSize:], field2)

	// ---- EntryObject (2 data items) ----
	buf[entryOff] = typeEntry
	binary.LittleEndian.PutUint64(buf[entryOff+8:], uint64(entrySize))
	binary.LittleEndian.PutUint64(buf[entryOff+16:], 1) // seqnum
	// Items at entryOff+64: each is [offset uint64, hash uint64]
	binary.LittleEndian.PutUint64(buf[entryOff+64:], uint64(data1Off))
	binary.LittleEndian.PutUint64(buf[entryOff+80:], uint64(data2Off))

	// ---- EntryArrayObject (1 entry) ----
	buf[eaOff] = typeEntryArray
	binary.LittleEndian.PutUint64(buf[eaOff+8:], uint64(eaSize))
	// next_entry_array_offset at eaOff+16: 0 (no continuation)
	binary.LittleEndian.PutUint64(buf[eaOff+24:], uint64(entryOff))

	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatal(err)
	}
}
