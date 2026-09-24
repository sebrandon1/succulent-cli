package cmd

import (
	"encoding/json"
	"errors"
	"io"
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

func TestLoadEnvironmentGroupsRejectsMalformedAndEmptyGroups(t *testing.T) {
	for _, tc := range []struct {
		name     string
		contents string
		wantErr  string
	}{
		{name: "malformed YAML", contents: "groups: [", wantErr: "parsing environment groups"},
		{name: "invalid group name", contents: "groups:\n  invalid/group:\n    - env-one\n", wantErr: "invalid environment group name"},
		{name: "empty group", contents: "groups:\n  staging: []\n", wantErr: "has no members"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			writeGroupsFile(t, tc.contents)
			if _, err := loadEnvironmentGroups(); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("loadEnvironmentGroups() error = %v, want %q", err, tc.wantErr)
			}
		})
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
	if got, err := environmentDestination("", "env-one", targets); err != nil || got != "" {
		t.Fatalf("empty destination = %q, %v; want empty destination without error", got, err)
	}
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
	if err := validateBatchDestination("./kubeconfigs/{env}.yaml", targets); err != nil {
		t.Fatalf("validateBatchDestination() error = %v", err)
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
	for _, cmd := range []*cobra.Command{
		deleteCmd, reprovisionCmd, snoProvisionCmd, snoKubeconfigCmd,
		ztpProvisionCmd, ztpKubeconfigCmd, hsProvisionCmd, hsKubeconfigCmd, fetchKubeconfigCmd,
	} {
		if !supportsEnvironmentGroups(cmd) {
			t.Errorf("supportsEnvironmentGroups(%q) = false", cmd.CommandPath())
		}
	}
	if supportsEnvironmentGroups(&cobra.Command{Use: "other"}) {
		t.Fatal("supportsEnvironmentGroups() accepted an unsupported command")
	}
}

func TestCurrentEnvironmentTargets(t *testing.T) {
	previousTargets, previousEnv := selectedEnvironments, envName
	t.Cleanup(func() {
		selectedEnvironments, envName = previousTargets, previousEnv
	})

	selectedEnvironments = []string{"env-one", "env-two"}
	envName = "fallback"
	if got := strings.Join(currentEnvironmentTargets(), ","); got != "env-one,env-two" {
		t.Fatalf("selected targets = %q", got)
	}

	selectedEnvironments = nil
	if got := strings.Join(currentEnvironmentTargets(), ","); got != "fallback" {
		t.Fatalf("fallback targets = %q", got)
	}

	envName = ""
	if got := currentEnvironmentTargets(); got != nil {
		t.Fatalf("empty targets = %v, want nil", got)
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

func TestResolveEnvironmentTargetsRejectsInvalidDirectEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := resolveEnvironmentTargets("invalid/env", nil); err == nil || !strings.Contains(err.Error(), "invalid environment name") {
		t.Fatalf("resolveEnvironmentTargets() error = %v, want invalid environment name error", err)
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

func TestRunForEnvironmentTargetsSingleAndDryRun(t *testing.T) {
	previousTargets, previousEnv := selectedEnvironments, envName
	t.Cleanup(func() {
		selectedEnvironments, envName = previousTargets, previousEnv
	})
	selectedEnvironments = []string{"env-one"}
	envName = ""

	called := false
	err := runForEnvironmentTargets(func(target string) (string, error) {
		if target != "env-one" {
			t.Fatalf("action target = %q", target)
		}
		return "completed", nil
	}, func(target, message string) error {
		called = target == "env-one" && message == "completed"
		return nil
	}, false)
	if err != nil || !called {
		t.Fatalf("single-target run: error = %v, result callback called = %t", err, called)
	}

	called = false
	err = runForEnvironmentTargets(func(string) (string, error) { return "preview", nil }, func(string, string) error {
		called = true
		return nil
	}, true)
	if err != nil || called {
		t.Fatalf("single-target dry run: error = %v, result callback called = %t", err, called)
	}
}

func TestRunForEnvironmentTargetsSingleActionError(t *testing.T) {
	previousTargets, previousEnv := selectedEnvironments, envName
	t.Cleanup(func() {
		selectedEnvironments, envName = previousTargets, previousEnv
	})
	selectedEnvironments = []string{"env-one"}
	envName = ""
	wantErr := errors.New("operation failed")
	callbackCalled := false

	err := runForEnvironmentTargets(func(string) (string, error) { return "", wantErr }, func(string, string) error {
		callbackCalled = true
		return nil
	}, false)
	if !errors.Is(err, wantErr) || callbackCalled {
		t.Fatalf("single-target error = %v, result callback called = %t", err, callbackCalled)
	}
}

func TestRunForEnvironmentTargetsBatchJSON(t *testing.T) {
	previousTargets, previousEnv, previousFormat := selectedEnvironments, envName, outputFormat
	t.Cleanup(func() {
		selectedEnvironments, envName, outputFormat = previousTargets, previousEnv, previousFormat
	})
	selectedEnvironments = []string{"env-one", "env-two"}
	envName = ""
	outputFormat = "json"

	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = stdoutWrite, stderrWrite
	defer func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
		_ = stdoutRead.Close()
		_ = stderrRead.Close()
	}()

	called := make([]string, 0, 2)
	runErr := runForEnvironmentTargets(func(target string) (string, error) {
		called = append(called, target)
		return "completed", nil
	}, nil, false)
	_ = stdoutWrite.Close()
	_ = stderrWrite.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	output, err := io.ReadAll(stdoutRead)
	if err != nil {
		t.Fatal(err)
	}

	var outcomes []environmentOutcome
	if err := json.Unmarshal(output, &outcomes); err != nil {
		t.Fatalf("batch JSON output %q is invalid: %v", output, err)
	}
	if runErr != nil || strings.Join(called, ",") != "env-one,env-two" || len(outcomes) != 2 {
		t.Fatalf("batch run error = %v, calls = %v, outcomes = %v", runErr, called, outcomes)
	}
	for _, outcome := range outcomes {
		if outcome.Status != "success" || outcome.Message != "completed" {
			t.Errorf("unexpected batch outcome: %+v", outcome)
		}
	}
}
