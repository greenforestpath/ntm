package integration

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/status"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("tmux"); err != nil {
		// tmux is required for these integration tests
		return
	}
	os.Exit(m.Run())
}

func TestStatusDetectsIdlePrompt(t *testing.T) {
	testutil.RequireTmux(t)
	logger := testutil.NewTestLogger(t, t.TempDir())

	_, paneID := createSessionWithTitle(t, logger, "user_1")

	if err := tmux.SendKeys(paneID, "printf \"idle-ready\\n$ \"", true); err != nil {
		t.Fatalf("failed to seed pane output: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	requirePaneActivity(t, paneID)

	detector := status.NewDetector()
	st, err := detector.Detect(paneID)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if st.State != status.StateIdle {
		t.Fatalf("expected idle, got %s", st.State)
	}
}

func TestStatusDetectsWorkingPane(t *testing.T) {
	testutil.RequireTmux(t)
	logger := testutil.NewTestLogger(t, t.TempDir())

	_, paneID := createSessionWithTitle(t, logger, "cod_1")

	if err := tmux.SendKeys(paneID, "for i in 1 2 3; do echo working-$i; sleep 1; done", true); err != nil {
		t.Fatalf("failed to start work loop: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	requirePaneActivity(t, paneID)

	detector := status.NewDetector()
	st, err := detector.Detect(paneID)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if st.State != status.StateWorking {
		t.Fatalf("expected working, got %s", st.State)
	}
}

func TestStatusDetectsErrors(t *testing.T) {
	testutil.RequireTmux(t)
	logger := testutil.NewTestLogger(t, t.TempDir())

	_, paneID := createSessionWithTitle(t, logger, "cc_1")

	if err := tmux.SendKeys(paneID, "echo \"HTTP 429 rate limit\"; printf \"$ \"", true); err != nil {
		t.Fatalf("failed to write error output: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	requirePaneActivity(t, paneID)

	detector := status.NewDetector()
	st, err := detector.Detect(paneID)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if st.State != status.StateError {
		t.Fatalf("expected error state, got %s", st.State)
	}
	if st.ErrorType != status.ErrorRateLimit {
		t.Fatalf("expected ErrorRateLimit, got %s", st.ErrorType)
	}
}

func TestStatusDetectsAgentTypes(t *testing.T) {
	testutil.RequireTmux(t)
	logger := testutil.NewTestLogger(t, t.TempDir())

	session, pane1 := createSessionWithTitle(t, logger, "cc_1")

	pane2, err := tmux.SplitWindow(session, t.TempDir())
	if err != nil {
		t.Fatalf("failed to split window for cod pane: %v", err)
	}
	if err := tmux.SetPaneTitle(pane2, fmt.Sprintf("%s__cod_1", session)); err != nil {
		t.Fatalf("failed to set cod pane title: %v", err)
	}

	pane3, err := tmux.SplitWindow(session, t.TempDir())
	if err != nil {
		t.Fatalf("failed to split window for gmi pane: %v", err)
	}
	if err := tmux.SetPaneTitle(pane3, fmt.Sprintf("%s__gmi_1", session)); err != nil {
		t.Fatalf("failed to set gmi pane title: %v", err)
	}

	_ = tmux.ApplyTiledLayout(session)

	requirePaneActivity(t, pane1)

	detector := status.NewDetector()
	statuses, err := detector.DetectAll(session)
	if err != nil {
		t.Fatalf("detect all failed: %v", err)
	}

	found := map[string]bool{}
	for _, st := range statuses {
		found[st.AgentType] = true
	}

	for _, agent := range []string{"cc", "cod", "gmi"} {
		if !found[agent] {
			t.Fatalf("expected to detect agent type %s", agent)
		}
	}

	// Ensure the original pane retained its type
	for _, st := range statuses {
		if st.PaneID == pane1 && st.AgentType != "cc" {
			t.Fatalf("expected pane %s to be cc, got %s", pane1, st.AgentType)
		}
	}
}

func TestStatusIgnoresANSISequences(t *testing.T) {
	testutil.RequireTmux(t)
	logger := testutil.NewTestLogger(t, t.TempDir())

	_, paneID := createSessionWithTitle(t, logger, "cc_1")

	if err := tmux.SendKeys(paneID, "printf \"\\033[31mHTTP 429 rate limit\\033[0m\\n$ \"", true); err != nil {
		t.Fatalf("failed to write colored error output: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	requirePaneActivity(t, paneID)

	detector := status.NewDetector()
	st, err := detector.Detect(paneID)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if st.State != status.StateError || st.ErrorType != status.ErrorRateLimit {
		t.Fatalf("expected colored output to be detected as rate limit error, got state=%s type=%s", st.State, st.ErrorType)
	}
}

func createSessionWithTitle(t *testing.T, logger *testutil.TestLogger, titleSuffix string) (string, string) {
	t.Helper()

	session := fmt.Sprintf("ntm_status_%d", time.Now().UnixNano())
	logger.Log("Creating tmux session %s", session)

	if err := tmux.CreateSession(session, t.TempDir()); err != nil {
		t.Skipf("tmux not available: %v", err)
	}

	t.Cleanup(func() {
		logger.Log("Killing tmux session %s", session)
		_ = tmux.KillSession(session)
	})

	panes, err := tmux.GetPanesWithActivity(session)
	if err != nil {
		t.Fatalf("failed to list panes: %v", err)
	}
	if len(panes) == 0 {
		t.Fatalf("session %s has no panes", session)
	}

	paneID := panes[0].Pane.ID
	title := fmt.Sprintf("%s__%s", session, titleSuffix)
	if err := tmux.SetPaneTitle(paneID, title); err != nil {
		t.Fatalf("failed to set pane title: %v", err)
	}

	return session, paneID
}

func requirePaneActivity(t *testing.T, paneID string) {
	t.Helper()

	if _, err := tmux.GetPaneActivity(paneID); err != nil {
		t.Skipf("tmux pane_last_activity unavailable: %v", err)
	}
}
