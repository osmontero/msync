package utils

import (
	"fmt"
)

// FormatBytes converts bytes to human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatDuration converts duration to human readable format
func FormatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}

	minutes := int(seconds / 60)
	secs := seconds - float64(minutes*60)

	if minutes < 60 {
		return fmt.Sprintf("%dm %.1fs", minutes, secs)
	}

	hours := minutes / 60
	minutes = minutes % 60

	return fmt.Sprintf("%dh %dm %.1fs", hours, minutes, secs)
}
