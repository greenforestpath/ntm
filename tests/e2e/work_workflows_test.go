package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/bv"
	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

type workflowsListResult struct {
	Workflows []struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Source       string `json:"source"`
		Coordination string `json:"coordination"`
		AgentCount   int    `json:"agent_count"`
	} `json:"workflows"`
	Total int `json:"total"`
}

type workflowsShowResult struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Source       string `json:"source"`
	Coordination string `json:"coordination"`
	AgentCount   int    `json:"agent_count"`
}

func findRepoRootForE2E(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root (go.mod) from %s", dir)
		}
		dir = parent
	}
}

func TestE2EWorkAndWorkflows_JSON(t *testing.T) {
	testutil.RequireE2E(t)
	testutil.RequireNTMBinary(t)

	logger := testutil.NewTestLoggerStdout(t)
	repoRoot := findRepoRootForE2E(t)

	t.Run("work_triage_json", func(t *testing.T) {
		testutil.SkipIfBvUnavailable(t)

		logger.LogSection("ntm work triage --format=json")
		out := runCmd(t, repoRoot, "ntm", "work", "triage", "--format=json")

		var triage bv.TriageResponse
		if err := json.Unmarshal(out, &triage); err != nil {
			t.Fatalf("unmarshal triage: %v\nout=%s", err, string(out))
		}
		if triage.DataHash == "" {
			t.Fatalf("expected data_hash to be set")
		}
		if triage.GeneratedAt.IsZero() {
			t.Fatalf("expected generated_at to be set")
		}
		if triage.Triage.Meta.IssueCount <= 0 {
			t.Fatalf("expected issue_count > 0, got %d", triage.Triage.Meta.IssueCount)
		}
	})

	t.Run("work_next_json", func(t *testing.T) {
		testutil.SkipIfBvUnavailable(t)

		logger.LogSection("ntm --json work next")
		out := runCmd(t, repoRoot, "ntm", "--json", "work", "next")

		var rec bv.TriageRecommendation
		if err := json.Unmarshal(out, &rec); err != nil {
			t.Fatalf("unmarshal next: %v\nout=%s", err, string(out))
		}
		if rec.ID == "" || rec.Title == "" {
			t.Fatalf("expected recommendation fields to be set (id=%q title=%q)", rec.ID, rec.Title)
		}
	})

	t.Run("workflows_list_json", func(t *testing.T) {
		logger.LogSection("ntm --json workflows list")
		out := runCmd(t, repoRoot, "ntm", "--json", "workflows", "list")

		var resp workflowsListResult
		if err := json.Unmarshal(out, &resp); err != nil {
			t.Fatalf("unmarshal workflows list: %v\nout=%s", err, string(out))
		}
		if resp.Total <= 0 {
			t.Fatalf("expected total > 0, got %d", resp.Total)
		}
		if len(resp.Workflows) == 0 {
			t.Fatalf("expected workflows list to be non-empty")
		}
	})

	t.Run("workflows_show_json", func(t *testing.T) {
		logger.LogSection("ntm --json workflows show red-green")
		out := runCmd(t, repoRoot, "ntm", "--json", "workflows", "show", "red-green")

		var resp workflowsShowResult
		if err := json.Unmarshal(out, &resp); err != nil {
			t.Fatalf("unmarshal workflows show: %v\nout=%s", err, string(out))
		}
		if resp.Name != "red-green" {
			t.Fatalf("expected name=red-green, got %q", resp.Name)
		}
		if resp.AgentCount <= 0 {
			t.Fatalf("expected agent_count > 0, got %d", resp.AgentCount)
		}
		if resp.Coordination == "" {
			t.Fatalf("expected coordination to be set")
		}
	})
}
