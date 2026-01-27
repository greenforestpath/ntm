package pipeline

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/robot"
)

// Mock objects for testing
type mockRouter struct{}

func (r *mockRouter) Route(agents []robot.ScoredAgent, strategy robot.RoutingStrategy, ctx robot.RoutingContext) robot.RoutingResult {
	if len(agents) == 0 {
		return robot.RoutingResult{Reason: "no agents"}
	}
	// Just pick the first one
	return robot.RoutingResult{
		Selected: &agents[0],
		Reason:   "mock",
	}
}

type mockScorer struct {
	agents []robot.ScoredAgent
}

func (s *mockScorer) ScoreAgents(session string, prompt string) ([]robot.ScoredAgent, error) {
	return s.agents, nil
}

func TestParallelPaneSelectionRace(t *testing.T) {
	t.Skip("TODO: requires dependency injection for deterministic pane selection")
}
