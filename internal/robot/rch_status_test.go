package robot

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/tools"
)

func TestRCHWorkerCounts(t *testing.T) {
	workers := []tools.RCHWorker{
		{Name: "a", Available: true, Healthy: true, Load: 10, Queue: 0},
		{Name: "b", Available: true, Healthy: true, Load: 90, Queue: 0},
		{Name: "c", Available: true, Healthy: false, Load: 10, Queue: 0},
		{Name: "d", Available: false, Healthy: true, Load: 10, Queue: 0},
	}

	if got := countRCHHealthyWorkers(workers); got != 2 {
		t.Fatalf("expected 2 healthy workers, got %d", got)
	}
	if got := countRCHBusyWorkers(workers); got != 1 {
		t.Fatalf("expected 1 busy worker, got %d", got)
	}
}
