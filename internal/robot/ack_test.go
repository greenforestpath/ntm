// Package robot provides machine-readable output for AI agents.
// ack_test.go contains tests for the acknowledgment detection logic.
package robot

import (
	"testing"
)

func TestDetectAcknowledgment(t *testing.T) {
	tests := []struct {
		name           string
		initialOutput  string
		currentOutput  string
		message        string
		paneTitle      string
		expectedType   AckType
		expectedDetect bool
	}{
		{
			name:           "no change means no ack",
			initialOutput:  "some output\n",
			currentOutput:  "some output\n",
			message:        "test message",
			paneTitle:      "cc_1",
			expectedType:   AckNone,
			expectedDetect: false,
		},
		{
			name:           "explicit ack - understood",
			initialOutput:  "waiting...\n",
			currentOutput:  "waiting...\nunderstood, I'll work on that\n",
			message:        "fix the bug",
			paneTitle:      "cc_1",
			expectedType:   AckExplicitAck,
			expectedDetect: true,
		},
		{
			name:           "explicit ack - let me",
			initialOutput:  "> ",
			currentOutput:  "> \nLet me take a look at that file\n",
			message:        "check file",
			paneTitle:      "cc_1",
			expectedType:   AckExplicitAck,
			expectedDetect: true,
		},
		{
			name:           "explicit ack - working on",
			initialOutput:  "idle\n",
			currentOutput:  "idle\nWorking on the tests now\n",
			message:        "fix tests",
			paneTitle:      "cod_1",
			expectedType:   AckExplicitAck,
			expectedDetect: true,
		},
		{
			name:           "output started - multiple lines without ack phrase",
			initialOutput:  "prompt> \n",
			currentOutput:  "prompt> \nProcessing your request now\nChecking the files...\n",
			message:        "check something",
			paneTitle:      "cc_1",
			expectedType:   AckExplicitAck, // "processing" is in explicit ack phrases
			expectedDetect: true,
		},
		{
			name:           "echo detected with follow-up",
			initialOutput:  "> ",
			currentOutput:  "> fix the bug\nOkay, looking at the code...\n",
			message:        "fix the bug",
			paneTitle:      "cc_1",
			expectedType:   AckEchoDetected,
			expectedDetect: true,
		},
		{
			name:           "just echo with no follow-up - no ack yet",
			initialOutput:  "> ",
			currentOutput:  "> fix the bug",
			message:        "fix the bug",
			paneTitle:      "cc_1",
			expectedType:   AckNone,
			expectedDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ackType, detected := detectAcknowledgment(tt.initialOutput, tt.currentOutput, tt.message, tt.paneTitle)
			if detected != tt.expectedDetect {
				t.Errorf("detectAcknowledgment() detected = %v, want %v", detected, tt.expectedDetect)
			}
			if detected && ackType != tt.expectedType {
				t.Errorf("detectAcknowledgment() type = %v, want %v", ackType, tt.expectedType)
			}
		})
	}
}

func TestGetNewContent(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		current  string
		expected string
	}{
		{
			name:     "simple append",
			initial:  "hello",
			current:  "hello world",
			expected: " world",
		},
		{
			name:     "new lines",
			initial:  "line1\nline2",
			current:  "line1\nline2\nline3\nline4",
			expected: "\nline3\nline4",
		},
		{
			name:     "no change",
			initial:  "same",
			current:  "same",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNewContent(tt.initial, tt.current)
			if result != tt.expected {
				t.Errorf("getNewContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTruncateForMatch(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "short message",
			message:  "fix the bug",
			expected: "fix the bug",
		},
		{
			name:     "long message truncated",
			message:  "this is a very long message that should be truncated at 50 characters for matching purposes",
			expected: "this is a very long message that should be truncat", // 50 chars
		},
		{
			name:     "multiline takes first line",
			message:  "first line\nsecond line",
			expected: "first line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateForMatch(tt.message)
			if result != tt.expected {
				t.Errorf("truncateForMatch() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsIdlePrompt(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"> ", true},
		{"$ ", true},
		{"% ", true},
		{"# ", true},
		{"claude>", true},
		{"Claude>", true},
		{"codex>", true},
		{">>> ", true},
		{"some text", false},
		{"", false},
		{"working...", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isIdlePrompt(tt.line)
			if result != tt.expected {
				t.Errorf("isIdlePrompt(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestIsPromptLine(t *testing.T) {
	tests := []struct {
		line      string
		paneTitle string
		expected  bool
	}{
		{"user@host:~$ ", "", true},
		{"claude> ", "cc_1", true},
		{"> ", "", true},
		{">>> ", "", true},
		{"some output text", "", false},
		{"error: something failed", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isPromptLine(tt.line, tt.paneTitle)
			if result != tt.expected {
				t.Errorf("isPromptLine(%q, %q) = %v, want %v", tt.line, tt.paneTitle, result, tt.expected)
			}
		})
	}
}

func TestGetLastNonEmptyLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		n        int
		expected []string
	}{
		{
			name:     "simple case",
			content:  "line1\nline2\nline3\n",
			n:        2,
			expected: []string{"line3", "line2"},
		},
		{
			name:     "with empty lines",
			content:  "line1\n\nline2\n\n\nline3\n",
			n:        3,
			expected: []string{"line3", "line2", "line1"},
		},
		{
			name:     "fewer lines than requested",
			content:  "only one",
			n:        5,
			expected: []string{"only one"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLastNonEmptyLines(tt.content, tt.n)
			if len(result) != len(tt.expected) {
				t.Errorf("getLastNonEmptyLines() returned %d lines, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("getLastNonEmptyLines()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetContentAfterEcho(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		message  string
		expected string
	}{
		{
			name:     "with content after echo",
			content:  "fix the bug\nOkay, I'll fix it",
			message:  "fix the bug",
			expected: "Okay, I'll fix it",
		},
		{
			name:     "no content after echo",
			content:  "fix the bug",
			message:  "fix the bug",
			expected: "",
		},
		{
			name:     "message not found",
			content:  "some other text",
			message:  "fix the bug",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContentAfterEcho(tt.content, tt.message)
			if result != tt.expected {
				t.Errorf("getContentAfterEcho() = %q, want %q", result, tt.expected)
			}
		})
	}
}
