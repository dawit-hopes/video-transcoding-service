package transcoder

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Progress struct {
	Folder  string
	Percent float64
}

var allProgress = make(map[string]float64)
var mu sync.Mutex

var timeRegex = regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

// extractTime searches a line of ffmpeg output for the timestamp
func extractTime(line string) string {
	match := timeRegex.FindStringSubmatch(line)
	if len(match) > 1 {
		return match[1] // Returns the "00:00:05.00" part
	}
	return ""
}

func GetDuration(inputPath string) (float64, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	}

	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v", err)
	}

	durationStr := strings.TrimSpace(string(out))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %v", durationStr, err)
	}

	return duration, nil
}

func TimeToSeconds(timeStr string) (float64, error) {
	// 1. Split "00:01:30.00" into ["00", "01", "30.00"]
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format")
	}

	// 2. Parse each part
	h, _ := strconv.ParseFloat(parts[0], 64)
	m, _ := strconv.ParseFloat(parts[1], 64)
	s, _ := strconv.ParseFloat(parts[2], 64)

	// 3. Sum it up: (h * 3600) + (m * 60) + s
	return h*3600 + m*60 + s, nil
}

func MonitorProgress(folderName string, stderrPipe io.ReadCloser, totalDuration float64) {
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "time=") {
			timeStr := extractTime(line)
			currentSec, _ := TimeToSeconds(timeStr)

			mu.Lock()
			allProgress[folderName] = (currentSec / totalDuration) * 100
			mu.Unlock()
		}
	}
}
