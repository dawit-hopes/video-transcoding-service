package transcoder

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type VariantInfo struct {
	Height     int
	Width      int
	Bandwidth  int
	FolderName string
}

var Resolutions = map[string]int{
	"360p":  360,
	"720p":  720,
	"1080p": 1080,
}

var Results = make(chan VariantInfo, len(Resolutions))

func GetExactBitrate(segmentPath string) (int, error) {
	// ffprobe flags:
	// -v error: Hide all logs except actual errors
	// -select_streams v:0: Look only at the first video stream
	// -show_entries stream=bit_rate: Specifically request the bitrate
	// -of default=noprint_wrappers=1:nokey=1: Output only the raw value
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=bit_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		segmentPath,
	}

	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v", err)
	}

	bitrateStr := strings.TrimSpace(string(out))
	bitrate, err := strconv.Atoi(bitrateStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse bitrate '%s': %v", bitrateStr, err)
	}

	return bitrate, nil
}

// GetVariantMetadata fetches the actual width and bitrate of a transcoded segment
func GetVariantMetadata(segmentPath string) (width int, bitrate int, err error) {
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,bit_rate",
		"-of", "csv=p=0:x=1",
		segmentPath,
	}

	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return 0, 0, err
	}

	// Split the CSV output
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output")
	}

	width, _ = strconv.Atoi(parts[0])
	bitrate, _ = strconv.Atoi(parts[1])

	return width, bitrate, nil
}

func CloseResultsChannel() {
	close(Results)
}
