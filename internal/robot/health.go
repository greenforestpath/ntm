// Package robot provides machine-readable output for AI agents.
// health.go contains the --robot-health flag implementation.
package robot

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/bv"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

// HealthOutput provides a focused project health summary for AI agents
type HealthOutput struct {
	CheckedAt time.Time `json:"checked_at"`

	// System-level health
	System SystemHealthInfo `json:"system"`

	// Agent/session health matrix
	Sessions map[string]SessionHealthInfo `json:"sessions"`

	// Alerts for detected issues
	Alerts []string `json:"alerts"`

	// Project/beads health (existing functionality)
	BvAvailable       bool                  `json:"bv_available"`
	BdAvailable       bool                  `json:"bd_available"`
	Error             string                `json:"error,omitempty"`
	DriftStatus       string                `json:"drift_status,omitempty"`
	DriftMessage      string                `json:"drift_message,omitempty"`
	TopBottlenecks    []bv.NodeScore        `json:"top_bottlenecks,omitempty"`
	TopKeystones      []bv.NodeScore        `json:"top_keystones,omitempty"`
	ReadyCount        int                   `json:"ready_count"`
	InProgressCount   int                   `json:"in_progress_count"`
	BlockedCount      int                   `json:"blocked_count"`
	NextRecommended   []RecommendedAction   `json:"next_recommended,omitempty"`
	DependencyContext *bv.DependencyContext `json:"dependency_context,omitempty"`
}

// SystemHealthInfo contains system-level health metrics
type SystemHealthInfo struct {
	TmuxOK     bool    `json:"tmux_ok"`
	DiskFreeGB float64 `json:"disk_free_gb"`
	LoadAvg    float64 `json:"load_avg"`
}

// SessionHealthInfo contains health info for a single session
type SessionHealthInfo struct {
	Healthy bool                       `json:"healthy"`
	Agents  map[string]AgentHealthInfo `json:"agents"`
}

// AgentHealthInfo contains health metrics for a single agent
type AgentHealthInfo struct {
	Responsive      bool   `json:"responsive"`
	OutputRate      string `json:"output_rate"` // "high", "medium", "low", "none"
	LastActivitySec int    `json:"last_activity_sec"`
	Issue           string `json:"issue,omitempty"`
}

// RecommendedAction is a simplified priority recommendation
type RecommendedAction struct {
	IssueID  string `json:"issue_id"`
	Title    string `json:"title"`
	Reason   string `json:"reason"`
	Priority int    `json:"priority"`
}

// noOutputThreshold is the time in seconds after which an agent is considered unresponsive
const noOutputThreshold = 300 // 5 minutes

// PrintHealth outputs a focused project health summary for AI consumption
func PrintHealth() error {
	output := HealthOutput{
		CheckedAt:   time.Now().UTC(),
		BvAvailable: bv.IsInstalled(),
		BdAvailable: bv.IsBdInstalled(),
		Sessions:    make(map[string]SessionHealthInfo),
		Alerts:      []string{},
	}

	// Get system health
	output.System = getSystemHealth()

	// Get agent/session health matrix
	populateAgentHealth(&output)

	// Get drift status
	drift := bv.CheckDrift("")
	output.DriftStatus = drift.Status.String()
	output.DriftMessage = drift.Message

	// Get top bottlenecks (limit to 5)
	bottlenecks, err := bv.GetTopBottlenecks("", 5)
	if err == nil {
		output.TopBottlenecks = bottlenecks
	}

	// Get insights for keystones
	insights, err := bv.GetInsights("")
	if err == nil && insights != nil {
		keystones := insights.Keystones
		if len(keystones) > 5 {
			keystones = keystones[:5]
		}
		output.TopKeystones = keystones
	}

	// Get priority recommendations
	recommendations, err := bv.GetNextActions("", 5)
	if err == nil {
		for _, rec := range recommendations {
			var reason string
			if len(rec.Reasoning) > 0 {
				reason = rec.Reasoning[0]
			}
			output.NextRecommended = append(output.NextRecommended, RecommendedAction{
				IssueID:  rec.IssueID,
				Title:    rec.Title,
				Reason:   reason,
				Priority: rec.SuggestedPriority,
			})
		}
	}

	// Get dependency context (includes ready/in-progress/blocked counts)
	depCtx, err := bv.GetDependencyContext("", 5)
	if err == nil {
		output.DependencyContext = depCtx
		output.ReadyCount = depCtx.ReadyCount
		output.BlockedCount = depCtx.BlockedCount
		output.InProgressCount = len(depCtx.InProgressTasks)
	}

	return encodeJSON(output)
}

// getSystemHealth returns system-level health metrics
func getSystemHealth() SystemHealthInfo {
	info := SystemHealthInfo{
		TmuxOK: tmux.IsInstalled(),
	}

	// Get disk free space (platform-specific)
	info.DiskFreeGB = getDiskFreeGB()

	// Get load average (platform-specific)
	info.LoadAvg = getLoadAverage()

	return info
}

// getDiskFreeGB returns the free disk space in GB for the current directory
func getDiskFreeGB() float64 {
	switch runtime.GOOS {
	case "darwin", "linux":
		// Use df command
		cmd := exec.Command("df", "-k", ".")
		out, err := cmd.Output()
		if err != nil {
			return -1
		}
		lines := strings.Split(string(out), "\n")
		if len(lines) < 2 {
			return -1
		}
		// Parse the second line (data line)
		fields := strings.Fields(lines[1])
		if len(fields) < 4 {
			return -1
		}
		// Field 3 is available space in KB
		availKB, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			return -1
		}
		return availKB / (1024 * 1024) // Convert KB to GB
	default:
		return -1
	}
}

// getLoadAverage returns the 1-minute load average
func getLoadAverage() float64 {
	switch runtime.GOOS {
	case "darwin", "linux":
		// Use sysctl on macOS, /proc/loadavg on Linux
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("sysctl", "-n", "vm.loadavg")
			out, err := cmd.Output()
			if err != nil {
				return -1
			}
			// Output format: "{ 1.23 2.34 3.45 }"
			s := strings.TrimSpace(string(out))
			s = strings.TrimPrefix(s, "{ ")
			s = strings.TrimSuffix(s, " }")
			fields := strings.Fields(s)
			if len(fields) < 1 {
				return -1
			}
			load, err := strconv.ParseFloat(fields[0], 64)
			if err != nil {
				return -1
			}
			return load
		}
		// Linux: read from /proc/loadavg
		cmd := exec.Command("cat", "/proc/loadavg")
		out, err := cmd.Output()
		if err != nil {
			return -1
		}
		fields := strings.Fields(string(out))
		if len(fields) < 1 {
			return -1
		}
		load, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return -1
		}
		return load
	default:
		return -1
	}
}

// populateAgentHealth fills in the agent health matrix for all sessions
func populateAgentHealth(output *HealthOutput) {
	if !output.System.TmuxOK {
		output.Alerts = append(output.Alerts, "tmux not available")
		return
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		output.Alerts = append(output.Alerts, fmt.Sprintf("failed to list sessions: %v", err))
		return
	}

	for _, sess := range sessions {
		sessHealth := SessionHealthInfo{
			Healthy: true,
			Agents:  make(map[string]AgentHealthInfo),
		}

		panes, err := tmux.GetPanes(sess.Name)
		if err != nil {
			output.Alerts = append(output.Alerts, fmt.Sprintf("%s: failed to get panes: %v", sess.Name, err))
			sessHealth.Healthy = false
			output.Sessions[sess.Name] = sessHealth
			continue
		}

		for _, pane := range panes {
			paneKey := fmt.Sprintf("%d.%d", 0, pane.Index)
			agentHealth := getAgentHealth(sess.Name, pane)

			sessHealth.Agents[paneKey] = agentHealth

			// Check for issues and add to alerts
			if !agentHealth.Responsive {
				sessHealth.Healthy = false
				output.Alerts = append(output.Alerts, fmt.Sprintf("%s %s: %s", sess.Name, paneKey, agentHealth.Issue))
			}
		}

		output.Sessions[sess.Name] = sessHealth
	}
}

// getAgentHealth calculates health metrics for a single agent pane
func getAgentHealth(session string, pane tmux.Pane) AgentHealthInfo {
	health := AgentHealthInfo{
		Responsive:      true,
		OutputRate:      "unknown",
		LastActivitySec: -1,
	}

	// Get pane activity time
	activityTime, err := tmux.GetPaneActivity(pane.ID)
	if err == nil {
		health.LastActivitySec = int(time.Since(activityTime).Seconds())

		// Check if unresponsive (no output for threshold time)
		if health.LastActivitySec > noOutputThreshold {
			health.Responsive = false
			health.Issue = fmt.Sprintf("no_output_%dm", noOutputThreshold/60)
		}
	}

	// Calculate output rate from recent activity
	health.OutputRate = calculateOutputRate(health.LastActivitySec)

	// Capture recent output to detect error states
	captured, err := tmux.CapturePaneOutput(pane.ID, 20)
	if err == nil {
		lines := splitLines(stripANSI(captured))
		state := detectState(lines, pane.Title)

		if state == "error" {
			health.Responsive = false
			health.Issue = "error_state_detected"
		}
	}

	return health
}

// calculateOutputRate determines output rate based on last activity time
func calculateOutputRate(lastActivitySec int) string {
	if lastActivitySec < 0 {
		return "unknown"
	}
	switch {
	case lastActivitySec <= 1:
		return "high" // >1 line/sec equivalent
	case lastActivitySec <= 10:
		return "medium"
	case lastActivitySec <= 60:
		return "low" // <1 line/min equivalent
	default:
		return "none"
	}
}

// =============================================================================
// Agent Health States and Activity Detection Integration
// =============================================================================
//
// Note: HealthState enum is defined in routing.go with values:
// - HealthHealthy, HealthDegraded, HealthUnhealthy, HealthRateLimited

// HealthCheck contains the result of a comprehensive health check
type HealthCheck struct {
	PaneID       string      `json:"pane_id"`
	AgentType    string      `json:"agent_type"`
	HealthState  HealthState `json:"health_state"`
	ProcessCheck *ProcessCheckResult `json:"process_check"`
	StallCheck   *StallCheckResult   `json:"stall_check"`
	ErrorCheck   *ErrorCheckResult   `json:"error_check"`
	Confidence   float64     `json:"confidence"`
	Reason       string      `json:"reason"`
	CheckedAt    time.Time   `json:"checked_at"`
}

// ProcessCheckResult contains the result of process-level health check
type ProcessCheckResult struct {
	Running    bool   `json:"running"`
	Crashed    bool   `json:"crashed"`
	ExitStatus string `json:"exit_status,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// StallCheckResult contains the result of stall detection using activity detection
type StallCheckResult struct {
	Stalled        bool          `json:"stalled"`
	ActivityState  string        `json:"activity_state"` // from StateClassifier
	Velocity       float64       `json:"velocity"`       // chars/sec
	IdleSeconds    int           `json:"idle_seconds"`
	Confidence     float64       `json:"confidence"`
	Reason         string        `json:"reason,omitempty"`
}

// ErrorCheckResult contains the result of error pattern detection
type ErrorCheckResult struct {
	HasErrors   bool     `json:"has_errors"`
	RateLimited bool     `json:"rate_limited"`
	Patterns    []string `json:"patterns,omitempty"`
	WaitSeconds int      `json:"wait_seconds,omitempty"` // suggested wait time for rate limit
	Reason      string   `json:"reason,omitempty"`
}

// Error patterns for detailed detection (literal string patterns for strings.Contains)
var healthErrorPatterns = []struct {
	Pattern string
	Type    string
}{
	// Rate limit patterns
	{"rate limit", "rate_limit"},
	{"ratelimit", "rate_limit"},
	{"rate-limit", "rate_limit"},
	{"429", "rate_limit"},
	{"too many requests", "rate_limit"},
	{"quota exceeded", "rate_limit"},
	// Auth error patterns
	{"authentication failed", "auth_error"},
	{"authentication error", "auth_error"},
	{"401", "auth_error"},
	{"unauthorized", "auth_error"},
	// Crash patterns
	{"panic:", "crash"},
	{"fatal error", "crash"},
	{"segmentation fault", "crash"},
	{"stack trace", "crash"},
	// Network error patterns
	{"connection refused", "network_error"},
	{"connection reset", "network_error"},
	{"connection timeout", "network_error"},
	{"network error", "network_error"},
	{"network unreachable", "network_error"},
}

// CheckAgentHealthWithActivity performs a comprehensive health check using activity detection
func CheckAgentHealthWithActivity(paneID string, agentType string) (*HealthCheck, error) {
	check := &HealthCheck{
		PaneID:      paneID,
		AgentType:   agentType,
		HealthState: HealthHealthy,
		Confidence:  1.0,
		CheckedAt:   time.Now().UTC(),
	}

	// 1. Process check - is the agent still running or crashed?
	check.ProcessCheck = checkProcess(paneID)

	// 2. Stall check - use activity detection for stall detection
	check.StallCheck = checkStallWithActivity(paneID, agentType)

	// 3. Error check - detect error patterns
	check.ErrorCheck = checkErrors(paneID)

	// Calculate overall health state
	check.HealthState, check.Reason = calculateHealthState(check)

	// Calculate confidence based on checks
	check.Confidence = calculateHealthConfidence(check)

	return check, nil
}

// checkProcess checks if the agent process is running or crashed
func checkProcess(paneID string) *ProcessCheckResult {
	result := &ProcessCheckResult{
		Running: true,
		Crashed: false,
	}

	// Capture pane output to check for exit indicators
	output, err := tmux.CapturePaneOutput(paneID, 30)
	if err != nil {
		result.Reason = "failed to capture pane output"
		return result
	}

	output = stripANSI(output)
	outputLower := strings.ToLower(output)

	// Check for exit indicators
	exitPatterns := []string{
		"exit status", "exited with", "process exited",
		"connection closed", "session ended", "terminated",
	}

	for _, pattern := range exitPatterns {
		if strings.Contains(outputLower, pattern) {
			// Check if it's really a crash (shell prompt at end)
			lines := splitLines(output)
			if len(lines) > 0 {
				lastLine := strings.TrimSpace(lines[len(lines)-1])
				// If the last line looks like a shell prompt, agent may have crashed
				if lastLine == "$" || lastLine == "bash$" || lastLine == "zsh$" ||
					strings.HasSuffix(lastLine, "$") && !strings.Contains(lastLine, ">") {
					result.Running = false
					result.Crashed = true
					result.ExitStatus = pattern
					result.Reason = "detected shell prompt - agent may have crashed"
					return result
				}
			}
		}
	}

	// Check for explicit exit messages
	if strings.Contains(outputLower, "exited with code") || strings.Contains(outputLower, "exit code:") {
		result.Running = false
		result.Crashed = true
		result.Reason = "exit code detected"
		return result
	}

	// Check for bare shell prompt at end of output (agent crashed to shell)
	lines := splitLines(output)
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		// Detect common shell prompts: "$", "bash$", "zsh%", "user@host$", etc.
		// Be conservative - only match if line is very short (prompt-like)
		if len(lastLine) < 50 {
			if lastLine == "$" || lastLine == "%" ||
				strings.HasSuffix(lastLine, " $") || strings.HasSuffix(lastLine, " %") ||
				strings.HasSuffix(lastLine, "$ ") || strings.HasSuffix(lastLine, "% ") {
				result.Running = false
				result.Crashed = true
				result.ExitStatus = "shell prompt"
				result.Reason = "detected shell prompt - agent may have crashed"
			}
		}
	}

	return result
}

// checkStallWithActivity uses the StateClassifier for stall detection
func checkStallWithActivity(paneID string, agentType string) *StallCheckResult {
	result := &StallCheckResult{
		Stalled:    false,
		Confidence: 0.5,
	}

	// Create a classifier for this pane
	classifier := NewStateClassifier(paneID, &ClassifierConfig{
		AgentType:      agentType,
		StallThreshold: DefaultStallThreshold,
	})

	// Classify the current state
	activity, err := classifier.Classify()
	if err != nil {
		result.Reason = "failed to classify activity: " + err.Error()
		return result
	}

	// Extract activity state
	result.ActivityState = string(activity.State)
	result.Velocity = activity.Velocity
	result.Confidence = activity.Confidence

	// Calculate idle time from StateSince for idle-like states
	if !activity.StateSince.IsZero() {
		result.IdleSeconds = int(time.Since(activity.StateSince).Seconds())
	}

	// Check for stall conditions
	switch activity.State {
	case StateStalled:
		result.Stalled = true
		result.Reason = "agent stalled - no output for extended period"
	case StateError:
		result.Stalled = true
		result.Reason = "agent in error state"
	case StateUnknown:
		// Unknown might indicate a stall if velocity is 0
		if activity.Velocity == 0 && result.IdleSeconds > int(DefaultStallThreshold.Seconds()) {
			result.Stalled = true
			result.Reason = "unknown state with no output"
		}
	}

	return result
}

// checkErrors detects error patterns in pane output
func checkErrors(paneID string) *ErrorCheckResult {
	result := &ErrorCheckResult{
		HasErrors:   false,
		RateLimited: false,
		Patterns:    []string{},
	}

	// Capture pane output
	output, err := tmux.CapturePaneOutput(paneID, 50)
	if err != nil {
		result.Reason = "failed to capture pane output"
		return result
	}

	output = stripANSI(output)
	outputLower := strings.ToLower(output)

	// Check for error patterns
	seenPatterns := make(map[string]bool)
	for _, ep := range healthErrorPatterns {
		if strings.Contains(outputLower, strings.ToLower(ep.Pattern)) {
			if !seenPatterns[ep.Type] {
				result.Patterns = append(result.Patterns, ep.Type)
				seenPatterns[ep.Type] = true

				if ep.Type == "rate_limit" {
					result.RateLimited = true
					result.WaitSeconds = parseRateLimitWait(output)
				}

				if ep.Type == "crash" || ep.Type == "auth_error" || ep.Type == "network_error" {
					result.HasErrors = true
				}
			}
		}
	}

	if result.RateLimited {
		result.HasErrors = true
		result.Reason = "rate limit detected"
	} else if len(result.Patterns) > 0 {
		result.Reason = fmt.Sprintf("detected: %v", result.Patterns)
	}

	return result
}

// parseRateLimitWait extracts wait time from rate limit messages
func parseRateLimitWait(output string) int {
	outputLower := strings.ToLower(output)

	// Look for common wait time indicators and extract the number
	// Patterns: "wait 60 seconds", "retry in 30s", "try again in 60s", "60 second cooldown"
	indicators := []string{
		"wait ", "retry in ", "retry after ",
		"try again in ", " second", " sec", "cooldown", "delay",
	}

	// Find any indicator and look for nearby numbers
	for _, ind := range indicators {
		idx := strings.Index(outputLower, ind)
		if idx < 0 {
			continue
		}

		// Search around the indicator for a number (before or after)
		searchStart := idx - 10
		if searchStart < 0 {
			searchStart = 0
		}
		searchEnd := idx + len(ind) + 10
		if searchEnd > len(outputLower) {
			searchEnd = len(outputLower)
		}

		region := outputLower[searchStart:searchEnd]

		// Extract numbers from the region
		var num int
		for i := 0; i < len(region); i++ {
			if region[i] >= '0' && region[i] <= '9' {
				fmt.Sscanf(region[i:], "%d", &num)
				if num > 0 && num <= 3600 { // Reasonable wait time (1 sec to 1 hour)
					return num
				}
				// Skip past this number
				for i < len(region) && region[i] >= '0' && region[i] <= '9' {
					i++
				}
			}
		}
	}
	return 0
}

// calculateHealthState determines the overall health state from all checks
func calculateHealthState(check *HealthCheck) (HealthState, string) {
	// Priority order: unhealthy > rate_limited > degraded > healthy

	// Check for crash (unhealthy)
	if check.ProcessCheck != nil && check.ProcessCheck.Crashed {
		return HealthUnhealthy, "agent crashed"
	}

	// Check for error state (unhealthy)
	if check.ErrorCheck != nil && check.ErrorCheck.HasErrors && !check.ErrorCheck.RateLimited {
		return HealthUnhealthy, "error detected: " + check.ErrorCheck.Reason
	}

	// Check for rate limit
	if check.ErrorCheck != nil && check.ErrorCheck.RateLimited {
		return HealthRateLimited, "rate limit detected"
	}

	// Check for stall (degraded)
	if check.StallCheck != nil && check.StallCheck.Stalled {
		return HealthDegraded, "agent stalled: " + check.StallCheck.Reason
	}

	// Check for low velocity (degraded)
	if check.StallCheck != nil && check.StallCheck.IdleSeconds > 300 { // 5 minutes
		return HealthDegraded, "agent idle for extended period"
	}

	return HealthHealthy, "all checks passed"
}

// calculateHealthConfidence determines confidence in the health assessment
func calculateHealthConfidence(check *HealthCheck) float64 {
	confidence := 1.0

	// Lower confidence if stall check has low confidence
	if check.StallCheck != nil && check.StallCheck.Confidence < 0.7 {
		confidence *= check.StallCheck.Confidence
	}

	// Lower confidence if we couldn't perform all checks
	if check.ProcessCheck == nil || check.StallCheck == nil || check.ErrorCheck == nil {
		confidence *= 0.8
	}

	return confidence
}

// =============================================================================
// Session Health API (--robot-health=SESSION)
// =============================================================================

// SessionHealthOutput is the response format for --robot-health=SESSION
type SessionHealthOutput struct {
	Success   bool                       `json:"success"`
	Session   string                     `json:"session"`
	CheckedAt time.Time                  `json:"checked_at"`
	Agents    []SessionAgentHealth       `json:"agents"`
	Summary   SessionHealthSummary       `json:"summary"`
	Error     string                     `json:"error,omitempty"`
}

// SessionAgentHealth contains health metrics for a single agent pane
type SessionAgentHealth struct {
	Pane             int     `json:"pane"`
	AgentType        string  `json:"agent_type"`
	Health           string  `json:"health"` // healthy, degraded, unhealthy, rate_limited
	UptimeSeconds    int     `json:"uptime_seconds"`
	Restarts         int     `json:"restarts"`
	LastError        string  `json:"last_error,omitempty"`
	RateLimitCount   int     `json:"rate_limit_count"`
	BackoffRemaining int     `json:"backoff_remaining"` // seconds until ready
	Confidence       float64 `json:"confidence"`
}

// SessionHealthSummary contains aggregate health counts
type SessionHealthSummary struct {
	Total       int `json:"total"`
	Healthy     int `json:"healthy"`
	Degraded    int `json:"degraded"`
	Unhealthy   int `json:"unhealthy"`
	RateLimited int `json:"rate_limited"`
}

// PrintSessionHealth outputs per-agent health for a specific session
func PrintSessionHealth(session string) error {
	output := SessionHealthOutput{
		Success:   true,
		Session:   session,
		CheckedAt: time.Now().UTC(),
		Agents:    []SessionAgentHealth{},
		Summary:   SessionHealthSummary{},
	}

	// Check if session exists
	if !tmux.SessionExists(session) {
		output.Success = false
		output.Error = fmt.Sprintf("session '%s' not found", session)
		return encodeJSON(output)
	}

	// Get panes in the session
	panes, err := tmux.GetPanes(session)
	if err != nil {
		output.Success = false
		output.Error = fmt.Sprintf("failed to get panes: %v", err)
		return encodeJSON(output)
	}

	// Check health for each pane
	for _, pane := range panes {
		agentType := detectAgentTypeFromPane(pane)
		if agentType == "user" || agentType == "unknown" {
			continue // Skip non-agent panes
		}

		agentHealth := SessionAgentHealth{
			Pane:      pane.Index,
			AgentType: agentType,
			Health:    "healthy",
		}

		// Get activity time for uptime calculation
		activityTime, err := tmux.GetPaneActivity(pane.ID)
		if err == nil {
			agentHealth.UptimeSeconds = int(time.Since(activityTime).Seconds())
		}

		// Perform comprehensive health check
		check, err := CheckAgentHealthWithActivity(pane.ID, agentType)
		if err == nil {
			agentHealth.Confidence = check.Confidence

			// Map health state to string
			switch check.HealthState {
			case HealthHealthy:
				agentHealth.Health = "healthy"
			case HealthDegraded:
				agentHealth.Health = "degraded"
			case HealthUnhealthy:
				agentHealth.Health = "unhealthy"
				agentHealth.LastError = check.Reason
			case HealthRateLimited:
				agentHealth.Health = "rate_limited"
				agentHealth.RateLimitCount = 1
				if check.ErrorCheck != nil && check.ErrorCheck.WaitSeconds > 0 {
					agentHealth.BackoffRemaining = check.ErrorCheck.WaitSeconds
				}
			}

			// Track errors
			if check.ProcessCheck != nil && check.ProcessCheck.Crashed {
				agentHealth.Restarts++ // Track as a restart indicator
				agentHealth.LastError = check.ProcessCheck.Reason
			}
		}

		output.Agents = append(output.Agents, agentHealth)

		// Update summary
		output.Summary.Total++
		switch agentHealth.Health {
		case "healthy":
			output.Summary.Healthy++
		case "degraded":
			output.Summary.Degraded++
		case "unhealthy":
			output.Summary.Unhealthy++
		case "rate_limited":
			output.Summary.RateLimited++
		}
	}

	return encodeJSON(output)
}

// detectAgentTypeFromPane determines the agent type from pane information
func detectAgentTypeFromPane(pane tmux.Pane) string {
	switch pane.Type {
	case tmux.AgentClaude:
		return "claude"
	case tmux.AgentCodex:
		return "codex"
	case tmux.AgentGemini:
		return "gemini"
	case tmux.AgentUser:
		return "user"
	default:
		return "unknown"
	}
}
