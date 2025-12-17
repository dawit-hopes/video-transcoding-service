package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	cmd := exec.Command("ffmpeg", "-version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("error running ffmpeg: %v", err)
		return
	}

	fmt.Printf("FFmpeg says:\n%s\n", string(output))

	resolutions := []string{"360p", "720p", "1080p"}
	fileName := "sample_video"

	for _, res := range resolutions {
		err := createDirectory(fileName, res)
		if err != nil {
			fmt.Printf("error creating directory for %s: %v\n", res, err)
			return
		}
		fmt.Printf("Successfully created directory for %s\n", res)
	}
}

func createDirectory(fileName, resolution string) error {
	targetDir := filepath.Join("output", fileName, resolution)

	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %v", targetDir, err)
	}

	return nil
}

func getFFmpegArgs(inputFile, outputDir string, height int) []string {
	return []string{
		"-i", inputFile,
		"-vf", fmt.Sprintf("scale=-2:%d", height),
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(outputDir, "seg_%03d.ts"),
		filepath.Join(outputDir, "index.m3u8"),
	}
}
