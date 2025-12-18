package service

import (
	"context"
	"fmt"
	"strings"
)

func sortVariantsByHeight(variants []VariantInfo) []VariantInfo {
	sorted := make([]VariantInfo, len(variants))
	copy(sorted, variants)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Height > sorted[j+1].Height {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	return sorted
}

func CloseResultsChannel(results chan VariantInfo, cancel context.CancelFunc) {
	close(results)
	cancel()
}

func extractTime(line string) string {
	match := timeRegex.FindStringSubmatch(line)
	if len(match) > 1 {
		return match[1] // Returns the "00:00:05.00" part
	}
	return ""
}

func drawBars(folderCount int) {
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("\033[%dA", folderCount)

	// 2. Print each bar
	for folder, pct := range allProgress {
		// Calculate how many '#' to show for a bar of length 20
		barLength := 20
		filled := min(int(pct/100*float64(barLength)), barLength)

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

		// \r moves to start, \033[K clears the rest of the line
		fmt.Printf("\r\033[K[%-7s] %s %.2f%%\n", folder, bar, pct)
	}
}
