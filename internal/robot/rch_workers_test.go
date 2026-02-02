package robot

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/tools"
)

func TestRCHWorkerStatus(t *testing.T) {
	cases := []struct {
		name   string
		worker tools.RCHWorker
		want   string
	}{
		{"unavailable", tools.RCHWorker{Available: false, Healthy: true}, "unavailable"},
		{"unhealthy", tools.RCHWorker{Available: true, Healthy: false}, "unhealthy"},
		{"busy-queue", tools.RCHWorker{Available: true, Healthy: true, Queue: 1}, "busy"},
		{"busy-load", tools.RCHWorker{Available: true, Healthy: true, Load: 85}, "busy"},
		{"healthy", tools.RCHWorker{Available: true, Healthy: true, Load: 10, Queue: 0}, "healthy"},
	}

	for _, tc := range cases {
		if got := rchWorkerStatus(tc.worker); got != tc.want {
			t.Fatalf("%s: expected %s, got %s", tc.name, tc.want, got)
		}
	}
}
