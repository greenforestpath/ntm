package ensemble

import (
	"log/slog"
	"strings"
)

// EarlyStopConfig controls early stopping behavior for ensembles.
type EarlyStopConfig struct {
	Enabled             bool    `json:"enabled" toml:"enabled" yaml:"enabled"`
	MinAgentsBeforeStop int     `json:"min_agents_before_stop" toml:"min_agents_before_stop" yaml:"min_agents_before_stop"`
	FindingsThreshold   float64 `json:"findings_threshold" toml:"findings_threshold" yaml:"findings_threshold"`
	SimilarityThreshold float64 `json:"similarity_threshold" toml:"similarity_threshold" yaml:"similarity_threshold"`
	WindowSize          int     `json:"window_size" toml:"window_size" yaml:"window_size"`
}

// EarlyStopDetector evaluates marginal utility across mode outputs.
type EarlyStopDetector struct {
	Config      EarlyStopConfig
	Outputs     []ModeOutput
	TokensSpent []int
	Logger      *slog.Logger
}

// StopDecision is the result of an early stop evaluation.
type StopDecision struct {
	ShouldStop      bool
	Reason          string
	FindingsRate    float64
	SimilarityScore float64
	AgentsRun       int
}

// NewEarlyStopDetector creates a detector with the provided configuration.
func NewEarlyStopDetector(cfg EarlyStopConfig) *EarlyStopDetector {
	return &EarlyStopDetector{
		Config: cfg,
		Logger: slog.Default(),
	}
}

// RecordOutput appends a mode output and its token cost.
func (d *EarlyStopDetector) RecordOutput(output ModeOutput, tokens int) {
	if d == nil {
		return
	}
	if tokens < 0 {
		tokens = 0
	}
	d.Outputs = append(d.Outputs, output)
	d.TokensSpent = append(d.TokensSpent, tokens)
}

// ShouldStop evaluates whether the ensemble should stop early.
func (d *EarlyStopDetector) ShouldStop() StopDecision {
	decision := StopDecision{}
	if d == nil {
		return decision
	}
	decision.AgentsRun = len(d.Outputs)
	if !d.Config.Enabled {
		decision.Reason = "disabled"
		return decision
	}

	minAgents := d.Config.MinAgentsBeforeStop
	if minAgents < 0 {
		minAgents = 0
	}
	if decision.AgentsRun < minAgents {
		decision.Reason = "min_agents"
		return decision
	}

	windowOutputs := d.windowOutputs()
	decision.FindingsRate = d.CalculateFindingsRate()
	decision.SimilarityScore = d.CalculateSimilarity()

	rateStop := d.Config.FindingsThreshold > 0 && d.windowTokens() > 0 && decision.FindingsRate < d.Config.FindingsThreshold
	simStop := d.Config.SimilarityThreshold > 0 && len(windowOutputs) >= 2 && decision.SimilarityScore > d.Config.SimilarityThreshold

	switch {
	case rateStop && simStop:
		decision.ShouldStop = true
		decision.Reason = "findings_rate_and_similarity"
	case rateStop:
		decision.ShouldStop = true
		decision.Reason = "findings_rate"
	case simStop:
		decision.ShouldStop = true
		decision.Reason = "similarity"
	default:
		decision.Reason = "continue"
	}

	d.logDecision(decision)
	d.logApproachingThreshold(decision)
	return decision
}

// CalculateFindingsRate returns unique findings per token within the window.
func (d *EarlyStopDetector) CalculateFindingsRate() float64 {
	if d == nil {
		return 0
	}
	outputs := d.windowOutputs()
	if len(outputs) == 0 {
		return 0
	}
	tokens := d.windowTokens()
	if tokens <= 0 {
		return 0
	}

	unique := make(map[string]struct{})
	for _, output := range outputs {
		for _, finding := range output.TopFindings {
			normalized := normalizeText(finding.Finding)
			if normalized == "" {
				continue
			}
			unique[normalized] = struct{}{}
		}
	}
	return float64(len(unique)) / float64(tokens)
}

// CalculateSimilarity returns average pairwise similarity across the window.
func (d *EarlyStopDetector) CalculateSimilarity() float64 {
	if d == nil {
		return 0
	}
	outputs := d.windowOutputs()
	if len(outputs) < 2 {
		return 0
	}

	tokens := make([]map[string]struct{}, 0, len(outputs))
	for _, output := range outputs {
		text := normalizeText(outputSignature(output))
		tokens = append(tokens, tokenize(text))
	}

	var total float64
	var pairs int
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			total += jaccardSimilarity(tokens[i], tokens[j])
			pairs++
		}
	}
	if pairs == 0 {
		return 0
	}
	return total / float64(pairs)
}

func (d *EarlyStopDetector) windowOutputs() []ModeOutput {
	if d == nil || len(d.Outputs) == 0 {
		return nil
	}
	start := d.windowStartIndex()
	return d.Outputs[start:]
}

func (d *EarlyStopDetector) windowTokens() int {
	if d == nil || len(d.TokensSpent) == 0 {
		return 0
	}
	start := d.windowStartIndex()
	total := 0
	for i := start; i < len(d.TokensSpent); i++ {
		total += d.TokensSpent[i]
	}
	return total
}

func (d *EarlyStopDetector) windowStartIndex() int {
	if d == nil {
		return 0
	}
	window := d.Config.WindowSize
	if window <= 0 || window > len(d.Outputs) {
		window = len(d.Outputs)
	}
	start := len(d.Outputs) - window
	if start < 0 {
		start = 0
	}
	return start
}

func outputSignature(output ModeOutput) string {
	parts := make([]string, 0, 1+len(output.TopFindings)+len(output.Risks)+len(output.Recommendations)+len(output.QuestionsForUser))
	if output.Thesis != "" {
		parts = append(parts, output.Thesis)
	}
	for _, finding := range output.TopFindings {
		if finding.Finding != "" {
			parts = append(parts, finding.Finding)
		}
	}
	for _, risk := range output.Risks {
		if risk.Risk != "" {
			parts = append(parts, risk.Risk)
		}
	}
	for _, rec := range output.Recommendations {
		if rec.Recommendation != "" {
			parts = append(parts, rec.Recommendation)
		}
	}
	for _, question := range output.QuestionsForUser {
		if question.Question != "" {
			parts = append(parts, question.Question)
		}
	}
	return strings.Join(parts, " ")
}

func (d *EarlyStopDetector) logDecision(decision StopDecision) {
	logger := d.logger()
	logger.Debug("ensemble early stop decision",
		"agents_run", decision.AgentsRun,
		"window_size", d.Config.WindowSize,
		"findings_rate", decision.FindingsRate,
		"findings_threshold", d.Config.FindingsThreshold,
		"similarity_score", decision.SimilarityScore,
		"similarity_threshold", d.Config.SimilarityThreshold,
		"stop", decision.ShouldStop,
		"reason", decision.Reason,
	)
	if decision.ShouldStop {
		logger.Info("ensemble early stop triggered",
			"agents_run", decision.AgentsRun,
			"reason", decision.Reason,
			"findings_rate", decision.FindingsRate,
			"similarity_score", decision.SimilarityScore,
		)
	}
}

func (d *EarlyStopDetector) logApproachingThreshold(decision StopDecision) {
	if d == nil || decision.ShouldStop {
		return
	}
	logger := d.logger()
	if d.Config.FindingsThreshold > 0 && d.windowTokens() > 0 {
		near := d.Config.FindingsThreshold * 1.1
		if decision.FindingsRate <= near {
			logger.Info("ensemble early stop approaching findings threshold",
				"agents_run", decision.AgentsRun,
				"findings_rate", decision.FindingsRate,
				"threshold", d.Config.FindingsThreshold,
			)
		}
	}
	if d.Config.SimilarityThreshold > 0 && len(d.windowOutputs()) >= 2 {
		near := d.Config.SimilarityThreshold * 0.9
		if decision.SimilarityScore >= near {
			logger.Info("ensemble early stop approaching similarity threshold",
				"agents_run", decision.AgentsRun,
				"similarity_score", decision.SimilarityScore,
				"threshold", d.Config.SimilarityThreshold,
			)
		}
	}
}

func (d *EarlyStopDetector) logger() *slog.Logger {
	if d != nil && d.Logger != nil {
		return d.Logger
	}
	return slog.Default()
}
