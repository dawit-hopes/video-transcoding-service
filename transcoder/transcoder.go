package transcoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

func StartTranscoding(inputFile, videoName string, resolutions map[string]int, duration float64) error {
	g, ctx := errgroup.WithContext(context.Background())
	for folder, height := range resolutions {
		folderName := folder
		targetHeight := height
		g.Go(func() error {

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

			width, bitrate, err := GetVariantMetadata(filepath.Join("output", videoName, folderName, "seg_000.ts"))
			if err != nil {
				return fmt.Errorf("failed to get variant metadata for %s: %v", folderName, err)
			}

			Results <- VariantInfo{
				Height:     targetHeight,
				Width:      width,
				Bandwidth:  bitrate,
				FolderName: folderName,
			}

			return nil

		})

	}

	if err := g.Wait(); err != nil {
		return err
	}

	CloseResultsChannel()
	return nil
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
