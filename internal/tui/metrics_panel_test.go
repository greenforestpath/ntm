package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/ensemble"
)

func TestNewMetricsPanel(t *testing.T) {
	panel := NewMetricsPanel(80, 40)
	if panel == nil {
		t.Fatal("NewMetricsPanel returned nil")
	}
	if panel.Width != 80 {
		t.Errorf("expected width 80, got %d", panel.Width)
	}
	if panel.Height != 40 {
		t.Errorf("expected height 40, got %d", panel.Height)
	}
}

func TestMetricsPanel_SetMetrics(t *testing.T) {
	panel := NewMetricsPanel(80, 40)

	metrics := ensemble.NewObservabilityMetrics(nil)
	outputs := []ensemble.ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []ensemble.Finding{{Finding: "F1"}}},
	}
	_ = metrics.ComputeFromOutputs(outputs, nil, nil)

	panel.SetMetrics(metrics)

	if panel.metrics == nil {
		t.Error("metrics should be set")
	}
	if panel.report == nil {
		t.Error("report should be computed from metrics")
	}
	if panel.err != nil {
		t.Error("err should be nil after SetMetrics")
	}
}

func TestMetricsPanel_SetError(t *testing.T) {
	panel := NewMetricsPanel(80, 40)

	panel.SetError(nil)
	if panel.err != nil {
		t.Error("err should be nil")
	}

	testErr := &testError{msg: "test error"}
	panel.SetError(testErr)
	if panel.err == nil {
		t.Error("err should be set")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestMetricsPanel_Init(t *testing.T) {
	panel := NewMetricsPanel(80, 40)
	cmd := panel.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestMetricsPanel_ViewEmpty(t *testing.T) {
	panel := NewMetricsPanel(80, 40)
	view := panel.View()

	if !strings.Contains(view, "Observability Metrics") {
		t.Error("view should contain title")
	}
	if !strings.Contains(view, "No metrics available") {
		t.Error("view should show empty state")
	}
}

func TestMetricsPanel_ViewWithMetrics(t *testing.T) {
	panel := NewMetricsPanel(80, 40)

	metrics := ensemble.NewObservabilityMetrics(nil)
	outputs := []ensemble.ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []ensemble.Finding{{Finding: "F1", Confidence: 0.8}}},
		{ModeID: "mode-b", Thesis: "B", TopFindings: []ensemble.Finding{{Finding: "F2", Confidence: 0.7}}},
	}
	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	panel.SetMetrics(metrics)

	view := panel.View()

	// Should contain metric sections
	if !strings.Contains(view, "Coverage") {
		t.Error("view should contain Coverage section")
	}
	if !strings.Contains(view, "Findings Velocity") {
		t.Error("view should contain Velocity section")
	}
	if !strings.Contains(view, "Redundancy") {
		t.Error("view should contain Redundancy section")
	}
	if !strings.Contains(view, "Conflicts") {
		t.Error("view should contain Conflicts section")
	}
}

func TestMetricsPanel_ViewWithError(t *testing.T) {
	panel := NewMetricsPanel(80, 40)
	panel.SetError(&testError{msg: "connection failed"})

	view := panel.View()

	if !strings.Contains(view, "Error") {
		t.Error("view should contain error indication")
	}
	if !strings.Contains(view, "connection failed") {
		t.Error("view should contain error message")
	}
}

func TestMetricsPanel_ViewZeroSize(t *testing.T) {
	panel := NewMetricsPanel(0, 0)
	view := panel.View()

	if view != "" {
		t.Error("view should be empty for zero size")
	}
}

func TestRenderCompactMetrics(t *testing.T) {
	// Test with nil report
	compact := RenderCompactMetrics(nil)
	if compact != "No metrics" {
		t.Errorf("expected 'No metrics', got %q", compact)
	}

	// Test with valid report
	metrics := ensemble.NewObservabilityMetrics(nil)
	outputs := []ensemble.ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []ensemble.Finding{{Finding: "F1"}}},
		{ModeID: "mode-b", Thesis: "B", TopFindings: []ensemble.Finding{{Finding: "F2"}}},
	}
	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	report := metrics.GetReport()

	compact = RenderCompactMetrics(report)

	// Should contain metric abbreviations
	if !strings.Contains(compact, "Cov:") {
		t.Error("compact view should contain coverage")
	}
	if !strings.Contains(compact, "Vel:") {
		t.Error("compact view should contain velocity")
	}
	if !strings.Contains(compact, "Red:") {
		t.Error("compact view should contain redundancy")
	}
}

func TestRenderMetricsBadges(t *testing.T) {
	// Test with nil report
	badges := RenderMetricsBadges(nil)
	if badges != nil {
		t.Error("badges should be nil for nil report")
	}

	// Test with valid report
	metrics := ensemble.NewObservabilityMetrics(nil)
	outputs := []ensemble.ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []ensemble.Finding{{Finding: "F1"}}},
	}
	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	report := metrics.GetReport()

	badges = RenderMetricsBadges(report)

	if len(badges) == 0 {
		t.Error("expected non-empty badges")
	}
}

func TestMetricsFromSession(t *testing.T) {
	// Test with nil session
	metrics := MetricsFromSession(nil, nil, nil, nil, nil)
	if metrics != nil {
		t.Error("expected nil metrics for nil session")
	}

	// Test with valid session
	session := &ensemble.EnsembleSession{
		SessionName: "test",
		Question:    "Test question",
	}
	outputs := []ensemble.ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []ensemble.Finding{{Finding: "F1"}}},
	}

	metrics = MetricsFromSession(session, outputs, nil, nil, nil)

	if metrics == nil {
		t.Error("expected non-nil metrics")
	}
}

func TestMetricsUpdateCmd(t *testing.T) {
	outputs := []ensemble.ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []ensemble.Finding{{Finding: "F1"}}},
	}

	cmd := MetricsUpdateCmd(outputs, nil, nil, nil)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Execute the command
	msg := cmd()

	panelMsg, ok := msg.(MetricsPanelMsg)
	if !ok {
		t.Fatalf("expected MetricsPanelMsg, got %T", msg)
	}

	if panelMsg.Err != nil {
		t.Errorf("unexpected error: %v", panelMsg.Err)
	}
	if panelMsg.Metrics == nil {
		t.Error("expected metrics in message")
	}
	if panelMsg.Report == nil {
		t.Error("expected report in message")
	}
}

func TestFormatMetricsTimestamp(t *testing.T) {
	// Test zero time
	var zeroTime time.Time
	result := FormatMetricsTimestamp(zeroTime)
	if result != "never" {
		t.Errorf("expected 'never' for zero time, got %q", result)
	}

	// Test non-zero time
	tm := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)
	result = FormatMetricsTimestamp(tm)
	if result != "14:30:45" {
		t.Errorf("expected '14:30:45', got %q", result)
	}
}

func TestRenderMetricsBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
		wantLen int
	}{
		{"zero percent", 0, 10, 10},
		{"50 percent", 0.5, 10, 10},
		{"100 percent", 1.0, 10, 10},
		{"over 100 percent", 1.5, 10, 10},
		{"negative percent", -0.5, 10, 10},
		{"zero width", 0.5, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use a simple color for testing
			result := renderMetricsBar(tc.percent, tc.width, "#00FF00", "#333333")

			// The result includes ANSI codes, so check logical length
			if tc.width == 0 && result != "" {
				t.Errorf("expected empty for zero width")
			}
		})
	}
}

func TestInterpretRedundancyScore(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{0.8, "high"},
		{0.6, "moderate"},
		{0.4, "acceptable"},
		{0.2, "good"},
		{0.0, "good"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := interpretRedundancyScore(tc.score)
			if result != tc.expected {
				t.Errorf("interpretRedundancyScore(%f) = %q, want %q", tc.score, result, tc.expected)
			}
		})
	}
}
