package transcoder

import (
	"context"
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
	"1440p": 1440, // 2K / QHD
	"2160p": 2160, // 4K / UHD
}

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
	// We use -show_format as well, because sometimes bitrate is there instead of in the stream
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,bit_rate:format=bit_rate",
		"-of", "default=noprint_wrappers=1",
		segmentPath,
	}

	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe failed to read file: %v", err)
	}

	output := string(out)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "width=") {
			fmt.Sscanf(line, "width=%d", &width)
		}
		// Try to catch bitrate from either stream or format entry
		if strings.HasPrefix(line, "bit_rate=") {
			var tempBitrate int
			val := strings.TrimPrefix(line, "bit_rate=")
			if val != "N/A" {
				fmt.Sscanf(line, "bit_rate=%d", &tempBitrate)
				if tempBitrate > 0 {
					bitrate = tempBitrate
				}
			}
		}
	}

	// Fallback: If bitrate is still 0, HLS won't play well.
	// We provide a sensible estimate based on height if ffprobe fails.
	if bitrate == 0 {
		if width <= 640 {
			bitrate = 800000
		} // 360p
		if width <= 1280 {
			bitrate = 2500000
		} // 720p
		if width <= 1920 {
			bitrate = 5000000
		} // 1080p
		if width <= 2560 {
			bitrate = 14000000
		} // 1440p (2K)
		if width <= 3840 {
			bitrate = 30000000
		} // 2160p (4K)
	}

	return width, bitrate, nil
}

func CloseResultsChannel(results chan VariantInfo, cancel context.CancelFunc) {
	close(results)
	cancel()
}
