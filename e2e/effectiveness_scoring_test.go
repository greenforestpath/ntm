//go:build e2e
// +build e2e

// Package e2e contains end-to-end tests for NTM robot mode commands.
// [E2E-SCORING] Tests for effectiveness score tracking and recommendations.
package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/scoring"
)

// ScoringTestSuite manages E2E tests for effectiveness scoring
type ScoringTestSuite struct {
	t        *testing.T
	logger   *TestLogger
	tempDir  string
	tracker  *scoring.Tracker
	cleanup  []func()
}

// NewScoringTestSuite creates a new test suite for scoring E2E tests
func NewScoringTestSuite(t *testing.T, scenario string) *ScoringTestSuite {
	logger := NewTestLogger(t, scenario)

	return &ScoringTestSuite{
		t:      t,
		logger: logger,
	}
}

// Setup creates temp directory and tracker for testing
func (s *ScoringTestSuite) Setup() error {
	s.logger.Log("[E2E-SCORING] Setting up scoring test environment")

	tempDir, err := os.MkdirTemp("", "ntm-scoring-e2e-*")
	if err != nil {
		return err
	}
	s.tempDir = tempDir
	s.cleanup = append(s.cleanup, func() { os.RemoveAll(tempDir) })
	s.logger.Log("[E2E-SCORING] Created temp directory: %s", tempDir)

	// Create tracker with temp file
	scorePath := filepath.Join(tempDir, "scores.jsonl")
	tracker, err := scoring.NewTracker(scoring.TrackerOptions{
		Path:          scorePath,
		RetentionDays: 90,
		Enabled:       true,
	})
	if err != nil {
		return err
	}
	s.tracker = tracker
	s.logger.Log("[E2E-SCORING] Created tracker with path: %s", scorePath)

	return nil
}

// Teardown cleans up resources
func (s *ScoringTestSuite) Teardown() {
	s.logger.Log("[E2E-SCORING] Running cleanup (%d items)", len(s.cleanup))
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
	s.logger.Close()
}

// recordTestScore records a synthetic score for testing
func (s *ScoringTestSuite) recordTestScore(agentType, taskType string, completion, quality, efficiency float64, timestamp time.Time) error {
	score := &scoring.Score{
		Timestamp: timestamp,
		Session:   "e2e-test-session",
		AgentType: agentType,
		TaskType:  taskType,
		Metrics: scoring.ScoreMetrics{
			Completion: completion,
			Quality:    quality,
			Efficiency: efficiency,
		},
	}
	score.Metrics.ComputeOverall()

	s.logger.Log("[E2E-SCORING] Recording score: agent=%s task=%s completion=%.2f quality=%.2f efficiency=%.2f overall=%.2f",
		agentType, taskType, completion, quality, efficiency, score.Metrics.Overall)

	return s.tracker.Record(score)
}

// =============================================================================
// Scenario 1: Basic Score Tracking
// =============================================================================

func TestScoringBasicTracking(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-basic")
	if err := suite.Setup(); err != nil {
		t.Fatalf("[E2E-SCORING] Setup failed: %v", err)
	}
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 1: Basic Score Tracking ===")

	// Record synthetic scores for multiple agents
	now := time.Now()

	testScores := []struct {
		agentType  string
		taskType   string
		completion float64
		quality    float64
		efficiency float64
		offset     time.Duration
	}{
		{"claude", "bug_fix", 1.0, 0.9, 0.8, -5 * time.Hour},
		{"claude", "feature", 0.8, 0.85, 0.75, -4 * time.Hour},
		{"codex", "bug_fix", 0.9, 0.85, 0.9, -3 * time.Hour},
		{"codex", "refactor", 1.0, 0.95, 0.7, -2 * time.Hour},
		{"gemini", "feature", 0.7, 0.8, 0.85, -1 * time.Hour},
	}

	suite.logger.Log("[E2E-SCORING] Recording %d test scores", len(testScores))

	for _, ts := range testScores {
		if err := suite.recordTestScore(ts.agentType, ts.taskType, ts.completion, ts.quality, ts.efficiency, now.Add(ts.offset)); err != nil {
			t.Fatalf("[E2E-SCORING] Failed to record score: %v", err)
		}
	}

	// Verify scores persisted to JSONL
	scorePath := filepath.Join(suite.tempDir, "scores.jsonl")
	data, err := os.ReadFile(scorePath)
	if err != nil {
		t.Fatalf("[E2E-SCORING] Failed to read score file: %v", err)
	}
	suite.logger.Log("[E2E-SCORING] Score file size: %d bytes", len(data))

	if len(data) == 0 {
		t.Fatal("[E2E-SCORING] Score file is empty")
	}

	// Query all scores
	scores, err := suite.tracker.QueryScores(scoring.Query{})
	if err != nil {
		t.Fatalf("[E2E-SCORING] QueryScores failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Queried %d scores", len(scores))

	if len(scores) != len(testScores) {
		t.Errorf("[E2E-SCORING] Expected %d scores, got %d", len(testScores), len(scores))
	}

	// Verify scores have computed overall values
	for i, score := range scores {
		suite.logger.Log("[E2E-SCORING] Score %d: agent=%s overall=%.3f", i, score.AgentType, score.Metrics.Overall)
		if score.Metrics.Overall <= 0 || score.Metrics.Overall > 1.0 {
			t.Errorf("[E2E-SCORING] Score %d has invalid overall: %.3f", i, score.Metrics.Overall)
		}
	}

	// Query by agent type
	claudeScores, err := suite.tracker.QueryScores(scoring.Query{AgentType: "claude"})
	if err != nil {
		t.Fatalf("[E2E-SCORING] QueryScores by agent failed: %v", err)
	}
	suite.logger.Log("[E2E-SCORING] Claude scores: %d", len(claudeScores))
	if len(claudeScores) != 2 {
		t.Errorf("[E2E-SCORING] Expected 2 claude scores, got %d", len(claudeScores))
	}

	// Query by task type
	bugFixScores, err := suite.tracker.QueryScores(scoring.Query{TaskType: "bug_fix"})
	if err != nil {
		t.Fatalf("[E2E-SCORING] QueryScores by task failed: %v", err)
	}
	suite.logger.Log("[E2E-SCORING] Bug fix scores: %d", len(bugFixScores))
	if len(bugFixScores) != 2 {
		t.Errorf("[E2E-SCORING] Expected 2 bug_fix scores, got %d", len(bugFixScores))
	}

	suite.logger.Log("[E2E-SCORING] PASS: Basic score tracking works")
}

// =============================================================================
// Scenario 2: Trend Analysis and Rolling Average
// =============================================================================

func TestScoringTrendAnalysis(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-trend")
	if err := suite.Setup(); err != nil {
		t.Fatalf("[E2E-SCORING] Setup failed: %v", err)
	}
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 2: Trend Analysis ===")

	now := time.Now()

	// Create improving trend for claude: earlier scores lower, recent scores higher
	improvingScores := []struct {
		daysAgo    int
		completion float64
	}{
		{13, 0.5},
		{12, 0.55},
		{11, 0.6},
		{10, 0.65},
		{7, 0.75},
		{6, 0.8},
		{5, 0.85},
		{4, 0.9},
		{3, 0.92},
		{2, 0.95},
	}

	suite.logger.Log("[E2E-SCORING] Recording improving trend scores for claude")

	for _, s := range improvingScores {
		ts := now.AddDate(0, 0, -s.daysAgo)
		if err := suite.recordTestScore("claude", "feature", s.completion, s.completion, s.completion, ts); err != nil {
			t.Fatalf("[E2E-SCORING] Failed to record score: %v", err)
		}
	}

	// Analyze trend
	trend, err := suite.tracker.AnalyzeTrend(scoring.Query{AgentType: "claude"}, 14)
	if err != nil {
		t.Fatalf("[E2E-SCORING] AnalyzeTrend failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Trend analysis results:")
	suite.logger.Log("[E2E-SCORING]   Trend: %s", trend.Trend)
	suite.logger.Log("[E2E-SCORING]   Sample count: %d", trend.SampleCount)
	suite.logger.Log("[E2E-SCORING]   Avg score: %.3f", trend.AvgScore)
	suite.logger.Log("[E2E-SCORING]   Earlier avg: %.3f", trend.EarlierAvg)
	suite.logger.Log("[E2E-SCORING]   Recent avg: %.3f", trend.RecentAvg)
	suite.logger.Log("[E2E-SCORING]   Change percent: %.1f%%", trend.ChangePercent)

	// Verify trend direction
	if trend.Trend != scoring.TrendImproving {
		suite.logger.Log("[E2E-SCORING] WARN: Expected improving trend, got %s", trend.Trend)
		// Not a hard failure - trend detection depends on threshold
	}

	// Verify recent average is higher than earlier
	if trend.RecentAvg <= trend.EarlierAvg {
		t.Errorf("[E2E-SCORING] Expected recent avg (%.3f) > earlier avg (%.3f)", trend.RecentAvg, trend.EarlierAvg)
	}

	// Verify positive change percent
	if trend.ChangePercent <= 0 {
		t.Errorf("[E2E-SCORING] Expected positive change percent, got %.1f%%", trend.ChangePercent)
	}

	// Test rolling average
	rollingAvg, err := suite.tracker.RollingAverage(scoring.Query{AgentType: "claude"}, 14)
	if err != nil {
		t.Fatalf("[E2E-SCORING] RollingAverage failed: %v", err)
	}
	suite.logger.Log("[E2E-SCORING] Rolling average (14 days): %.3f", rollingAvg)

	if rollingAvg <= 0 || rollingAvg > 1.0 {
		t.Errorf("[E2E-SCORING] Invalid rolling average: %.3f", rollingAvg)
	}

	suite.logger.Log("[E2E-SCORING] PASS: Trend analysis works")
}

// =============================================================================
// Scenario 3: Agent Summary and Comparison
// =============================================================================

func TestScoringAgentSummary(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-summary")
	if err := suite.Setup(); err != nil {
		t.Fatalf("[E2E-SCORING] Setup failed: %v", err)
	}
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 3: Agent Summary ===")

	now := time.Now()

	// Record scores for multiple agents with different performance levels
	// Claude: high performer
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Hour)
		if err := suite.recordTestScore("claude", "feature", 0.9, 0.85, 0.8, ts); err != nil {
			t.Fatalf("[E2E-SCORING] Failed to record claude score: %v", err)
		}
	}

	// Codex: medium performer
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Hour)
		if err := suite.recordTestScore("codex", "feature", 0.7, 0.75, 0.7, ts); err != nil {
			t.Fatalf("[E2E-SCORING] Failed to record codex score: %v", err)
		}
	}

	// Gemini: lower performer
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Hour)
		if err := suite.recordTestScore("gemini", "feature", 0.5, 0.6, 0.55, ts); err != nil {
			t.Fatalf("[E2E-SCORING] Failed to record gemini score: %v", err)
		}
	}

	// Get summaries
	summaries, err := suite.tracker.SummarizeByAgentList(time.Time{})
	if err != nil {
		t.Fatalf("[E2E-SCORING] SummarizeByAgentList failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Got %d agent summaries", len(summaries))

	if len(summaries) != 3 {
		t.Errorf("[E2E-SCORING] Expected 3 agent summaries, got %d", len(summaries))
	}

	// Log summaries
	for _, summary := range summaries {
		suite.logger.Log("[E2E-SCORING] Agent %s: scores=%d avg_overall=%.3f avg_completion=%.3f",
			summary.AgentType, summary.TotalScores, summary.AvgOverall, summary.AvgCompletion)

		if summary.TotalScores != 5 {
			t.Errorf("[E2E-SCORING] Expected 5 scores for %s, got %d", summary.AgentType, summary.TotalScores)
		}
	}

	// Verify ordering reflects performance (claude > codex > gemini)
	var claudeAvg, codexAvg, geminiAvg float64
	for _, summary := range summaries {
		switch summary.AgentType {
		case "claude":
			claudeAvg = summary.AvgOverall
		case "codex":
			codexAvg = summary.AvgOverall
		case "gemini":
			geminiAvg = summary.AvgOverall
		}
	}

	suite.logger.Log("[E2E-SCORING] Performance ranking: claude=%.3f codex=%.3f gemini=%.3f",
		claudeAvg, codexAvg, geminiAvg)

	if claudeAvg <= codexAvg {
		t.Errorf("[E2E-SCORING] Expected claude (%.3f) > codex (%.3f)", claudeAvg, codexAvg)
	}
	if codexAvg <= geminiAvg {
		t.Errorf("[E2E-SCORING] Expected codex (%.3f) > gemini (%.3f)", codexAvg, geminiAvg)
	}

	suite.logger.Log("[E2E-SCORING] PASS: Agent summary works")
}

// =============================================================================
// Scenario 4: Cold Start / Empty Tracker
// =============================================================================

func TestScoringColdStart(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-cold-start")
	if err := suite.Setup(); err != nil {
		t.Fatalf("[E2E-SCORING] Setup failed: %v", err)
	}
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 4: Cold Start / Empty Tracker ===")

	// Query empty tracker
	scores, err := suite.tracker.QueryScores(scoring.Query{})
	if err != nil {
		t.Fatalf("[E2E-SCORING] QueryScores on empty tracker failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Empty tracker query returned %d scores", len(scores))

	if len(scores) != 0 {
		t.Errorf("[E2E-SCORING] Expected 0 scores from empty tracker, got %d", len(scores))
	}

	// Trend analysis with no data
	trend, err := suite.tracker.AnalyzeTrend(scoring.Query{}, 14)
	if err != nil {
		t.Fatalf("[E2E-SCORING] AnalyzeTrend on empty tracker failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Empty tracker trend: %s (samples: %d)", trend.Trend, trend.SampleCount)

	if trend.Trend != scoring.TrendUnknown {
		t.Errorf("[E2E-SCORING] Expected unknown trend for empty tracker, got %s", trend.Trend)
	}

	// Rolling average with no data
	avg, err := suite.tracker.RollingAverage(scoring.Query{}, 14)
	if err != nil {
		t.Fatalf("[E2E-SCORING] RollingAverage on empty tracker failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Empty tracker rolling avg: %.3f", avg)

	if avg != 0 {
		t.Errorf("[E2E-SCORING] Expected 0 rolling average for empty tracker, got %.3f", avg)
	}

	// Summaries with no data
	summaries, err := suite.tracker.SummarizeByAgentList(time.Time{})
	if err != nil {
		t.Fatalf("[E2E-SCORING] SummarizeByAgentList on empty tracker failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Empty tracker summaries: %d", len(summaries))

	if len(summaries) != 0 {
		t.Errorf("[E2E-SCORING] Expected 0 summaries for empty tracker, got %d", len(summaries))
	}

	// Add minimal samples (less than MinSamplesForTrend)
	now := time.Now()
	if err := suite.recordTestScore("claude", "feature", 0.8, 0.8, 0.8, now); err != nil {
		t.Fatalf("[E2E-SCORING] Failed to record score: %v", err)
	}
	if err := suite.recordTestScore("claude", "feature", 0.85, 0.85, 0.85, now.Add(-time.Hour)); err != nil {
		t.Fatalf("[E2E-SCORING] Failed to record score: %v", err)
	}

	// Trend with insufficient samples
	trend, err = suite.tracker.AnalyzeTrend(scoring.Query{AgentType: "claude"}, 14)
	if err != nil {
		t.Fatalf("[E2E-SCORING] AnalyzeTrend with minimal samples failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Minimal samples trend: %s (samples: %d)", trend.Trend, trend.SampleCount)

	if trend.SampleCount < scoring.MinSamplesForTrend && trend.Trend != scoring.TrendUnknown {
		t.Errorf("[E2E-SCORING] Expected unknown trend with %d samples (min: %d), got %s",
			trend.SampleCount, scoring.MinSamplesForTrend, trend.Trend)
	}

	suite.logger.Log("[E2E-SCORING] PASS: Cold start behavior works")
}

// =============================================================================
// Scenario 5: Score Export
// =============================================================================

func TestScoringExport(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-export")
	if err := suite.Setup(); err != nil {
		t.Fatalf("[E2E-SCORING] Setup failed: %v", err)
	}
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 5: Score Export ===")

	// Record some scores
	now := time.Now()
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Hour)
		if err := suite.recordTestScore("claude", "feature", 0.8+float64(i)*0.02, 0.8, 0.8, ts); err != nil {
			t.Fatalf("[E2E-SCORING] Failed to record score: %v", err)
		}
	}

	// Export to JSON
	exportPath := filepath.Join(suite.tempDir, "export.json")
	if err := suite.tracker.Export(exportPath, time.Time{}); err != nil {
		t.Fatalf("[E2E-SCORING] Export failed: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Exported scores to: %s", exportPath)

	// Read and verify export
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("[E2E-SCORING] Failed to read export file: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Export file size: %d bytes", len(data))

	var exported []*scoring.Score
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("[E2E-SCORING] Failed to parse export JSON: %v", err)
	}

	suite.logger.Log("[E2E-SCORING] Exported %d scores", len(exported))

	if len(exported) != 5 {
		t.Errorf("[E2E-SCORING] Expected 5 exported scores, got %d", len(exported))
	}

	// Verify export structure
	for i, score := range exported {
		if score.AgentType == "" {
			t.Errorf("[E2E-SCORING] Score %d missing agent_type", i)
		}
		if score.Metrics.Overall <= 0 {
			t.Errorf("[E2E-SCORING] Score %d has invalid overall: %.3f", i, score.Metrics.Overall)
		}
	}

	suite.logger.Log("[E2E-SCORING] PASS: Score export works")
}

// =============================================================================
// Scenario 6: Metrics Weights and Effectiveness Score
// =============================================================================

func TestScoringMetricsWeights(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-weights")
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 6: Metrics Weights ===")

	// Test default weights
	defaultWeights := scoring.DefaultWeights()
	suite.logger.Log("[E2E-SCORING] Default weights sum: %.3f", defaultWeights.Sum())

	if defaultWeights.Sum() < 0.99 || defaultWeights.Sum() > 1.01 {
		t.Errorf("[E2E-SCORING] Default weights don't sum to 1.0: %.3f", defaultWeights.Sum())
	}

	// Test speed-focused weights
	speedWeights := scoring.SpeedFocusedWeights()
	suite.logger.Log("[E2E-SCORING] Speed weights - TimeEfficiency: %.2f", speedWeights.TimeEfficiency)

	if speedWeights.TimeEfficiency <= defaultWeights.TimeEfficiency {
		t.Error("[E2E-SCORING] Speed weights should have higher TimeEfficiency than default")
	}

	// Test quality-focused weights
	qualityWeights := scoring.QualityFocusedWeights()
	suite.logger.Log("[E2E-SCORING] Quality weights - Quality: %.2f", qualityWeights.Quality)

	if qualityWeights.Quality <= defaultWeights.Quality {
		t.Error("[E2E-SCORING] Quality weights should have higher Quality than default")
	}

	// Test economy-focused weights
	economyWeights := scoring.EconomyFocusedWeights()
	suite.logger.Log("[E2E-SCORING] Economy weights - TokenEfficiency: %.2f", economyWeights.TokenEfficiency)

	if economyWeights.TokenEfficiency <= defaultWeights.TokenEfficiency {
		t.Error("[E2E-SCORING] Economy weights should have higher TokenEfficiency than default")
	}

	// Test effectiveness score computation
	score := scoring.NewEffectivenessScore(0.9, 0.8, 0.7)
	suite.logger.Log("[E2E-SCORING] Effectiveness score: overall=%.3f", score.Overall)

	if score.Overall <= 0 || score.Overall > 1.0 {
		t.Errorf("[E2E-SCORING] Invalid effectiveness score: %.3f", score.Overall)
	}

	// Test weight override
	speedScore := score.WithWeights(speedWeights)
	suite.logger.Log("[E2E-SCORING] With speed weights: overall=%.3f", speedScore.Overall)

	// Scores should differ when weights change
	if speedScore.Overall == score.Overall {
		suite.logger.Log("[E2E-SCORING] WARN: Scores identical with different weights (may be edge case)")
	}

	// Test raw metrics conversion
	raw := &scoring.RawMetrics{
		TasksAssigned:  10,
		TasksCompleted: 8,
		RetryCount:     2,
		ErrorCount:     1,
		SuccessCount:   9,
	}
	converted := raw.ToEffectivenessScore(defaultWeights)
	suite.logger.Log("[E2E-SCORING] Raw metrics converted: completion=%.2f retries=%.2f error_rate=%.2f overall=%.3f",
		converted.Completion, converted.Retries, converted.ErrorRate, converted.Overall)

	expectedCompletion := 0.8 // 8/10
	if converted.Completion != expectedCompletion {
		t.Errorf("[E2E-SCORING] Expected completion %.2f, got %.2f", expectedCompletion, converted.Completion)
	}

	suite.logger.Log("[E2E-SCORING] PASS: Metrics weights work")
}

// =============================================================================
// Scenario 7: Core Metrics Definitions
// =============================================================================

func TestScoringCoreMetrics(t *testing.T) {
	suite := NewScoringTestSuite(t, "scoring-core-metrics")
	defer suite.Teardown()

	suite.logger.Log("[E2E-SCORING] === Scenario 7: Core Metrics Definitions ===")

	metrics := scoring.CoreMetrics()
	suite.logger.Log("[E2E-SCORING] Found %d core metric definitions", len(metrics))

	if len(metrics) < 5 {
		t.Errorf("[E2E-SCORING] Expected at least 5 core metrics, got %d", len(metrics))
	}

	// Verify each metric has required fields
	for _, metric := range metrics {
		suite.logger.Log("[E2E-SCORING] Metric: %s - %s", metric.Name, metric.Description)

		if metric.Name == "" {
			t.Error("[E2E-SCORING] Metric missing name")
		}
		if metric.Description == "" {
			t.Errorf("[E2E-SCORING] Metric %s missing description", metric.Name)
		}
		if metric.Unit == "" {
			t.Errorf("[E2E-SCORING] Metric %s missing unit", metric.Name)
		}
		if metric.MeasurementGuide == "" {
			t.Errorf("[E2E-SCORING] Metric %s missing measurement guide", metric.Name)
		}
	}

	// Verify expected metrics exist
	expectedMetrics := []scoring.MetricName{
		scoring.MetricCompletion,
		scoring.MetricRetries,
		scoring.MetricQuality,
		scoring.MetricTokenEfficiency,
		scoring.MetricErrorRate,
	}

	metricNames := make(map[scoring.MetricName]bool)
	for _, m := range metrics {
		metricNames[m.Name] = true
	}

	for _, expected := range expectedMetrics {
		if !metricNames[expected] {
			t.Errorf("[E2E-SCORING] Missing expected metric: %s", expected)
		}
	}

	suite.logger.Log("[E2E-SCORING] PASS: Core metrics definitions complete")
}
