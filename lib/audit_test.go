package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAuditLogRecordAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "audit.log")
	log, err := NewAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}

	when := time.Date(2026, time.September, 24, 12, 30, 0, 0, time.FixedZone("test", -5*60*60))
	entry := AuditEntry{
		Timestamp:   when,
		UserID:      "alice",
		Operation:   "ztp provision",
		Environment: "lab1",
		Parameters:  map[string]string{"type": "sno"},
		Result:      "failure",
		Error:       "server unavailable",
	}
	if err := log.Record(entry); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	line := string(data)
	if strings.Count(line, "\n") != 1 {
		t.Fatalf("Expected one JSONL line, got %q", line)
	}
	assertJSONFieldOrder(t, line, []string{`"timestamp"`, `"user_id"`, `"operation"`, `"environment"`, `"parameters"`, `"result"`, `"error"`})

	var got AuditEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Invalid JSONL entry: %v", err)
	}
	if !got.Timestamp.Equal(when) || got.UserID != entry.UserID || got.Operation != entry.Operation ||
		got.Environment != entry.Environment || got.Result != entry.Result || got.Error != entry.Error ||
		got.Parameters["type"] != "sno" {
		t.Errorf("Read entry mismatch: got %+v", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0o600 {
		t.Errorf("Expected audit log mode 0600, got %04o", gotMode)
	}
}

func assertJSONFieldOrder(t *testing.T, line string, fieldOrder []string) {
	t.Helper()
	lastIndex := -1
	for _, field := range fieldOrder {
		index := strings.Index(line, field)
		if index <= lastIndex {
			t.Fatalf("Expected JSON fields in order %v, got %q", fieldOrder, line)
		}
		lastIndex = index
	}
}

func TestAuditLogReadLast(t *testing.T) {
	log, err := NewAuditLog(filepath.Join(t.TempDir(), "audit.log"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 4; i++ {
		if err := log.Record(AuditEntry{Operation: "operation", Environment: strconv.Itoa(i), Result: "success"}); err != nil {
			t.Fatalf("Record %d failed: %v", i, err)
		}
	}

	entries, err := log.ReadLast(2)
	if err != nil {
		t.Fatalf("ReadLast failed: %v", err)
	}
	if len(entries) != 2 || entries[0].Environment != "3" || entries[1].Environment != "4" {
		t.Errorf("Expected last entries [3, 4], got %+v", entries)
	}

	all, err := log.ReadLast(10)
	if err != nil || len(all) != 4 {
		t.Errorf("Expected all 4 entries, got %d (err %v)", len(all), err)
	}
}

func TestAuditLogReadMissingFile(t *testing.T) {
	log, err := NewAuditLog(filepath.Join(t.TempDir(), "missing.log"))
	if err != nil {
		t.Fatal(err)
	}

	entries, err := log.ReadLast(10)
	if err != nil || len(entries) != 0 {
		t.Errorf("Expected empty history for a missing file, got %v (err %v)", entries, err)
	}
}

func TestNewAuditLogRequiresPath(t *testing.T) {
	if _, err := NewAuditLog(""); err == nil {
		t.Fatal("Expected an error for an empty audit log path")
	}
}

func TestAuditLogReadLastRejectsMalformedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	log, err := NewAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := log.ReadLast(10); err == nil || !strings.Contains(err.Error(), "decoding audit entry") {
		t.Fatalf("Expected malformed entry error, got %v", err)
	}
}

func TestAuditLogReadLastNonPositiveLimit(t *testing.T) {
	log, err := NewAuditLog(filepath.Join(t.TempDir(), "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := log.ReadLast(0)
	if err != nil || len(entries) != 0 {
		t.Fatalf("Expected empty history for zero limit, got %v (err %v)", entries, err)
	}
}

func TestAuditLogRecordReportsInvalidDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file", "audit.log")
	if err := os.WriteFile(filepath.Dir(path), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	log, err := NewAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Record(AuditEntry{Operation: "delete", Result: "success"}); err == nil {
		t.Fatal("Expected an error when the audit log directory is a file")
	}
}
