package agentmail

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadSessionAgentAmbiguity demonstrates that LoadSessionAgent is ambiguous
// when multiple projects share a session name and no projectKey is provided.
func TestLoadSessionAgentAmbiguity(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "agentmail-ambiguity-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the config dir
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	sessionName := "ambiguous-session"
	projectA := "/data/projects/project-a"
	projectB := "/data/projects/project-b"

	// Create agent info for Project A
	infoA := &SessionAgentInfo{
		AgentName:  "agent-a",
		ProjectKey: projectA,
	}
	if err := SaveSessionAgent(sessionName, projectA, infoA); err != nil {
		t.Fatalf("SaveSessionAgent(A) failed: %v", err)
	}

	// Create agent info for Project B
	infoB := &SessionAgentInfo{
		AgentName:  "agent-b",
		ProjectKey: projectB,
	}
	if err := SaveSessionAgent(sessionName, projectB, infoB); err != nil {
		t.Fatalf("SaveSessionAgent(B) failed: %v", err)
	}

	// Load without projectKey - this is the ambiguous case
	loaded, err := LoadSessionAgent(sessionName, "")
	if err != nil {
		t.Fatalf("LoadSessionAgent(empty) failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected loaded info to be non-nil")
	}

	// It should find one of them. With filepath.Walk/ReadDir, it's usually alphabetical
	// based on the slug.
	// Slug for /data/projects/project-a -> project_a
	// Slug for /data/projects/project-b -> project_b
	// So likely 'agent-a' is returned.
	t.Logf("Loaded agent: %s for project: %s", loaded.AgentName, loaded.ProjectKey)

	// Now try to load specifically for Project B using the correct key
	loadedB, err := LoadSessionAgent(sessionName, projectB)
	if err != nil {
		t.Fatalf("LoadSessionAgent(B) failed: %v", err)
	}
	if loadedB.AgentName != "agent-b" {
		t.Errorf("LoadSessionAgent(B) returned wrong agent: %s, want agent-b", loadedB.AgentName)
	}

	// This confirms that we MUST pass the projectKey to get the correct agent
	// if we know the project context. The bug is that callers (like UpdateSessionActivity)
	// pass "" even when they could know the project.
}

func TestUpdateSessionActivityTargeting(t *testing.T) {
	// Setup same environment as above
	tmpDir, err := os.MkdirTemp("", "agentmail-update-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	sessionName := "ambiguous-session"
	projectA := "/data/projects/project-a"
	projectB := "/data/projects/project-b"

	// Create agent info for Project A
	infoA := &SessionAgentInfo{AgentName: "agent-a", ProjectKey: projectA, LastActiveAt: time.Now().Add(-1 * time.Hour)}
	SaveSessionAgent(sessionName, projectA, infoA)

	// Create agent info for Project B
	infoB := &SessionAgentInfo{AgentName: "agent-b", ProjectKey: projectB, LastActiveAt: time.Now().Add(-1 * time.Hour)}
	SaveSessionAgent(sessionName, projectB, infoB)

	// Update activity for Project B explicitly
	client := NewClient()
	// We don't care about server availability for the local update part
	if err := client.UpdateSessionActivity(context.Background(), sessionName, projectB); err != nil {
		t.Fatalf("UpdateSessionActivity failed: %v", err)
	}

	// Verify Project B was updated
	loadedB, _ := LoadSessionAgent(sessionName, projectB)
	if time.Since(loadedB.LastActiveAt) > 1*time.Minute {
		t.Errorf("Project B was not updated! LastActiveAt: %v", loadedB.LastActiveAt)
	}

	// Verify Project A was NOT updated (should still be old)
	loadedA, _ := LoadSessionAgent(sessionName, projectA)
	if time.Since(loadedA.LastActiveAt) < 50*time.Minute {
		t.Errorf("Project A was incorrectly updated! LastActiveAt: %v", loadedA.LastActiveAt)
	}
}
