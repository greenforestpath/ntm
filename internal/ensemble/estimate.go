//go:build ensemble_experimental
// +build ensemble_experimental

package ensemble

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	tokenpkg "github.com/Dicklesworthstone/ntm/internal/tokens"
)

// EstimateOptions controls estimation behavior.
type EstimateOptions struct {
	// ContextPack overrides context generation when set.
	ContextPack *ContextPack
	// DisableContext skips context pack generation when true.
	DisableContext bool
}

// EnsembleEstimate summarizes token estimates for an ensemble run.
type EnsembleEstimate struct {
	GeneratedAt          time.Time      `json:"generated_at"`
	Question             string         `json:"question,omitempty"`
	PresetUsed           string         `json:"preset_used,omitempty"`
	ModeCount            int            `json:"mode_count"`
	Budget               BudgetConfig   `json:"budget"`
	EstimatedTotalTokens int            `json:"estimated_total_tokens"`
	OverBudget           bool           `json:"over_budget"`
	OverBy               int            `json:"over_by,omitempty"`
	Warnings             []string       `json:"warnings,omitempty"`
	Modes                []ModeEstimate `json:"modes"`
}

// ModeEstimate captures token estimates for a single mode.
type ModeEstimate struct {
	ID                  string            `json:"id"`
	Code                string            `json:"code,omitempty"`
	Name                string            `json:"name,omitempty"`
	Category            string            `json:"category,omitempty"`
	Tier                string            `json:"tier,omitempty"`
	PromptTokens        int               `json:"prompt_tokens"`
	BasePromptTokens    int               `json:"base_prompt_tokens,omitempty"`
	ContextTokens       int               `json:"context_tokens,omitempty"`
	OutputTokens        int               `json:"output_tokens"`
	TypicalOutputTokens int               `json:"typical_output_tokens"`
	TotalTokens         int               `json:"total_tokens"`
	ValueScore          float64           `json:"value_score"`
	ValuePerToken       float64           `json:"value_per_token"`
	Alternatives        []ModeAlternative `json:"alternatives,omitempty"`
}

// ModeAlternative suggests a lower-cost alternative to a mode.
type ModeAlternative struct {
	ID              string  `json:"id"`
	Code            string  `json:"code,omitempty"`
	Name            string  `json:"name,omitempty"`
	EstimatedTokens int     `json:"estimated_tokens"`
	Savings         int     `json:"savings"`
	ValueScore      float64 `json:"value_score"`
	ValuePerToken   float64 `json:"value_per_token"`
	Reason          string  `json:"reason,omitempty"`
}

// EstimateEnsemble estimates token usage and budget fit for an ensemble config.
func (m *EnsembleManager) EstimateEnsemble(ctx context.Context, cfg *EnsembleConfig, opts EstimateOptions) (*EnsembleEstimate, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ensemble config is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	logger := m.logger()
	catalog, err := m.catalog()
	if err != nil {
		return nil, fmt.Errorf("load mode catalog: %w", err)
	}

	var registry *EnsembleRegistry
	if cfg.Ensemble != "" {
		registry, err = m.registry(catalog)
		if err != nil {
			return nil, fmt.Errorf("load ensemble registry: %w", err)
		}
	}

	modeIDs, resolvedCfg, _, err := resolveEnsembleConfig(cfg, catalog, registry)
	if err != nil {
		return nil, err
	}

	question := strings.TrimSpace(cfg.Question)

	var pack *ContextPack
	if opts.ContextPack != nil {
		pack = opts.ContextPack
	} else if !opts.DisableContext {
		generator, cacheCfg := m.contextPackGenerator(cfg.ProjectDir, resolvedCfg.cache)
		if generated, genErr := generator.Generate(question, "", cacheCfg); genErr == nil {
			pack = generated
		} else {
			logger.Warn("context pack generation failed", "error", genErr)
		}
	}

	engine := NewPreambleEngine()
	estimateCache := make(map[string]ModeEstimate, len(modeIDs))

	estimateMode := func(mode *ReasoningMode) (ModeEstimate, error) {
		if cached, ok := estimateCache[mode.ID]; ok {
			return cached, nil
		}

		preamble, err := engine.Render(&PreambleData{
			Problem:     question,
			ContextPack: pack,
			Mode:        mode,
			TokenCap:    resolvedCfg.budget.MaxTokensPerMode,
		})
		if err != nil {
			return ModeEstimate{}, fmt.Errorf("render preamble for %s: %w", mode.ID, err)
		}

		promptTokens := tokenpkg.EstimateTokensWithLanguageHint(preamble, tokenpkg.ContentMarkdown)
		contextTokens := 0
		if pack != nil {
			contextTokens = pack.TokenEstimate
		}
		basePromptTokens := promptTokens
		if contextTokens > 0 && promptTokens > contextTokens {
			basePromptTokens = promptTokens - contextTokens
		}

		typicalOutput := estimateTypicalCost(mode)
		outputTokens := typicalOutput
		if resolvedCfg.budget.MaxTokensPerMode > 0 && outputTokens > resolvedCfg.budget.MaxTokensPerMode {
			outputTokens = resolvedCfg.budget.MaxTokensPerMode
		}

		totalTokens := promptTokens + outputTokens
		valueScore := modeValueScore(mode)
		valuePerToken := 0.0
		if totalTokens > 0 {
			valuePerToken = valueScore / float64(totalTokens)
		}

		estimate := ModeEstimate{
			ID:                  mode.ID,
			Code:                mode.Code,
			Name:                mode.Name,
			Category:            mode.Category.String(),
			Tier:                mode.Tier.String(),
			PromptTokens:        promptTokens,
			BasePromptTokens:    basePromptTokens,
			ContextTokens:       contextTokens,
			OutputTokens:        outputTokens,
			TypicalOutputTokens: typicalOutput,
			TotalTokens:         totalTokens,
			ValueScore:          valueScore,
			ValuePerToken:       valuePerToken,
		}

		estimateCache[mode.ID] = estimate

		logger.Info("ensemble estimate mode",
			"mode_id", mode.ID,
			"prompt_tokens", promptTokens,
			"output_tokens", outputTokens,
			"typical_output_tokens", typicalOutput,
			"calibration_delta", outputTokens-typicalOutput,
			"total_tokens", totalTokens,
		)

		return estimate, nil
	}

	result := &EnsembleEstimate{
		GeneratedAt: time.Now().UTC(),
		Question:    question,
		PresetUsed:  resolvedCfg.presetName,
		ModeCount:   len(modeIDs),
		Budget:      resolvedCfg.budget,
		Modes:       make([]ModeEstimate, 0, len(modeIDs)),
	}

	for _, modeID := range modeIDs {
		mode := catalog.GetMode(modeID)
		if mode == nil {
			return nil, fmt.Errorf("mode %q not found in catalog", modeID)
		}
		estimate, err := estimateMode(mode)
		if err != nil {
			return nil, err
		}
		result.Modes = append(result.Modes, estimate)
		result.EstimatedTotalTokens += estimate.TotalTokens
	}

	reserveTokens := resolvedCfg.budget.SynthesisReserveTokens + resolvedCfg.budget.ContextReserveTokens
	if reserveTokens > 0 {
		result.EstimatedTotalTokens += reserveTokens
	}

	if resolvedCfg.budget.MaxTotalTokens > 0 && result.EstimatedTotalTokens > resolvedCfg.budget.MaxTotalTokens {
		result.OverBudget = true
		result.OverBy = result.EstimatedTotalTokens - resolvedCfg.budget.MaxTotalTokens
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("estimated tokens (%d) exceed budget (%d) by %d",
				result.EstimatedTotalTokens, resolvedCfg.budget.MaxTotalTokens, result.OverBy),
		)
	}

	for _, est := range result.Modes {
		if resolvedCfg.budget.MaxTokensPerMode > 0 && est.TypicalOutputTokens > resolvedCfg.budget.MaxTokensPerMode {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("mode %s typical output (%d) exceeds per-mode cap (%d)",
					est.ID, est.TypicalOutputTokens, resolvedCfg.budget.MaxTokensPerMode),
			)
		}
	}

	allowAdvanced := cfg.AllowAdvanced
	if !allowAdvanced {
		for _, est := range result.Modes {
			if est.Tier != string(TierCore) {
				allowAdvanced = true
				break
			}
		}
	}

	if result.OverBudget {
		for i := range result.Modes {
			mode := catalog.GetMode(result.Modes[i].ID)
			if mode == nil {
				continue
			}
			result.Modes[i].Alternatives = suggestAlternatives(mode, result.Modes[i], catalog, allowAdvanced, estimateMode)
		}
	}

	logger.Info("ensemble estimate summary",
		"modes", len(result.Modes),
		"estimated_total_tokens", result.EstimatedTotalTokens,
		"budget_total", resolvedCfg.budget.MaxTotalTokens,
		"over_budget", result.OverBudget,
	)

	return result, nil
}

func modeValueScore(mode *ReasoningMode) float64 {
	if mode == nil {
		return 0.0
	}

	score := 1.0
	switch mode.Tier {
	case TierCore:
		score *= 1.0
	case TierAdvanced:
		score *= 0.85
	case TierExperimental:
		score *= 0.7
	default:
		score *= 0.9
	}

	if len(mode.BestFor) > 0 {
		score += math.Min(0.3, 0.03*float64(len(mode.BestFor)))
	}
	if strings.TrimSpace(mode.Differentiator) != "" {
		score += 0.05
	}

	return score
}

func suggestAlternatives(
	mode *ReasoningMode,
	current ModeEstimate,
	catalog *ModeCatalog,
	allowAdvanced bool,
	estimateMode func(*ReasoningMode) (ModeEstimate, error),
) []ModeAlternative {
	if mode == nil || catalog == nil || estimateMode == nil {
		return nil
	}

	candidates := catalog.ListByCategory(mode.Category)
	if len(candidates) == 0 {
		return nil
	}

	minSavings := int(math.Max(200, float64(current.TotalTokens)*0.1))
	alternatives := make([]ModeAlternative, 0, 3)

	for i := range candidates {
		candidate := candidates[i]
		if candidate.ID == mode.ID {
			continue
		}
		if !allowAdvanced && candidate.Tier != TierCore {
			continue
		}

		estimate, err := estimateMode(&candidate)
		if err != nil {
			continue
		}
		if estimate.TotalTokens >= current.TotalTokens {
			continue
		}

		savings := current.TotalTokens - estimate.TotalTokens
		if savings < minSavings {
			continue
		}

		alternatives = append(alternatives, ModeAlternative{
			ID:              candidate.ID,
			Code:            candidate.Code,
			Name:            candidate.Name,
			EstimatedTokens: estimate.TotalTokens,
			Savings:         savings,
			ValueScore:      estimate.ValueScore,
			ValuePerToken:   estimate.ValuePerToken,
			Reason:          fmt.Sprintf("lower-cost %s-tier mode in %s category", candidate.Tier, candidate.Category),
		})
	}

	sort.Slice(alternatives, func(i, j int) bool {
		if alternatives[i].ValuePerToken == alternatives[j].ValuePerToken {
			return alternatives[i].Savings > alternatives[j].Savings
		}
		return alternatives[i].ValuePerToken > alternatives[j].ValuePerToken
	})

	if len(alternatives) > 3 {
		alternatives = alternatives[:3]
	}

	return alternatives
}
