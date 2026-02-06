package panels

import (
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/status"
)

func TestRanoNetworkPanelViewDisabled(t *testing.T) {
	panel := NewRanoNetworkPanel()
	panel.SetSize(60, 12)
	panel.SetData(RanoNetworkPanelData{
		Loaded:  true,
		Enabled: false,
	})

	out := status.StripANSI(panel.View())
	if !strings.Contains(out, "rano disabled") {
		t.Fatalf("expected disabled state, got:\n%s", out)
	}
}

func TestRanoNetworkPanelViewWithRowsExpanded(t *testing.T) {
	panel := NewRanoNetworkPanel()
	panel.SetSize(80, 16)
	panel.SetData(RanoNetworkPanelData{
		Loaded:       true,
		Enabled:      true,
		Available:    true,
		Version:      "0.1.0",
		PollInterval: 1 * time.Second,
		Rows: []RanoNetworkRow{
			{
				Label:        "proj__cc_1",
				AgentType:    "cc",
				RequestCount: 3,
				BytesOut:     45 * 1024,
				BytesIn:      120 * 1024,
				LastRequest:  time.Now().Add(-100 * time.Millisecond),
			},
			{
				Label:        "proj__cod_1",
				AgentType:    "cod",
				RequestCount: 1,
				BytesOut:     10 * 1024,
				BytesIn:      50 * 1024,
				LastRequest:  time.Now().Add(-10 * time.Second),
			},
		},
		TotalRequests: 4,
		TotalBytesOut: 55 * 1024,
		TotalBytesIn:  170 * 1024,
	})

	out := status.StripANSI(panel.View())
	for _, want := range []string{
		"Network Activity",
		"proj__cc_1",
		"proj__cod_1",
		"Total:",
		"By provider:",
		"anthropic:",
		"openai:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}
