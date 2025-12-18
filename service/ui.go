package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Progress struct {
	Folder  string
	Percent float64
}

var mu sync.Mutex

var allProgress = make(map[string]float64)
var timeRegex = regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

type progressUI struct{}

type ProgressUIService interface {
	StartUI(ctx context.Context, folderCount int)
	GetDuration(inputPath string) (float64, error)
	MonitorProgress(folderName string, stderrPipe io.ReadCloser, totalDuration float64)
	TimeToSeconds(timeStr string) (float64, error)
}

func NewProgressUI() ProgressUIService {
	return &progressUI{}
}

// StartUI begins the progress UI that updates the progress bars periodically
func (p *progressUI) StartUI(ctx context.Context, folderCount int) {
	ticker := time.NewTicker(200 * time.Millisecond) // Smooth updates
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			drawBars(folderCount)
			return
		case <-ticker.C:
			drawBars(folderCount)
		}
	}
}

// GetDuration retrieves the total duration of the input video file using ffprobe
func (p *progressUI) GetDuration(inputPath string) (float64, error) {
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

// TimeToSeconds converts a time string in "HH:MM:SS.ss" format to total seconds as float64
func (p *progressUI) TimeToSeconds(timeStr string) (float64, error) {
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

// MonitorProgress reads ffmpeg stderr output to track progress for a specific folder
func (p *progressUI) MonitorProgress(folderName string, stderrPipe io.ReadCloser, totalDuration float64) {
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "time=") {
			timeStr := extractTime(line)
			currentSec, _ := p.TimeToSeconds(timeStr)

			mu.Lock()
			allProgress[folderName] = (currentSec / totalDuration) * 100
			mu.Unlock()
		}
	}
}
