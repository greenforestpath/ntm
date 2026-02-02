package e2e

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/tools"
	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

type robotEnvelope struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
}

func requireBVVersionAtLeast(t *testing.T, min tools.Version) tools.Version {
	t.Helper()

	out, err := exec.Command("bv", "--version").CombinedOutput()
	if err != nil {
		t.Skipf("bv --version failed: %v", err)
	}
	versionText := strings.TrimSpace(string(out))
	if !tools.VersionRegex.MatchString(versionText) {
		t.Skipf("bv version not parseable: %q", versionText)
	}
	version, _ := tools.ParseStandardVersion(versionText)
	if !version.AtLeast(min) {
		t.Skipf("bv version %s < %s", version.String(), min.String())
	}
	return version
}

func assertRobotEnvelope(t *testing.T, payload []byte, label string) {
	t.Helper()

	var envelope robotEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("%s: invalid JSON: %v", label, err)
	}
	if envelope.Timestamp == "" {
		t.Fatalf("%s: missing timestamp field", label)
	}
	if !envelope.Success {
		t.Fatalf("%s: expected success=true", label)
	}
}

func TestRobotBVAnalysisCommands(t *testing.T) {
	testutil.RequireE2E(t)
	testutil.RequireNTMBinary(t)
	testutil.SkipIfBvUnavailable(t)

	requireBVVersionAtLeast(t, tools.Version{Major: 0, Minor: 31, Patch: 0})

	logger := testutil.NewTestLoggerStdout(t)
	filePath := "internal/robot/robot.go"

	cases := []struct {
		name string
		args []string
	}{
		{name: "robot-alerts", args: []string{"--robot-alerts"}},
		{name: "robot-graph", args: []string{"--robot-graph"}},
		{name: "robot-forecast", args: []string{"--robot-forecast=all"}},
		{name: "robot-suggest", args: []string{"--robot-suggest"}},
		{name: "robot-impact", args: []string{"--robot-impact=" + filePath}},
		{name: "robot-search", args: []string{"--robot-search=triage"}},
		{name: "robot-label-attention", args: []string{"--robot-label-attention"}},
		{name: "robot-label-flow", args: []string{"--robot-label-flow"}},
		{name: "robot-label-health", args: []string{"--robot-label-health"}},
		{name: "robot-file-beads", args: []string{"--robot-file-beads=" + filePath}},
		{name: "robot-file-hotspots", args: []string{"--robot-file-hotspots"}},
		{name: "robot-file-relations", args: []string{"--robot-file-relations=" + filePath}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger.LogSection(fmt.Sprintf("BV %s", tc.name))
			out := testutil.AssertCommandSuccess(t, logger, "ntm", tc.args...)
			assertRobotEnvelope(t, out, tc.name)
		})
	}
}
