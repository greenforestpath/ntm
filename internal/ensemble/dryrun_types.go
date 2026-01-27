package ensemble

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// DryRunPlan represents the computed spawn plan without side effects.
type DryRunPlan struct {
	GeneratedAt time.Time        `json:"generated_at"`
	SessionName string           `json:"session_name"`
	Question    string           `json:"question"`
	PresetUsed  string           `json:"preset_used,omitempty"`
	Modes       []DryRunMode     `json:"modes"`
	Assignments []DryRunAssign   `json:"assignments"`
	Budget      DryRunBudget     `json:"budget"`
	Synthesis   DryRunSynthesis  `json:"synthesis"`
	Validation  DryRunValidation `json:"validation"`
	Preambles   []DryRunPreamble `json:"preambles,omitempty"`
}

// DryRunMode describes a resolved mode in the dry-run plan.
type DryRunMode struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Tier      string `json:"tier"`
	ShortDesc string `json:"short_desc,omitempty"`
}

// DryRunAssign describes a planned mode-to-agent assignment.
type DryRunAssign struct {
	ModeID      string `json:"mode_id"`
	ModeCode    string `json:"mode_code"`
	AgentType   string `json:"agent_type"`
	PaneIndex   int    `json:"pane_index"`
	TokenBudget int    `json:"token_budget"`
}

// DryRunBudget summarizes the token budget for the ensemble.
type DryRunBudget struct {
	MaxTokensPerMode       int `json:"max_tokens_per_mode"`
	MaxTotalTokens         int `json:"max_total_tokens"`
	SynthesisReserveTokens int `json:"synthesis_reserve_tokens"`
	ContextReserveTokens   int `json:"context_reserve_tokens"`
	EstimatedTotalTokens   int `json:"estimated_total_tokens"`
	ModeCount              int `json:"mode_count"`
}

// DryRunSynthesis summarizes the synthesis configuration.
type DryRunSynthesis struct {
	Strategy           string  `json:"strategy"`
	SynthesizerModeID  string  `json:"synthesizer_mode_id,omitempty"`
	MinConfidence      float64 `json:"min_confidence,omitempty"`
	MaxFindings        int     `json:"max_findings,omitempty"`
	ConflictResolution string  `json:"conflict_resolution,omitempty"`
}

// DryRunValidation reports validation status and any issues.
type DryRunValidation struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

// DryRunPreamble contains optional preamble preview for a mode.
type DryRunPreamble struct {
	ModeID   string `json:"mode_id"`
	ModeCode string `json:"mode_code"`
	Preview  string `json:"preview"`
	Length   int    `json:"length"`
}

// DryRunOptions configures the dry-run behavior.
type DryRunOptions struct {
	// IncludePreambles enables preamble preview generation.
	IncludePreambles bool
	// PreamblePreviewLength is the max chars to include (0 = full).
	PreamblePreviewLength int
}

// Validate returns an error if the dry-run plan has validation errors.
func (p *DryRunPlan) Validate() error {
	if p == nil {
		return errors.New("plan is nil")
	}
	if !p.Validation.Valid {
		return fmt.Errorf("validation failed: %s", strings.Join(p.Validation.Errors, "; "))
	}
	return nil
}

// ModeCount returns the number of modes in the plan.
func (p *DryRunPlan) ModeCount() int {
	if p == nil {
		return 0
	}
	return len(p.Modes)
}

// EstimatedTokens returns the total estimated token usage.
func (p *DryRunPlan) EstimatedTokens() int {
	if p == nil {
		return 0
	}
	return p.Budget.EstimatedTotalTokens
}
