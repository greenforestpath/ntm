package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

type safetyCheckResponse struct {
	Command string `json:"command"`
	Action  string `json:"action"`
	Pattern string `json:"pattern,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type policyShowResponse struct {
	Success bool `json:"success"`

	Version    int    `json:"version"`
	PolicyPath string `json:"policy_path,omitempty"`
	IsDefault  bool   `json:"is_default"`

	Stats struct {
		Blocked  int `json:"blocked"`
		Approval int `json:"approval"`
		Allowed  int `json:"allowed"`
		SLBRules int `json:"slb_rules"`
	} `json:"stats"`
}

func writePolicyFile(t *testing.T, homeDir string, yaml string) {
	t.Helper()
	ntmDir := filepath.Join(homeDir, ".ntm")
	if err := os.MkdirAll(ntmDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", ntmDir, err)
	}
	path := filepath.Join(ntmDir, "policy.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write policy.yaml: %v", err)
	}
}

func runSafetyCheck(t *testing.T, dir string, args ...string) safetyCheckResponse {
	t.Helper()
	out := runCmd(t, dir, "ntm", append([]string{"--json", "safety", "check", "--"}, args...)...)
	var resp safetyCheckResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal safety check: %v\nout=%s", err, string(out))
	}
	return resp
}

func runSafetyCheckExpectExit1(t *testing.T, dir string, args ...string) safetyCheckResponse {
	t.Helper()
	out, err := runCmdAllowFail(t, dir, "ntm", append([]string{"--json", "safety", "check", "--"}, args...)...)
	if err == nil {
		t.Fatalf("expected ntm safety check to fail (exit=1); out=%s", string(out))
	}
	var resp safetyCheckResponse
	if unmarshalErr := json.Unmarshal(out, &resp); unmarshalErr != nil {
		t.Fatalf("unmarshal safety check: %v\nout=%s", unmarshalErr, string(out))
	}
	return resp
}

func runPolicyShow(t *testing.T, dir string) policyShowResponse {
	t.Helper()
	out := runCmd(t, dir, "ntm", "--json", "policy", "show")
	var resp policyShowResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal policy show: %v\nout=%s", err, string(out))
	}
	return resp
}

func TestE2EPolicyEnforcement_SafetyCheck(t *testing.T) {
	testutil.RequireE2E(t)
	testutil.RequireNTMBinary(t)

	logger := testutil.NewTestLoggerStdout(t)

	t.Run("default_policy_blocks_dangerous_command", func(t *testing.T) {
		homeDir := t.TempDir()
		workDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		logger.Log("[E2E-POLICY] case=default_policy_blocks cmd=%q", "git reset --hard")
		resp := runSafetyCheckExpectExit1(t, workDir, "git", "reset", "--hard")
		if resp.Action != "block" {
			t.Fatalf("expected action=block, got %q (pattern=%q reason=%q)", resp.Action, resp.Pattern, resp.Reason)
		}
	})

	t.Run("allowed_rules_override_blocked_rules", func(t *testing.T) {
		homeDir := t.TempDir()
		workDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		policyYAML := `version: 1
automation:
  auto_push: false
  auto_commit: true
  force_release: approval
allowed:
  - pattern: 'git\s+status'
    reason: "git status is always safe"
blocked:
  - pattern: 'git\s+.*'
    reason: "block all git commands unless explicitly allowed"
approval_required: []
`
		writePolicyFile(t, homeDir, policyYAML)

		logger.Log("[E2E-POLICY] case=allowed_over_blocked cmd=%q", "git status")
		resp := runSafetyCheck(t, workDir, "git", "status")
		if resp.Action != "allow" {
			t.Fatalf("expected action=allow, got %q (pattern=%q reason=%q)", resp.Action, resp.Pattern, resp.Reason)
		}
		if resp.Pattern != `git\s+status` {
			t.Fatalf("expected allow pattern %q, got %q", `git\s+status`, resp.Pattern)
		}

		show := runPolicyShow(t, workDir)
		if show.IsDefault {
			t.Fatalf("expected is_default=false when policy.yaml exists")
		}
		if show.Stats.Allowed != 1 || show.Stats.Blocked != 1 || show.Stats.Approval != 0 {
			t.Fatalf("unexpected stats (allowed=%d blocked=%d approval=%d)", show.Stats.Allowed, show.Stats.Blocked, show.Stats.Approval)
		}
	})

	t.Run("blocked_rules_override_approval_required_rules", func(t *testing.T) {
		homeDir := t.TempDir()
		workDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		policyYAML := `version: 1
automation:
  auto_push: false
  auto_commit: true
  force_release: approval
allowed: []
blocked:
  - pattern: 'git\s+status'
    reason: "block git status"
approval_required:
  - pattern: 'git\s+status'
    reason: "also marked approval_required, but blocked should win"
`
		writePolicyFile(t, homeDir, policyYAML)

		logger.Log("[E2E-POLICY] case=blocked_over_approval cmd=%q", "git status")
		resp := runSafetyCheckExpectExit1(t, workDir, "git", "status")
		if resp.Action != "block" {
			t.Fatalf("expected action=block, got %q (pattern=%q reason=%q)", resp.Action, resp.Pattern, resp.Reason)
		}
		if resp.Pattern != `git\s+status` {
			t.Fatalf("expected block pattern %q, got %q", `git\s+status`, resp.Pattern)
		}
	})

	t.Run("home_policy_overrides_project_policy", func(t *testing.T) {
		homeDir := t.TempDir()
		workDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		// Home policy explicitly allows git status.
		homePolicy := `version: 1
automation: { auto_push: false, auto_commit: true, force_release: approval }
allowed:
  - pattern: 'git\s+status'
blocked:
  - pattern: 'git\s+.*'
approval_required: []
`
		writePolicyFile(t, homeDir, homePolicy)

		// Project policy blocks git status, but should not be consulted when home policy exists.
		projectPolicyPath := filepath.Join(workDir, ".ntm", "policy.yaml")
		if err := os.MkdirAll(filepath.Dir(projectPolicyPath), 0755); err != nil {
			t.Fatalf("mkdir project .ntm: %v", err)
		}
		projectPolicy := `version: 1
automation: { auto_push: false, auto_commit: true, force_release: approval }
allowed: []
blocked:
  - pattern: 'git\s+status'
approval_required: []
`
		if err := os.WriteFile(projectPolicyPath, []byte(projectPolicy), 0644); err != nil {
			t.Fatalf("write project policy.yaml: %v", err)
		}

		logger.Log("[E2E-POLICY] case=home_overrides_project cmd=%q", "git status")
		resp := runSafetyCheck(t, workDir, "git", "status")
		if resp.Action != "allow" {
			t.Fatalf("expected action=allow, got %q (pattern=%q reason=%q)", resp.Action, resp.Pattern, resp.Reason)
		}
	})

	t.Run("approval_required_returns_approve_action_without_blocking", func(t *testing.T) {
		homeDir := t.TempDir()
		workDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		policyYAML := `version: 1
automation:
  auto_push: false
  auto_commit: true
  force_release: approval
allowed: []
blocked: []
approval_required:
  - pattern: 'git\s+commit\s+--amend'
    reason: "rewrites history"
`
		writePolicyFile(t, homeDir, policyYAML)

		logger.Log("[E2E-POLICY] case=approval_required cmd=%q", "git commit --amend")
		resp := runSafetyCheck(t, workDir, "git", "commit", "--amend")
		if resp.Action != "approve" {
			t.Fatalf("expected action=approve, got %q (pattern=%q reason=%q)", resp.Action, resp.Pattern, resp.Reason)
		}
		if resp.Pattern != `git\s+commit\s+--amend` {
			t.Fatalf("expected approve pattern %q, got %q", `git\s+commit\s+--amend`, resp.Pattern)
		}
	})
}
