package cli

import (
	"fmt"
)

// colorize returns ANSI escape code for a lipgloss color
func colorize(c interface{}) string {
	return fmt.Sprintf("\033[38;2;%s", colorToRGB(c))
}

func colorToRGB(c interface{}) string {
	// Extract RGB from lipgloss color (hex format)
	s := fmt.Sprintf("%v", c)
	if len(s) == 7 && s[0] == '#' {
		r, g, b := hexToRGB(s)
		return fmt.Sprintf("%d;%d;%dm", r, g, b)
	}
	return "255;255;255m" // Default white
}

func hexToRGB(hex string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// formatBytes formats bytes in a human-readable way (e.g., "1.5 KB")
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatCleanupBytes is an alias for formatBytes used in cleanup operations
func formatCleanupBytes(b int64) string {
	return formatBytes(b)
}
