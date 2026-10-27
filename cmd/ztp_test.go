package cmd

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateZTPIPVersions(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		ipv6     bool
		hubIPv   string
		spokeIPv string
		wantErr  string
	}{
		{name: "defaults without ipv6", hubIPv: "dual", spokeIPv: "ipv6"},
		{name: "ipv6 with dual-stack spoke", args: []string{"--spoke-ipv", "dual"}, ipv6: true, hubIPv: "dual", spokeIPv: "dual"},
		{name: "ipv6-only hub", args: []string{"--hub-ipv", "ipv6"}, ipv6: true, hubIPv: "ipv6", spokeIPv: "ipv6"},
		{name: "invalid spoke value", ipv6: true, hubIPv: "dual", spokeIPv: "ipv4", wantErr: "--spoke-ipv must be dual or ipv6"},
		{name: "invalid hub value", ipv6: true, hubIPv: "v6", spokeIPv: "ipv6", wantErr: "--hub-ipv must be dual or ipv6"},
		{name: "spoke-ipv without ipv6", args: []string{"--spoke-ipv", "dual"}, hubIPv: "dual", spokeIPv: "dual", wantErr: "--spoke-ipv requires --ipv6"},
		{name: "hub-ipv without ipv6", args: []string{"--hub-ipv", "ipv6"}, hubIPv: "ipv6", spokeIPv: "ipv6", wantErr: "--hub-ipv requires --ipv6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("hub-ipv", "dual", "")
			cmd.Flags().String("spoke-ipv", "ipv6", "")
			if err := cmd.Flags().Parse(tt.args); err != nil {
				t.Fatalf("parsing flags: %v", err)
			}

			err := validateZTPIPVersions(cmd, tt.ipv6, tt.hubIPv, tt.spokeIPv)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

// TestZTPProvisionCommandIPv6 checks that the IPv6 flags reach the submitted form.
func TestZTPProvisionCommandIPv6(t *testing.T) {
	var form url.Values
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
		}
		form = r.PostForm
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()
	t.Cleanup(func() {
		ztpIPv6, ztpHubIPv, ztpSpokeIPv = false, ipVersionDual, ipVersionIPv6
		for _, name := range []string{"ipv6", "hub-ipv", "spoke-ipv"} {
			ztpProvisionCmd.Flags().Lookup(name).Changed = false
		}
	})

	rootCmd.SetArgs([]string{"ztp", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com",
		"--sno-tag", "4.17", "--spoke-tag", "4.17", "--type", "mno", "--ipv6", "--spoke-ipv", "dual", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	for field, want := range map[string]string{"enable_ipv6": "on", "hub_ipv": "dual", "spoke_ipv": "dual", "plan": "testenv"} {
		if got := form.Get(field); got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}
}
