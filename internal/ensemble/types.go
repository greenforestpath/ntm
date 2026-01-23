// Package ensemble provides types and utilities for multi-agent reasoning ensembles.
// An ensemble orchestrates multiple AI agents using different reasoning modes
// to analyze questions from multiple perspectives, then synthesizes their outputs.
package ensemble

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ModeCategory represents the taxonomy categories (A-L) for reasoning modes.
// Categories group related reasoning approaches to help users understand
// which modes address which types of problems.
type ModeCategory string

const (
	// CategoryFormal covers deductive, mathematical, and logical reasoning.
	CategoryFormal ModeCategory = "Formal"
	// CategoryAmpliative covers inductive and abductive reasoning.
	CategoryAmpliative ModeCategory = "Ampliative"
	// CategoryUncertainty covers probabilistic and statistical reasoning.
	CategoryUncertainty ModeCategory = "Uncertainty"
	// CategoryVagueness covers fuzzy and approximate reasoning.
	CategoryVagueness ModeCategory = "Vagueness"
	// CategoryChange covers temporal and belief revision reasoning.
	CategoryChange ModeCategory = "Change"
	// CategoryCausal covers causal inference and counterfactual reasoning.
	CategoryCausal ModeCategory = "Causal"
	// CategoryPractical covers decision-making and planning reasoning.
	CategoryPractical ModeCategory = "Practical"
	// CategoryStrategic covers game-theoretic and adversarial reasoning.
	CategoryStrategic ModeCategory = "Strategic"
	// CategoryDialectical covers argumentation and discourse reasoning.
	CategoryDialectical ModeCategory = "Dialectical"
	// CategoryModal covers necessity, possibility, and deontic reasoning.
	CategoryModal ModeCategory = "Modal"
	// CategoryDomain covers domain-specific (legal, scientific, ethical) reasoning.
	CategoryDomain ModeCategory = "Domain"
	// CategoryMeta covers reasoning about reasoning itself.
	CategoryMeta ModeCategory = "Meta"
)

// String returns the category as a string.
func (c ModeCategory) String() string {
	return string(c)
}

// IsValid returns true if this is a known category.
func (c ModeCategory) IsValid() bool {
	switch c {
	case CategoryFormal, CategoryAmpliative, CategoryUncertainty, CategoryVagueness,
		CategoryChange, CategoryCausal, CategoryPractical, CategoryStrategic,
		CategoryDialectical, CategoryModal, CategoryDomain, CategoryMeta:
		return true
	default:
		return false
	}
}

// AllCategories returns all valid mode categories.
func AllCategories() []ModeCategory {
	return []ModeCategory{
		CategoryFormal, CategoryAmpliative, CategoryUncertainty, CategoryVagueness,
		CategoryChange, CategoryCausal, CategoryPractical, CategoryStrategic,
		CategoryDialectical, CategoryModal, CategoryDomain, CategoryMeta,
	}
}

// ReasoningMode defines a named reasoning approach.
// Each mode represents a distinct way of analyzing a problem, with
// specific strengths, outputs, and failure modes.
type ReasoningMode struct {
	// ID is the unique identifier for this mode (e.g., "deductive", "bayesian").
	// Must be lowercase alphanumeric with optional hyphens.
	ID string `json:"id" toml:"id"`

	// Name is the human-readable name (e.g., "Deductive Logic").
	Name string `json:"name" toml:"name"`

	// Category is the taxonomy category this mode belongs to.
	Category ModeCategory `json:"category" toml:"category"`

	// ShortDesc is a one-line description for listings (max 80 chars).
	ShortDesc string `json:"short_desc" toml:"short_desc"`

	// Description is the full explanation of this reasoning approach.
	Description string `json:"description" toml:"description"`

	// Outputs describes what this mode produces (e.g., "Proof or counterexample").
	Outputs string `json:"outputs" toml:"outputs"`

	// BestFor lists problem types where this mode excels.
	BestFor []string `json:"best_for" toml:"best_for"`

	// FailureModes lists common pitfalls when using this mode.
	FailureModes []string `json:"failure_modes" toml:"failure_modes"`

	// Differentiator explains what makes this mode unique vs similar modes.
	Differentiator string `json:"differentiator" toml:"differentiator"`

	// Icon is a single emoji or Nerd Font glyph for UI display.
	Icon string `json:"icon" toml:"icon"`

	// Color is the hex color code for UI display (e.g., "#cba6f7").
	Color string `json:"color" toml:"color"`

	// PreambleKey is the key to lookup the preamble template for this mode.
	// The preamble is injected into the agent prompt to set its reasoning approach.
	PreambleKey string `json:"preamble_key" toml:"preamble_key"`
}

// Validate checks that the mode has all required fields and valid values.
func (m *ReasoningMode) Validate() error {
	if m.ID == "" {
		return errors.New("mode ID is required")
	}
	if err := ValidateModeID(m.ID); err != nil {
		return err
	}
	if m.Name == "" {
		return errors.New("mode name is required")
	}
	if !m.Category.IsValid() {
		return fmt.Errorf("invalid category %q", m.Category)
	}
	if m.ShortDesc == "" {
		return errors.New("mode short_desc is required")
	}
	if len(m.ShortDesc) > 80 {
		return fmt.Errorf("short_desc exceeds 80 characters (got %d)", len(m.ShortDesc))
	}
	return nil
}

// ModeAssignment maps a mode to an agent pane.
// This tracks which agent is using which reasoning mode in an ensemble session.
type ModeAssignment struct {
	// ModeID references the ReasoningMode.ID.
	ModeID string `json:"mode_id"`

	// PaneName is the tmux pane identifier (e.g., "myproject__cc_1").
	PaneName string `json:"pane_name"`

	// AgentType is the type of agent in this pane (cc, cod, gmi).
	AgentType string `json:"agent_type"`

	// Status tracks the assignment lifecycle.
	Status AssignmentStatus `json:"status"`

	// OutputPath is where captured output will be stored.
	OutputPath string `json:"output_path,omitempty"`

	// AssignedAt is when this assignment was created.
	AssignedAt time.Time `json:"assigned_at"`

	// CompletedAt is when the agent finished (status = done).
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error holds any error message if status = error.
	Error string `json:"error,omitempty"`
}

// AssignmentStatus tracks the lifecycle of a mode assignment.
type AssignmentStatus string

const (
	// AssignmentPending means the mode is queued but not yet sent to the agent.
	AssignmentPending AssignmentStatus = "pending"
	// AssignmentInjecting means the preamble is being sent to the agent.
	AssignmentInjecting AssignmentStatus = "injecting"
	// AssignmentActive means the agent is actively working with this mode.
	AssignmentActive AssignmentStatus = "active"
	// AssignmentDone means the agent has completed its analysis.
	AssignmentDone AssignmentStatus = "done"
	// AssignmentError means an error occurred during this assignment.
	AssignmentError AssignmentStatus = "error"
)

// String returns the status as a string.
func (s AssignmentStatus) String() string {
	return string(s)
}

// IsTerminal returns true if this is a final status (done or error).
func (s AssignmentStatus) IsTerminal() bool {
	return s == AssignmentDone || s == AssignmentError
}

// EnsembleStatus tracks the overall ensemble session state.
type EnsembleStatus string

const (
	// EnsembleSpawning means agents are being created.
	EnsembleSpawning EnsembleStatus = "spawning"
	// EnsembleInjecting means preambles are being sent to agents.
	EnsembleInjecting EnsembleStatus = "injecting"
	// EnsembleActive means agents are analyzing the question.
	EnsembleActive EnsembleStatus = "active"
	// EnsembleSynthesizing means outputs are being combined.
	EnsembleSynthesizing EnsembleStatus = "synthesizing"
	// EnsembleComplete means the ensemble run is finished.
	EnsembleComplete EnsembleStatus = "complete"
	// EnsembleError means an error occurred during the run.
	EnsembleError EnsembleStatus = "error"
)

// String returns the status as a string.
func (s EnsembleStatus) String() string {
	return string(s)
}

// IsTerminal returns true if this is a final status (complete or error).
func (s EnsembleStatus) IsTerminal() bool {
	return s == EnsembleComplete || s == EnsembleError
}

// EnsembleSession tracks a reasoning ensemble session.
// An ensemble session spawns multiple agents with different reasoning modes,
// captures their outputs, and synthesizes them into a combined analysis.
type EnsembleSession struct {
	// SessionName is the tmux session name hosting this ensemble.
	SessionName string `json:"session_name"`

	// Question is the user's question or problem being analyzed.
	Question string `json:"question"`

	// PresetUsed is the name of the preset used (if any).
	PresetUsed string `json:"preset_used,omitempty"`

	// Assignments maps modes to agent panes.
	Assignments []ModeAssignment `json:"assignments"`

	// Status is the overall ensemble state.
	Status EnsembleStatus `json:"status"`

	// SynthesisStrategy is how outputs will be combined.
	SynthesisStrategy SynthesisStrategy `json:"synthesis_strategy"`

	// CreatedAt is when the ensemble was started.
	CreatedAt time.Time `json:"created_at"`

	// SynthesizedAt is when synthesis completed.
	SynthesizedAt *time.Time `json:"synthesized_at,omitempty"`

	// SynthesisOutput is the final combined output.
	SynthesisOutput string `json:"synthesis_output,omitempty"`

	// Error holds the error message if status = error.
	Error string `json:"error,omitempty"`
}

// SynthesisStrategy defines how ensemble outputs are combined.
type SynthesisStrategy string

const (
	// StrategyConsensus combines outputs by finding agreement points.
	StrategyConsensus SynthesisStrategy = "consensus"
	// StrategyDebate synthesizes through dialectical comparison.
	StrategyDebate SynthesisStrategy = "debate"
	// StrategyWeighted uses quality-weighted combination.
	StrategyWeighted SynthesisStrategy = "weighted"
	// StrategySequential chains outputs in order.
	StrategySequential SynthesisStrategy = "sequential"
	// StrategyBestOf selects the highest-quality single response.
	StrategyBestOf SynthesisStrategy = "best-of"
)

// String returns the strategy as a string.
func (s SynthesisStrategy) String() string {
	return string(s)
}

// IsValid returns true if this is a known synthesis strategy.
func (s SynthesisStrategy) IsValid() bool {
	switch s {
	case StrategyConsensus, StrategyDebate, StrategyWeighted,
		StrategySequential, StrategyBestOf:
		return true
	default:
		return false
	}
}

// EnsemblePreset is a pre-configured mode combination.
// Presets make it easy to quickly start an ensemble with a curated
// set of modes for common use cases.
type EnsemblePreset struct {
	// Name is the unique identifier for this preset.
	Name string `json:"name" toml:"name"`

	// Description explains what this preset is for.
	Description string `json:"description" toml:"description"`

	// Modes lists the mode IDs to use in this preset.
	Modes []string `json:"modes" toml:"modes"`

	// SynthesisStrategy is how outputs should be combined.
	SynthesisStrategy SynthesisStrategy `json:"synthesis_strategy" toml:"synthesis_strategy"`

	// Tags are optional categories for organization.
	Tags []string `json:"tags,omitempty" toml:"tags"`
}

// Validate checks that the preset is valid and all mode IDs exist in the catalog.
func (p *EnsemblePreset) Validate(catalog []ReasoningMode) error {
	if p.Name == "" {
		return errors.New("preset name is required")
	}
	if len(p.Modes) == 0 {
		return errors.New("preset must have at least one mode")
	}
	if !p.SynthesisStrategy.IsValid() {
		return fmt.Errorf("invalid synthesis strategy %q", p.SynthesisStrategy)
	}

	// Build mode lookup
	modeIDs := make(map[string]bool)
	for _, m := range catalog {
		modeIDs[m.ID] = true
	}

	// Check all preset modes exist
	for _, modeID := range p.Modes {
		if !modeIDs[modeID] {
			return fmt.Errorf("mode %q not found in catalog", modeID)
		}
	}

	return nil
}

// modeIDRegex validates mode IDs (lowercase alphanumeric with hyphens).
var modeIDRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateModeID checks if a mode ID is valid.
// Valid IDs are lowercase, start with a letter, and contain only
// alphanumeric characters and hyphens.
func ValidateModeID(id string) error {
	if id == "" {
		return errors.New("mode ID cannot be empty")
	}
	if len(id) > 64 {
		return fmt.Errorf("mode ID exceeds 64 characters (got %d)", len(id))
	}
	if !modeIDRegex.MatchString(id) {
		return fmt.Errorf("invalid mode ID %q: must be lowercase, start with a letter, and contain only alphanumeric characters and hyphens", id)
	}
	return nil
}

// ValidatePreset checks if a preset is valid and all its modes exist.
// This is an alias for EnsemblePreset.Validate for convenience.
func ValidatePreset(preset EnsemblePreset, catalog []ReasoningMode) error {
	return preset.Validate(catalog)
}

// ModeCatalog holds a collection of reasoning modes.
// It provides lookup and filtering capabilities.
type ModeCatalog struct {
	modes   []ReasoningMode
	byID    map[string]*ReasoningMode
	byCat   map[ModeCategory][]*ReasoningMode
	version string
}

// NewModeCatalog creates a new catalog from a slice of modes.
func NewModeCatalog(modes []ReasoningMode, version string) (*ModeCatalog, error) {
	c := &ModeCatalog{
		modes:   make([]ReasoningMode, 0, len(modes)),
		byID:    make(map[string]*ReasoningMode),
		byCat:   make(map[ModeCategory][]*ReasoningMode),
		version: version,
	}

	for i := range modes {
		mode := modes[i]
		if err := mode.Validate(); err != nil {
			return nil, fmt.Errorf("invalid mode %q: %w", mode.ID, err)
		}
		if _, exists := c.byID[mode.ID]; exists {
			return nil, fmt.Errorf("duplicate mode ID %q", mode.ID)
		}
		c.modes = append(c.modes, mode)
		c.byID[mode.ID] = &c.modes[len(c.modes)-1]
		c.byCat[mode.Category] = append(c.byCat[mode.Category], &c.modes[len(c.modes)-1])
	}

	return c, nil
}

// GetMode returns a mode by ID, or nil if not found.
func (c *ModeCatalog) GetMode(id string) *ReasoningMode {
	return c.byID[id]
}

// ListModes returns all modes in the catalog.
func (c *ModeCatalog) ListModes() []ReasoningMode {
	result := make([]ReasoningMode, len(c.modes))
	copy(result, c.modes)
	return result
}

// ListByCategory returns all modes in a specific category.
func (c *ModeCatalog) ListByCategory(cat ModeCategory) []ReasoningMode {
	ptrs := c.byCat[cat]
	result := make([]ReasoningMode, len(ptrs))
	for i, p := range ptrs {
		result[i] = *p
	}
	return result
}

// SearchModes finds modes matching a search term in name, description, or best_for.
func (c *ModeCatalog) SearchModes(term string) []ReasoningMode {
	term = strings.ToLower(term)
	var result []ReasoningMode
	for _, m := range c.modes {
		if strings.Contains(strings.ToLower(m.Name), term) ||
			strings.Contains(strings.ToLower(m.Description), term) ||
			strings.Contains(strings.ToLower(m.ShortDesc), term) {
			result = append(result, m)
			continue
		}
		for _, bf := range m.BestFor {
			if strings.Contains(strings.ToLower(bf), term) {
				result = append(result, m)
				break
			}
		}
	}
	return result
}

// Version returns the catalog version string.
func (c *ModeCatalog) Version() string {
	return c.version
}

// Count returns the total number of modes.
func (c *ModeCatalog) Count() int {
	return len(c.modes)
}

// =============================================================================
// Output Schema Types
// =============================================================================
// These types define the mandatory structure for all mode outputs.
// Every mode must produce output conforming to ModeOutput to enable
// consistent synthesis and comparison across reasoning approaches.

// ImpactLevel categorizes the significance of findings and risks.
type ImpactLevel string

const (
	// ImpactHigh indicates a critical finding requiring immediate attention.
	ImpactHigh ImpactLevel = "high"
	// ImpactMedium indicates a significant finding worth addressing.
	ImpactMedium ImpactLevel = "medium"
	// ImpactLow indicates a minor finding for consideration.
	ImpactLow ImpactLevel = "low"
)

// String returns the impact level as a string.
func (i ImpactLevel) String() string {
	return string(i)
}

// IsValid returns true if this is a known impact level.
func (i ImpactLevel) IsValid() bool {
	switch i {
	case ImpactHigh, ImpactMedium, ImpactLow:
		return true
	default:
		return false
	}
}

// Confidence represents a confidence score between 0.0 and 1.0.
type Confidence float64

// Validate checks that the confidence is in the valid range [0.0, 1.0].
func (c Confidence) Validate() error {
	if c < 0.0 || c > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", c)
	}
	return nil
}

// String returns confidence as a percentage string.
func (c Confidence) String() string {
	return fmt.Sprintf("%.0f%%", c*100)
}

// Finding represents a specific discovery or insight from reasoning.
type Finding struct {
	// Finding is the description of what was discovered.
	Finding string `json:"finding"`

	// Impact is the significance level of this finding.
	Impact ImpactLevel `json:"impact"`

	// Confidence is how certain the mode is about this finding (0.0-1.0).
	Confidence Confidence `json:"confidence"`

	// EvidencePointer is a reference to supporting evidence (e.g., "file.go:42").
	EvidencePointer string `json:"evidence_pointer,omitempty"`

	// Reasoning explains how this finding was reached.
	Reasoning string `json:"reasoning,omitempty"`
}

// Validate checks that the finding is properly formed.
func (f *Finding) Validate() error {
	if f.Finding == "" {
		return errors.New("finding description is required")
	}
	if !f.Impact.IsValid() {
		return fmt.Errorf("invalid impact level %q", f.Impact)
	}
	if err := f.Confidence.Validate(); err != nil {
		return fmt.Errorf("invalid confidence: %w", err)
	}
	return nil
}

// Risk represents a potential problem or threat identified by reasoning.
type Risk struct {
	// Risk is the description of the potential problem.
	Risk string `json:"risk"`

	// Impact is the severity if this risk materializes.
	Impact ImpactLevel `json:"impact"`

	// Likelihood is the probability this risk will occur (0.0-1.0).
	Likelihood Confidence `json:"likelihood"`

	// Mitigation describes how to address this risk.
	Mitigation string `json:"mitigation,omitempty"`

	// AffectedAreas lists components or areas impacted by this risk.
	AffectedAreas []string `json:"affected_areas,omitempty"`
}

// Validate checks that the risk is properly formed.
func (r *Risk) Validate() error {
	if r.Risk == "" {
		return errors.New("risk description is required")
	}
	if !r.Impact.IsValid() {
		return fmt.Errorf("invalid impact level %q", r.Impact)
	}
	if err := r.Likelihood.Validate(); err != nil {
		return fmt.Errorf("invalid likelihood: %w", err)
	}
	return nil
}

// Recommendation represents a suggested action from reasoning.
type Recommendation struct {
	// Recommendation is the suggested action.
	Recommendation string `json:"recommendation"`

	// Priority indicates how urgent this recommendation is.
	Priority ImpactLevel `json:"priority"`

	// Rationale explains why this is recommended.
	Rationale string `json:"rationale,omitempty"`

	// Effort is an estimate of implementation complexity (low/medium/high).
	Effort string `json:"effort,omitempty"`

	// RelatedFindings lists finding indices that support this recommendation.
	RelatedFindings []int `json:"related_findings,omitempty"`
}

// Validate checks that the recommendation is properly formed.
func (r *Recommendation) Validate() error {
	if r.Recommendation == "" {
		return errors.New("recommendation text is required")
	}
	if !r.Priority.IsValid() {
		return fmt.Errorf("invalid priority %q", r.Priority)
	}
	return nil
}

// Question represents an unresolved question for the user.
type Question struct {
	// Question is the query for the user.
	Question string `json:"question"`

	// Context explains why this question matters.
	Context string `json:"context,omitempty"`

	// Blocking indicates if this question blocks further analysis.
	Blocking bool `json:"blocking,omitempty"`

	// SuggestedAnswers provides possible responses if applicable.
	SuggestedAnswers []string `json:"suggested_answers,omitempty"`
}

// Validate checks that the question is properly formed.
func (q *Question) Validate() error {
	if q.Question == "" {
		return errors.New("question text is required")
	}
	return nil
}

// FailureModeWarning represents a potential failure mode to watch for.
type FailureModeWarning struct {
	// Mode is the failure mode identifier.
	Mode string `json:"mode"`

	// Description explains what this failure mode entails.
	Description string `json:"description"`

	// Indicators are signs that this failure mode may be occurring.
	Indicators []string `json:"indicators,omitempty"`

	// Prevention describes how to avoid this failure mode.
	Prevention string `json:"prevention,omitempty"`
}

// Validate checks that the failure mode warning is properly formed.
func (f *FailureModeWarning) Validate() error {
	if f.Mode == "" {
		return errors.New("failure mode identifier is required")
	}
	if f.Description == "" {
		return errors.New("failure mode description is required")
	}
	return nil
}

// ModeOutput is the mandatory output schema for all reasoning modes.
// Every mode must produce output conforming to this structure to enable
// consistent synthesis and comparison across different reasoning approaches.
type ModeOutput struct {
	// ModeID identifies which reasoning mode produced this output.
	ModeID string `json:"mode_id"`

	// Thesis is the main conclusion or argument from this mode.
	Thesis string `json:"thesis"`

	// TopFindings are the key discoveries ranked by importance.
	TopFindings []Finding `json:"top_findings"`

	// Risks are potential problems or threats identified.
	Risks []Risk `json:"risks,omitempty"`

	// Recommendations are suggested actions.
	Recommendations []Recommendation `json:"recommendations,omitempty"`

	// QuestionsForUser are unresolved queries needing user input.
	QuestionsForUser []Question `json:"questions_for_user,omitempty"`

	// FailureModesToWatch are warnings about reasoning pitfalls.
	FailureModesToWatch []FailureModeWarning `json:"failure_modes_to_watch,omitempty"`

	// Confidence is the overall confidence in this analysis (0.0-1.0).
	Confidence Confidence `json:"confidence"`

	// RawOutput is the original unstructured output from the agent.
	RawOutput string `json:"raw_output,omitempty"`

	// GeneratedAt is when this output was produced.
	GeneratedAt time.Time `json:"generated_at"`
}

// Validate checks that the mode output is properly formed.
func (m *ModeOutput) Validate() error {
	if m.ModeID == "" {
		return errors.New("mode_id is required")
	}
	if m.Thesis == "" {
		return errors.New("thesis is required")
	}
	if len(m.TopFindings) == 0 {
		return errors.New("at least one finding is required")
	}
	if err := m.Confidence.Validate(); err != nil {
		return fmt.Errorf("invalid confidence: %w", err)
	}

	// Validate all findings
	for i, f := range m.TopFindings {
		if err := f.Validate(); err != nil {
			return fmt.Errorf("finding[%d]: %w", i, err)
		}
	}

	// Validate all risks
	for i, r := range m.Risks {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("risk[%d]: %w", i, err)
		}
	}

	// Validate all recommendations
	for i, r := range m.Recommendations {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("recommendation[%d]: %w", i, err)
		}
	}

	// Validate all questions
	for i, q := range m.QuestionsForUser {
		if err := q.Validate(); err != nil {
			return fmt.Errorf("question[%d]: %w", i, err)
		}
	}

	// Validate all failure mode warnings
	for i, f := range m.FailureModesToWatch {
		if err := f.Validate(); err != nil {
			return fmt.Errorf("failure_mode[%d]: %w", i, err)
		}
	}

	return nil
}

// =============================================================================
// Configuration Types
// =============================================================================

// BudgetConfig defines resource limits for ensemble execution.
type BudgetConfig struct {
	// MaxTokensPerMode is the token limit for each mode's response.
	MaxTokensPerMode int `json:"max_tokens_per_mode,omitempty" toml:"max_tokens_per_mode"`

	// MaxTotalTokens is the total token budget across all modes.
	MaxTotalTokens int `json:"max_total_tokens,omitempty" toml:"max_total_tokens"`

	// TimeoutPerMode is the max duration for each mode to complete.
	TimeoutPerMode time.Duration `json:"timeout_per_mode,omitempty" toml:"timeout_per_mode"`

	// TotalTimeout is the max duration for the entire ensemble.
	TotalTimeout time.Duration `json:"total_timeout,omitempty" toml:"total_timeout"`

	// MaxRetries is how many times to retry failed modes.
	MaxRetries int `json:"max_retries,omitempty" toml:"max_retries"`
}

// DefaultBudgetConfig returns sensible default budget limits.
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		MaxTokensPerMode: 4000,
		MaxTotalTokens:   50000,
		TimeoutPerMode:   5 * time.Minute,
		TotalTimeout:     30 * time.Minute,
		MaxRetries:       2,
	}
}

// SynthesisConfig defines how ensemble outputs are combined.
type SynthesisConfig struct {
	// Strategy is the synthesis approach to use.
	Strategy SynthesisStrategy `json:"strategy" toml:"strategy"`

	// MinConfidence is the minimum confidence threshold for inclusion.
	MinConfidence Confidence `json:"min_confidence,omitempty" toml:"min_confidence"`

	// MaxFindings limits how many findings to include in synthesis.
	MaxFindings int `json:"max_findings,omitempty" toml:"max_findings"`

	// IncludeRawOutputs includes original mode outputs in synthesis.
	IncludeRawOutputs bool `json:"include_raw_outputs,omitempty" toml:"include_raw_outputs"`

	// ConflictResolution specifies how to handle disagreements.
	ConflictResolution string `json:"conflict_resolution,omitempty" toml:"conflict_resolution"`
}

// DefaultSynthesisConfig returns sensible default synthesis settings.
func DefaultSynthesisConfig() SynthesisConfig {
	return SynthesisConfig{
		Strategy:           StrategyConsensus,
		MinConfidence:      0.5,
		MaxFindings:        10,
		IncludeRawOutputs:  false,
		ConflictResolution: "highlight",
	}
}

// Ensemble is a curated collection of modes for a specific use case.
// This is the primary user-facing interface - users select ensembles,
// not individual modes. Modes are internal implementation details.
type Ensemble struct {
	// Name is the unique identifier (e.g., "project-diagnosis").
	Name string `json:"name" toml:"name"`

	// DisplayName is the user-facing name (e.g., "Project Diagnosis").
	DisplayName string `json:"display_name" toml:"display_name"`

	// Description explains what this ensemble is for.
	Description string `json:"description" toml:"description"`

	// ModeIDs lists the reasoning modes in this ensemble.
	ModeIDs []string `json:"mode_ids" toml:"mode_ids"`

	// Synthesis configures how outputs are combined.
	Synthesis SynthesisConfig `json:"synthesis" toml:"synthesis"`

	// Budget defines resource limits.
	Budget BudgetConfig `json:"budget" toml:"budget"`

	// Tags enable filtering and discovery.
	Tags []string `json:"tags,omitempty" toml:"tags"`

	// Icon is a single emoji or glyph for UI display.
	Icon string `json:"icon,omitempty" toml:"icon"`

	// Source indicates where this ensemble was loaded from.
	Source string `json:"source,omitempty" toml:"-"`
}

// Validate checks that the ensemble is valid and all mode IDs exist in the catalog.
func (e *Ensemble) Validate(catalog *ModeCatalog) error {
	if e.Name == "" {
		return errors.New("ensemble name is required")
	}
	if err := ValidateModeID(e.Name); err != nil {
		return fmt.Errorf("invalid ensemble name: %w", err)
	}
	if e.DisplayName == "" {
		return errors.New("ensemble display_name is required")
	}
	if len(e.ModeIDs) == 0 {
		return errors.New("ensemble must have at least one mode")
	}

	// Verify all modes exist
	for _, modeID := range e.ModeIDs {
		if catalog.GetMode(modeID) == nil {
			return fmt.Errorf("mode %q not found in catalog", modeID)
		}
	}

	return nil
}
