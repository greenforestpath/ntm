package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/config"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

func TestSpawnSessionLogic_Idempotency(t *testing.T) {
	if !tmux.IsInstalled() {
		t.Skip("tmux not installed")
	}

	tmpDir, err := os.MkdirTemp("", "ntm-test-idempotency")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldCfg := cfg
	oldJsonOutput := jsonOutput
	defer func() {
		cfg = oldCfg
		jsonOutput = oldJsonOutput
	}()

	cfg = config.Default()
	cfg.ProjectsBase = tmpDir
	jsonOutput = true

	// Use 'cat' which blocks and echoes input
	cfg.Agents.Claude = "cat"

	sessionName := fmt.Sprintf("ntm-test-idem-%d", time.Now().UnixNano())
	defer func() {
		_ = tmux.KillSession(sessionName)
	}()

	agents := []FlatAgent{
		{Type: AgentTypeClaude, Index: 1, Model: "v1"},
	}

	opts := SpawnOptions{
		Session:  sessionName,
		Agents:   agents,
		CCCount:  1,
		UserPane: true,
	}

	// 1. Initial Spawn
	if err := spawnSessionLogic(opts); err != nil {
		t.Fatalf("Initial spawn failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	panes, _ := tmux.GetPanes(sessionName)
	var agentPane *tmux.Pane
	for _, p := range panes {
		if p.Type == tmux.AgentClaude {
			agentPane = &p
			break
		}
	}
	if agentPane == nil {
		t.Fatal("Agent pane not found")
	}

	// Capture initial output
	initialOutput, _ := tmux.CapturePaneOutput(agentPane.ID, 10)
	_ = initialOutput

	// 2. Second Spawn (Idempotency check)
	if err := spawnSessionLogic(opts); err != nil {
		t.Fatalf("Second spawn failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	finalOutput, _ := tmux.CapturePaneOutput(agentPane.ID, 10)
	
	// Check for multiple launches
	count := strings.Count(finalOutput, "&& cat")
	if count > 1 {
		t.Errorf("Expected 1 launch of 'cat', found %d. Idempotency failed.\nOutput:\n%s", count, finalOutput)
	}
}
