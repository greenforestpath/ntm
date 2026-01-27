//go:build !ensemble_experimental
// +build !ensemble_experimental

// Package robot provides machine-readable output for AI agents.
// ensemble_spawn_stub.go provides a stub when ensemble_experimental is disabled.
package robot

import (
	"fmt"
	"strings"

	"github.com/Dicklesworthstone/ntm/internal/config"
)

// EnsembleSpawnOptions configures --robot-ensemble-spawn.
type EnsembleSpawnOptions struct {
	Session       string
	Preset        string
	Modes         string
	Question      string
	Agents        string
	Assignment    string
	AllowAdvanced bool
	BudgetTotal   int
	BudgetPerMode int
	NoCache       bool
	NoQuestions   bool
	ProjectDir    string
}

// EnsembleSpawnOutput is the structured output for --robot-ensemble-spawn.
type EnsembleSpawnOutput struct {
	RobotResponse
	Action  string `json:"action"`
	Session string `json:"session"`
}

// PrintEnsembleSpawn returns a not implemented response when ensemble_experimental is disabled.
func PrintEnsembleSpawn(opts EnsembleSpawnOptions, _ *config.Config) error {
	output := EnsembleSpawnOutput{
		RobotResponse: NewErrorResponse(
			fmt.Errorf("ensemble spawn is experimental"),
			ErrCodeNotImplemented,
			"Rebuild with -tags ensemble_experimental to enable --robot-ensemble-spawn",
		),
		Action:  "ensemble_spawn",
		Session: strings.TrimSpace(opts.Session),
	}
	return outputJSON(output)
}
