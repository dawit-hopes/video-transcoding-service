package transcoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
)

func StartTranscoding(inputFile, videoName string, resolutions map[string]int, duration float64) (chan VariantInfo, error) {
	g, ctx := errgroup.WithContext(context.Background())
	sem := make(chan struct{}, 2)
	results := make(chan VariantInfo, len(resolutions))

	mu.Lock()
	for folder := range resolutions {
		allProgress[folder] = 0.0
		fmt.Println()
	}
	mu.Unlock()

	uiCtx, cancelUI := context.WithCancel(ctx)
	go StartUI(uiCtx, len(resolutions))

	for folder, height := range resolutions {
		folderName := folder
		targetHeight := height
		g.Go(func() error {

			sem <- struct{}{}
			defer func() { <-sem }()
			err := createDirectory(videoName, folderName)
			if err != nil {
				return fmt.Errorf("error creating directory for %s: %v", folderName, err)
			}

			args := getFFmpegArgs(
				inputFile,
				filepath.Join("output", videoName, folderName),
				targetHeight,
			)

			cmd := exec.CommandContext(ctx, "ffmpeg", args...)

			stdErr, err := cmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("failed to get stderr pipe: %v", err)
			}

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start ffmpeg for %s: %v", folderName, err)
			}

			go MonitorProgress(folderName, stdErr, duration)

			if err := cmd.Wait(); err != nil {
				return fmt.Errorf("ffmpeg failed for %s: %v", folderName, err)
			}

			time.Sleep(500 * time.Millisecond)
			pattern := filepath.Join("output", videoName, folderName, "*.ts")
			matches, err := filepath.Glob(pattern)
			if err != nil || len(matches) == 0 {
				return fmt.Errorf("metadata error: no segments found in %s (checked %s)", folderName, pattern)
			}

			width, bitrate, err := GetVariantMetadata(matches[0])
			if err != nil {
				return fmt.Errorf("failed to get variant metadata for %s: %v", folderName, err)
			}

			results <- VariantInfo{
				Height:     targetHeight,
				Width:      width,
				Bandwidth:  bitrate,
				FolderName: folderName,
			}

			return nil

		})

	}

	if err := g.Wait(); err != nil {
		cancelUI()
		return nil, err
	}

	CloseResultsChannel(results, cancelUI)

	return results, nil
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
