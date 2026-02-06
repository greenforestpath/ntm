package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

type batchSendOutput struct {
	Success    bool     `json:"success"`
	Session    string   `json:"session"`
	Randomized bool     `json:"randomized"`
	SeedUsed   int64    `json:"seed_used"`
	Order      []string `json:"order"`
}

// Deterministic shuffle mirrors internal/cli/send.go logic for E2E verification.
func expectedOrderSources(n int, seed int64) []string {
	perm := make([]int, n)
	for i := 0; i < n; i++ {
		perm[i] = i
	}
	if n <= 1 {
		return []string{"line:1"}
	}
	var x uint64 = uint64(seed)
	if x == 0 {
		x = 0x9e3779b97f4a7c15
	}
	next := func() uint64 {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		return x
	}
	for i := n - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		perm[i], perm[j] = perm[j], perm[i]
	}

	out := make([]string, 0, n)
	for _, idx := range perm {
		out = append(out, fmt.Sprintf("line:%d", idx+1))
	}
	return out
}

// bd-h3ha: E2E for --randomize + --seed in batch send mode (JSON output includes execution order).
func TestSendBatchRandomizeWithSeed(t *testing.T) {
	testutil.E2ETestPrecheckThrottled(t)

	logger := testutil.NewTestLogger(t, t.TempDir())

	// Isolate all history/state writes so this test never touches the user's real ~/.local/share or ~/.ntm.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sessionName := fmt.Sprintf("e2e_send_randomize_%d", time.Now().UnixNano())
	projectsBase := t.TempDir()

	configPath := filepath.Join(t.TempDir(), "config.toml")
	configContent := fmt.Sprintf(`
projects_base = %q

[agents]
claude = "bash"
`, projectsBase)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Cleanup(func() {
		_ = tmux.KillSession(sessionName)
	})

	logger.LogSection("spawn session")
	out, err := logger.Exec("ntm", "--config", configPath, "spawn", sessionName, "--cc=1", "--json")
	logger.Log("spawn output: %s (err=%v)", string(out), err)
	if err != nil {
		t.Fatalf("ntm spawn failed: %v", err)
	}

	// Give tmux time to create panes.
	time.Sleep(750 * time.Millisecond)
	testutil.AssertSessionExists(t, logger, sessionName)
	testutil.AssertPaneCountAtLeast(t, logger, sessionName, 2) // user pane + 1 agent

	// Create batch file.
	batchPath := filepath.Join(t.TempDir(), "batch.txt")
	batchContent := "echo RAND_A\necho RAND_B\necho RAND_C\necho RAND_D\necho RAND_E\n"
	if err := os.WriteFile(batchPath, []byte(batchContent), 0644); err != nil {
		t.Fatalf("failed to write batch file: %v", err)
	}

	seed := int64(123)
	logger.LogSection("send batch randomized (json)")
	out, err = logger.Exec(
		"ntm", "--config", configPath,
		"send", sessionName,
		"--batch", batchPath,
		"--agent", "1",
		"--randomize",
		"--seed", fmt.Sprintf("%d", seed),
		"--no-cass-check",
		"--json",
	)
	logger.Log("send output: %s (err=%v)", string(out), err)
	if err != nil {
		t.Fatalf("ntm send (batch randomized) failed: %v", err)
	}

	var got batchSendOutput
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("failed to parse send JSON: %v\nraw=%s", err, string(out))
	}
	if !got.Success {
		t.Fatalf("expected success=true, got false")
	}
	if !got.Randomized {
		t.Fatalf("expected randomized=true, got false")
	}
	if got.SeedUsed != seed {
		t.Fatalf("expected seed_used=%d, got %d", seed, got.SeedUsed)
	}
	if len(got.Order) != 5 {
		t.Fatalf("expected order length 5, got %d (%v)", len(got.Order), got.Order)
	}

	wantOrder := expectedOrderSources(5, seed)
	for i := range wantOrder {
		if got.Order[i] != wantOrder[i] {
			t.Fatalf("order[%d]=%q, want %q (got=%v)", i, got.Order[i], wantOrder[i], got.Order)
		}
	}
}
