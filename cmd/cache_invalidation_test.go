package cmd

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sebrandon1/succulent-cli/lib"
)

func preserveMutationFlagState(t *testing.T) {
	t.Helper()
	previousConfirmDelete, previousDryRunDelete := confirmDelete, dryRunDelete
	previousConfirmReprovision, previousDryRunReprovision := confirmReprovision, dryRunReprovision
	previousConfirmSNO, previousDryRunSNO := confirmSNO, dryRunSNO
	previousConfirmZTP, previousDryRunZTP := confirmZTP, dryRunZTP
	previousConfirmHS, previousDryRunHS := confirmHS, dryRunHS
	previousWatch := watchAfterProvision
	t.Cleanup(func() {
		confirmDelete, dryRunDelete = previousConfirmDelete, previousDryRunDelete
		confirmReprovision, dryRunReprovision = previousConfirmReprovision, previousDryRunReprovision
		confirmSNO, dryRunSNO = previousConfirmSNO, previousDryRunSNO
		confirmZTP, dryRunZTP = previousConfirmZTP, previousDryRunZTP
		confirmHS, dryRunHS = previousConfirmHS, previousDryRunHS
		watchAfterProvision = previousWatch
	})
}

func TestMutationCommandsInvalidateOnlyAfterServerSuccess(t *testing.T) {
	preserveMutationFlagState(t)
	commands := []struct {
		name string
		args []string
	}{
		{name: "delete", args: []string{"delete", "--env", "testenv", "--confirm", "--dry-run=false"}},
		{name: "MNO reprovision", args: []string{"reprovision", "--env", "testenv", "--email", "test@example.com", "--owner", "testuser", "--ocp-tag", "4.17", "--confirm", "--dry-run=false", "--watch=false"}},
		{name: "SNO provision", args: []string{"sno", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--ocp-tag", "4.17", "--confirm", "--dry-run=false", "--watch=false"}},
		{name: "ZTP provision", args: []string{"ztp", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--spoke-tag", "4.17", "--confirm", "--dry-run=false", "--watch=false"}},
		{name: "Hypershift provision", args: []string{"hypershift", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--hcp-tag", "4.17", "--confirm", "--dry-run=false", "--watch=false"}},
	}

	for _, command := range commands {
		for _, success := range []bool{true, false} {
			caseName := "server failure"
			if success {
				caseName = "server success"
			}
			t.Run(command.name+"/"+caseName, func(t *testing.T) {
				wantCacheHit := !success
				previousEnv, previousTargets := envName, selectedEnvironments
				previousCache, previousClient, previousAudit := sharedCache, sharedClient, sharedAuditLog
				t.Cleanup(func() {
					envName, selectedEnvironments = previousEnv, previousTargets
					sharedCache, sharedClient, sharedAuditLog = previousCache, previousClient, previousAudit
				})
				t.Setenv("HOME", t.TempDir())

				cache := lib.NewCache(configDir(), time.Minute)
				cache.SetInfo("testenv", &lib.ClusterInfo{Environment: "testenv", InstallerIP: "192.0.2.10"})
				cache.SetInfo("otherenv", &lib.ClusterInfo{Environment: "otherenv", InstallerIP: "192.0.2.20"})

				status := http.StatusOK
				if !success {
					status = http.StatusInternalServerError
				}
				cleanupServer := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(status)
				})
				defer cleanupServer()

				rootCmd.SetArgs(command.args)
				err := rootCmd.Execute()
				if success && err != nil {
					t.Fatalf("Execute() error = %v, want success", err)
				}
				if !success && err == nil {
					t.Fatal("Execute() error = nil, want server failure")
				}

				if _, ok := sharedCache.GetInfo("testenv"); ok != wantCacheHit {
					t.Fatalf("testenv cache hit = %t, want %t", ok, wantCacheHit)
				}
				if _, ok := sharedCache.GetInfo("otherenv"); !ok {
					t.Fatal("unrelated environment cache entry was invalidated")
				}
			})
		}
	}
}

func TestInvalidateCachedInfoWithoutSharedCache(t *testing.T) {
	previousCache := sharedCache
	sharedCache = nil
	t.Cleanup(func() { sharedCache = previousCache })

	invalidateCachedInfo("testenv")
}

func TestMutationCommandDryRunsPreserveCache(t *testing.T) {
	preserveMutationFlagState(t)

	commands := []struct {
		name string
		args []string
	}{
		{name: "delete", args: []string{"delete", "--env", "testenv", "--dry-run"}},
		{name: "MNO reprovision", args: []string{"reprovision", "--env", "testenv", "--email", "test@example.com", "--owner", "testuser", "--ocp-tag", "4.17", "--confirm", "--dry-run", "--watch=false"}},
		{name: "SNO provision", args: []string{"sno", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--ocp-tag", "4.17", "--confirm", "--dry-run", "--watch=false"}},
		{name: "ZTP provision", args: []string{"ztp", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--spoke-tag", "4.17", "--confirm", "--dry-run", "--watch=false"}},
		{name: "Hypershift provision", args: []string{"hypershift", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--hcp-tag", "4.17", "--confirm", "--dry-run", "--watch=false"}},
	}

	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			previousEnv, previousTargets := envName, selectedEnvironments
			previousCache, previousClient, previousAudit := sharedCache, sharedClient, sharedAuditLog
			t.Cleanup(func() {
				envName, selectedEnvironments = previousEnv, previousTargets
				sharedCache, sharedClient, sharedAuditLog = previousCache, previousClient, previousAudit
			})
			t.Setenv("HOME", t.TempDir())

			cache := lib.NewCache(configDir(), time.Minute)
			cache.SetInfo("testenv", &lib.ClusterInfo{Environment: "testenv"})
			requests := 0
			cleanupServer := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
				requests++
				w.WriteHeader(http.StatusOK)
			})
			defer cleanupServer()

			rootCmd.SetArgs(command.args)
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("dry-run Execute() error = %v", err)
			}
			if requests != 0 {
				t.Fatalf("server requests = %d, want none for dry run", requests)
			}
			if _, ok := sharedCache.GetInfo("testenv"); !ok {
				t.Fatal("dry run invalidated the cache entry")
			}
		})
	}
}

func TestDeleteInvalidationMakesDetailedReadFetchFreshInfo(t *testing.T) {
	previousEnv, previousTargets := envName, selectedEnvironments
	previousCache, previousClient, previousAudit := sharedCache, sharedClient, sharedAuditLog
	t.Cleanup(func() {
		envName, selectedEnvironments = previousEnv, previousTargets
		sharedCache, sharedClient, sharedAuditLog = previousCache, previousClient, previousAudit
	})
	t.Setenv("HOME", t.TempDir())

	cache := lib.NewCache(configDir(), time.Minute)
	cache.SetInfo("testenv", &lib.ClusterInfo{Environment: "testenv", InstallerIP: "192.0.2.10"})
	infoRequests := 0
	cleanupServer := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/infoplan/testenv" {
			infoRequests++
			w.Write([]byte(strings.Replace(testInfoHTML, "192.168.1.100", "203.0.113.10", 1)))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer cleanupServer()

	rootCmd.SetArgs([]string{"delete", "--env", "testenv", "--confirm", "--dry-run=false"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete Execute() error = %v", err)
	}
	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv", "--output", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("get info Execute() error = %v", err)
	}

	if infoRequests != 1 {
		t.Fatalf("info requests = %d, want 1 fresh request", infoRequests)
	}
	got, ok := sharedCache.GetInfo("testenv")
	if !ok || got.InstallerIP != "203.0.113.10" {
		t.Fatalf("cached info after fresh read = %+v, hit = %t; want updated installer IP", got, ok)
	}
}

func TestDeleteCachePersistenceFailureWarnsWithoutFailingAcceptedMutation(t *testing.T) {
	previousEnv, previousTargets := envName, selectedEnvironments
	previousCache, previousClient, previousAudit := sharedCache, sharedClient, sharedAuditLog
	previousConfirm, previousDryRun := confirmDelete, dryRunDelete
	previousLogger := slog.Default()
	previousContext := deleteCmd.Context()
	t.Cleanup(func() {
		envName, selectedEnvironments = previousEnv, previousTargets
		sharedCache, sharedClient, sharedAuditLog = previousCache, previousClient, previousAudit
		confirmDelete, dryRunDelete = previousConfirm, previousDryRun
		slog.SetDefault(previousLogger)
		deleteCmd.SetContext(previousContext)
	})

	accepted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accepted = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := lib.NewClient(server.URL, true, "")
	if err != nil {
		t.Fatal(err)
	}
	auditLog, err := lib.NewAuditLog(filepath.Join(t.TempDir(), "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	blockedCacheDir := filepath.Join(t.TempDir(), "cache-dir-is-a-file")
	if err := os.WriteFile(blockedCacheDir, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	sharedClient = client
	sharedAuditLog = auditLog
	sharedCache = lib.NewCache(blockedCacheDir, time.Minute)
	envName, selectedEnvironments = "testenv", nil
	confirmDelete, dryRunDelete = true, false
	deleteCmd.SetContext(context.Background())
	var logOutput bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelWarn})))

	if err := deleteCmd.RunE(deleteCmd, nil); err != nil {
		t.Fatalf("accepted delete returned error after cache persistence failure: %v", err)
	}
	if !accepted {
		t.Fatal("delete request was not accepted by the server")
	}
	if !strings.Contains(logOutput.String(), "cache invalidation failed") {
		t.Fatalf("log output = %q, want cache invalidation warning", logOutput.String())
	}
}

func TestDeleteInvalidatesCacheBeforeAuditFailure(t *testing.T) {
	previousEnv, previousTargets := envName, selectedEnvironments
	previousCache, previousClient, previousAudit := sharedCache, sharedClient, sharedAuditLog
	previousConfirm, previousDryRun := confirmDelete, dryRunDelete
	previousContext := deleteCmd.Context()
	t.Cleanup(func() {
		envName, selectedEnvironments = previousEnv, previousTargets
		sharedCache, sharedClient, sharedAuditLog = previousCache, previousClient, previousAudit
		confirmDelete, dryRunDelete = previousConfirm, previousDryRun
		deleteCmd.SetContext(previousContext)
	})

	accepted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accepted = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := lib.NewClient(server.URL, true, "")
	if err != nil {
		t.Fatal(err)
	}
	blockedAuditParent := filepath.Join(t.TempDir(), "audit-parent-is-a-file")
	if err := os.WriteFile(blockedAuditParent, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	auditLog, err := lib.NewAuditLog(filepath.Join(blockedAuditParent, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}

	sharedClient = client
	sharedAuditLog = auditLog
	sharedCache = lib.NewCache(t.TempDir(), time.Minute)
	sharedCache.SetInfo("testenv", &lib.ClusterInfo{Environment: "testenv"})
	sharedCache.SetInfo("otherenv", &lib.ClusterInfo{Environment: "otherenv"})
	envName, selectedEnvironments = "testenv", nil
	confirmDelete, dryRunDelete = true, false
	deleteCmd.SetContext(context.Background())

	err = deleteCmd.RunE(deleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "succeeded, but recording audit log") {
		t.Fatalf("delete error = %v, want accepted mutation with audit recording error", err)
	}
	if !accepted {
		t.Fatal("delete request was not accepted by the server")
	}
	if _, ok := sharedCache.GetInfo("testenv"); ok {
		t.Fatal("accepted delete retained the cached info after audit failure")
	}
	if _, ok := sharedCache.GetInfo("otherenv"); !ok {
		t.Fatal("audit failure invalidated an unrelated cache entry")
	}
}
