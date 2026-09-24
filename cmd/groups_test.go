package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func writeGroupsFile(t *testing.T, contents string) {
	t.Helper()
	groupsPath := filepath.Join(configDir(), environmentGroupsFile)
	if err := os.MkdirAll(filepath.Dir(groupsPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(groupsPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEnvironmentGroups(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeGroupsFile(t, "groups:\n  staging:\n    - env-one\n    - env-two\n    - env-one\n")

	groups, err := loadEnvironmentGroups()
	if err != nil {
		t.Fatalf("loadEnvironmentGroups() error = %v", err)
	}
	if got, want := strings.Join(groups["staging"], ","), "env-one,env-two"; got != want {
		t.Fatalf("group members = %q, want %q", got, want)
	}
}

func TestLoadEnvironmentGroupsRejectsInvalidMember(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeGroupsFile(t, "groups:\n  staging:\n    - invalid/env\n")

	if _, err := loadEnvironmentGroups(); err == nil || !strings.Contains(err.Error(), "invalid environment") {
		t.Fatalf("loadEnvironmentGroups() error = %v, want invalid environment error", err)
	}
}

func TestResolveEnvironmentTargets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeGroupsFile(t, "groups:\n  staging:\n    - env-one\n    - env-two\n")

	got, err := resolveEnvironmentTargets("env-two,env-three", []string{"staging", "env-three"})
	if err != nil {
		t.Fatalf("resolveEnvironmentTargets() error = %v", err)
	}
	if want := "env-one,env-two,env-three"; strings.Join(got, ",") != want {
		t.Fatalf("resolved targets = %q, want %q", strings.Join(got, ","), want)
	}
}

func TestEnvironmentDestinationRequiresTemplateForBatch(t *testing.T) {
	targets := []string{"env-one", "env-two"}
	if _, err := environmentDestination("./kubeconfig.yaml", "env-one", targets); err == nil {
		t.Fatal("expected batch destination without {env} to fail")
	}

	got, err := environmentDestination("./kubeconfigs/{env}.yaml", "env-two", targets)
	if err != nil {
		t.Fatalf("environmentDestination() error = %v", err)
	}
	if got != "./kubeconfigs/env-two.yaml" {
		t.Fatalf("destination = %q", got)
	}
}

func TestConfigGroupsCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeGroupsFile(t, "groups:\n  staging:\n    - env-one\n    - env-two\n")

	for _, args := range [][]string{{"config", "groups", "list"}, {"config", "groups", "show", "staging"}} {
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute(%v) error = %v", args, err)
		}
	}
}

func TestDeleteEnvironmentGroupContinuesAfterFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeGroupsFile(t, "groups:\n  staging:\n    - env-one\n    - env-two\n")

	requested := make([]string, 0, 2)
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		requested = append(requested, r.Form.Get("plan"))
		if r.Form.Get("plan") == "env-two" {
			http.Error(w, "environment not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()
	defer func() { selectedEnvironments = nil }()

	rootCmd.SetArgs([]string{"delete", "staging", "--confirm"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 2 environment operations failed") {
		t.Fatalf("Execute() error = %v, want aggregate failure", err)
	}
	if got, want := strings.Join(requested, ","), "env-one,env-two"; got != want {
		t.Fatalf("delete requests = %q, want %q", got, want)
	}
}

func TestSupportsEnvironmentGroups(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"ztp", "provision"})
	if err != nil {
		t.Fatal(err)
	}
	if !supportsEnvironmentGroups(cmd) {
		t.Fatalf("supportsEnvironmentGroups(%q) = false", cmd.CommandPath())
	}
	if supportsEnvironmentGroups(&cobra.Command{Use: "other"}) {
		t.Fatal("supportsEnvironmentGroups() accepted an unsupported command")
	}
}

func TestLoadMissingEnvironmentGroups(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	groups, err := loadEnvironmentGroups()
	if err != nil {
		t.Fatalf("loadEnvironmentGroups() error = %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("missing groups file returned %d groups", len(groups))
	}
}

func TestResolveEnvironmentTargetsRejectsEmptyEntry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := resolveEnvironmentTargets("env-one,", nil); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("resolveEnvironmentTargets() error = %v, want empty target error", err)
	}
}

func TestResolveEnvironmentTargetsNoTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := resolveEnvironmentTargets("", nil)
	if err != nil {
		t.Fatalf("resolveEnvironmentTargets() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty target request resolved to %v", got)
	}
}

func TestRunForEnvironmentTargetsRejectsNoTargets(t *testing.T) {
	selectedEnvironments = nil
	previousEnv := envName
	envName = ""
	t.Cleanup(func() { envName = previousEnv })
	if err := runForEnvironmentTargets(func(string) (string, error) { return "", nil }, nil, false); err == nil {
		t.Fatal("expected no target error")
	}
}
