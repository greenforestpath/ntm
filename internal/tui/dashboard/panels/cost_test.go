package panels

import "testing"

func TestNewCostPanel(t *testing.T) {
	panel := NewCostPanel()
	if panel == nil {
		t.Fatal("NewCostPanel returned nil")
	}

	cfg := panel.Config()
	if cfg.ID != "cost" {
		t.Errorf("Expected ID 'cost', got %q", cfg.ID)
	}
	if cfg.Title != "Cost Tracking" {
		t.Errorf("Expected Title 'Cost Tracking', got %q", cfg.Title)
	}
}

func TestCostPanel_SetSize(t *testing.T) {
	panel := NewCostPanel()
	panel.SetSize(80, 24)

	if panel.Width() != 80 {
		t.Errorf("Expected Width 80, got %d", panel.Width())
	}
	if panel.Height() != 24 {
		t.Errorf("Expected Height 24, got %d", panel.Height())
	}
}

func TestCostPanel_FocusBlur(t *testing.T) {
	panel := NewCostPanel()
	if panel.IsFocused() {
		t.Error("Panel should not be focused initially")
	}

	panel.Focus()
	if !panel.IsFocused() {
		t.Error("Panel should be focused after Focus()")
	}

	panel.Blur()
	if panel.IsFocused() {
		t.Error("Panel should not be focused after Blur()")
	}
}

func TestCostPanel_SetData_Sorts(t *testing.T) {
	panel := NewCostPanel()
	panel.SetSize(60, 12)

	panel.SetData(CostPanelData{
		Agents: []CostAgentRow{
			{PaneTitle: "proj__cc_2", InputTokens: 1000, OutputTokens: 1000, CostUSD: 1.0, Trend: CostTrendUp},
			{PaneTitle: "proj__cc_1", InputTokens: 1000, OutputTokens: 1000, CostUSD: 2.0, Trend: CostTrendUp},
		},
		SessionTotalUSD: 3.0,
		LastHourUSD:     1.2,
		DailyBudgetUSD:  10,
		BudgetUsedUSD:   3.0,
	}, nil)

	if panel.data.Agents[0].PaneTitle != "proj__cc_1" {
		t.Fatalf("expected highest cost agent first, got %q", panel.data.Agents[0].PaneTitle)
	}

	view := panel.View()
	if view == "" {
		t.Fatal("expected non-empty View output")
	}
}

func TestCostPanel_HasData(t *testing.T) {
	panel := NewCostPanel()
	if panel.HasData() {
		t.Fatal("expected HasData=false initially")
	}

	panel.SetData(CostPanelData{DailyBudgetUSD: 50, BudgetUsedUSD: 1}, nil)
	if !panel.HasData() {
		t.Fatal("expected HasData=true when budget is set")
	}
}
