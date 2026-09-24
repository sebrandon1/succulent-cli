package cmd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
)

func TestRunAuditedOperationRecordsSuccessAndFailure(t *testing.T) {
	log, err := lib.NewAuditLog(filepath.Join(t.TempDir(), "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	previousLog := sharedAuditLog
	sharedAuditLog = log
	defer func() { sharedAuditLog = previousLog }()

	previousEnv := envName
	envName = "lab1"
	defer func() { envName = previousEnv }()
	t.Setenv("USER", "audit-user")

	if err := runAuditedOperation("delete", auditParameters("reason", "test"), func() error { return nil }); err != nil {
		t.Fatalf("Successful operation returned an error: %v", err)
	}
	wantErr := errors.New("server rejected request")
	err = runAuditedOperation("reprovision", nil, func() error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected operation error %v, got %v", wantErr, err)
	}

	entries, err := log.ReadLast(10)
	if err != nil {
		t.Fatalf("ReadLast failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected two audit entries, got %d", len(entries))
	}
	if entries[0].UserID != "audit-user" || entries[0].Operation != "delete" || entries[0].Environment != "lab1" || entries[0].Result != "success" {
		t.Errorf("Unexpected success entry: %+v", entries[0])
	}
	if entries[1].Operation != "reprovision" || entries[1].Result != "failure" || entries[1].Error != wantErr.Error() {
		t.Errorf("Unexpected failure entry: %+v", entries[1])
	}
}

func TestRunAuditedOperationForEnvironmentRecordsTarget(t *testing.T) {
	log, err := lib.NewAuditLog(filepath.Join(t.TempDir(), "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	previousLog, previousEnv := sharedAuditLog, envName
	sharedAuditLog = log
	envName = "default-env"
	t.Cleanup(func() {
		sharedAuditLog, envName = previousLog, previousEnv
	})

	if err := runAuditedOperationForEnvironment("group-env", "delete", nil, func() error { return nil }); err != nil {
		t.Fatalf("runAuditedOperationForEnvironment() error = %v", err)
	}
	entries, err := log.ReadLast(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Environment != "group-env" {
		t.Fatalf("audit entries = %+v, want the selected environment", entries)
	}
}

func TestAuditLogPathEnvironmentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-audit.jsonl")
	t.Setenv("SUCCULENT_AUDIT_LOG", path)
	if got := auditLogPath(); got != path {
		t.Errorf("Expected audit path %q, got %q", path, got)
	}
}

func TestConfigHistoryCommandRejectsInvalidLimit(t *testing.T) {
	previousLimit := historyLimit
	historyLimit = 0
	defer func() { historyLimit = previousLimit }()

	if err := configHistoryCmd.RunE(nil, nil); err == nil {
		t.Fatal("Expected an error for a non-positive history limit")
	}
}

func TestConfigHistoryRegistered(t *testing.T) {
	for _, command := range configCmd.Commands() {
		if command.Name() == "history" {
			return
		}
	}
	t.Fatal("Expected config history subcommand to be registered")
}

func TestConfigHistoryCommandReadsAuditLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	log, err := lib.NewAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Record(lib.AuditEntry{UserID: "alice", Operation: "delete", Environment: "lab1", Result: "success"}); err != nil {
		t.Fatal(err)
	}

	previousLog := sharedAuditLog
	sharedAuditLog = nil
	defer func() { sharedAuditLog = previousLog }()
	t.Setenv("SUCCULENT_AUDIT_LOG", path)
	rootCmd.SetArgs([]string{"config", "history", "--limit", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config history failed: %v", err)
	}
}

func TestFormatAuditDetails(t *testing.T) {
	entry := lib.AuditEntry{
		Parameters: map[string]string{"ocp_tag": "4.17", "owner": "alice"},
		Error:      "request failed\ncheck server",
	}
	if got, want := formatAuditDetails(entry), "ocp_tag=4.17; owner=alice; error=request failed check server"; got != want {
		t.Errorf("formatAuditDetails() = %q, want %q", got, want)
	}
}
