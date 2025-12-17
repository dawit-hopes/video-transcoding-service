package transcoder

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func StartUI(ctx context.Context, folderCount int) {
	ticker := time.NewTicker(200 * time.Millisecond) // Smooth updates
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final update before exiting
			drawBars(folderCount)
			return
		case <-ticker.C:
			drawBars(folderCount)
		}
	}
}

func drawBars(folderCount int) {
	mu.Lock()
	defer mu.Unlock()

	// 1. Move cursor UP to the start of the progress bars
	// \033[%dA where %d is the number of lines
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
