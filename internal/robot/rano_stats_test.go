package robot

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/integrations/rano"
	"github.com/Dicklesworthstone/ntm/internal/tools"
)

func TestNormalizeRanoWindowDefault(t *testing.T) {
	got, err := normalizeRanoWindow("")
	if err != nil {
		t.Fatalf("normalizeRanoWindow returned error: %v", err)
	}
	if got != "5m" {
		t.Fatalf("expected default window 5m, got %s", got)
	}
}

func TestNormalizeRanoWindowInvalid(t *testing.T) {
	if _, err := normalizeRanoWindow("5x"); err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
}

func TestAggregateRanoStats(t *testing.T) {
	stats := []tools.RanoProcessStats{
		{PID: 100, RequestCount: 2, BytesIn: 10, BytesOut: 5, LastRequest: "2026-01-01T00:00:01Z"},
		{PID: 101, RequestCount: 3, BytesIn: 20, BytesOut: 8, LastRequest: "2026-01-01T00:00:02Z"},
		{PID: 200, RequestCount: 1, BytesIn: 5, BytesOut: 2, LastRequest: "2026-01-01T00:00:03Z"},
	}

	pidLookup := func(pid int) *rano.PaneIdentity {
		switch pid {
		case 100, 101:
			return &rano.PaneIdentity{
				Session:   "s1",
				PaneIndex: 1,
				PaneTitle: "s1__cc_1",
				NTMIndex:  1,
			}
		case 200:
			return &rano.PaneIdentity{
				Session:   "s1",
				PaneIndex: 2,
				PaneTitle: "s1__cc_2",
				NTMIndex:  2,
			}
		default:
			return nil
		}
	}

	allowPane := func(identity *rano.PaneIdentity) bool {
		return identity != nil && identity.PaneTitle == "s1__cc_1"
	}

	panes, total := aggregateRanoStats(stats, pidLookup, allowPane)

	pane, ok := panes["s1__cc_1"]
	if !ok {
		t.Fatalf("expected pane s1__cc_1 to be present")
	}
	if pane.RequestCount != 5 {
		t.Fatalf("expected request count 5, got %d", pane.RequestCount)
	}
	if pane.BytesIn != 30 {
		t.Fatalf("expected bytes_in 30, got %d", pane.BytesIn)
	}
	if pane.BytesOut != 13 {
		t.Fatalf("expected bytes_out 13, got %d", pane.BytesOut)
	}
	if pane.LastRequest != "2026-01-01T00:00:02Z" {
		t.Fatalf("expected last_request 2026-01-01T00:00:02Z, got %s", pane.LastRequest)
	}
	if len(pane.PIDs) != 2 {
		t.Fatalf("expected 2 pids, got %d", len(pane.PIDs))
	}

	if total.RequestCount != 5 {
		t.Fatalf("expected total request count 5, got %d", total.RequestCount)
	}
	if total.BytesIn != 30 {
		t.Fatalf("expected total bytes_in 30, got %d", total.BytesIn)
	}
	if total.BytesOut != 13 {
		t.Fatalf("expected total bytes_out 13, got %d", total.BytesOut)
	}
}
