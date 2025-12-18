package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type VariantInfo struct {
	Height     int
	Width      int
	Bandwidth  int
	FolderName string
}

type TranscodeService interface {
	GenerateMasterPlaylist(videoName string, results chan VariantInfo) error
	StoreFile(file multipart.File, header *multipart.FileHeader) (string, error)
	GetVariantMetadata(segmentPath string) (width int, height int, bitrate int, err error)
	StartTranscoding(inputFile, videoName string, resolutions map[string]int, duration float64) (chan VariantInfo, error)
}

type transcodeService struct {
	progressUI ProgressUIService
}

func NewTranscodeService(progressUI ProgressUIService) TranscodeService {
	return &transcodeService{
		progressUI: progressUI,
	}
}

// GenerateMasterPlaylist creates the master playlist file for HLS streaming
func (s *transcodeService) GenerateMasterPlaylist(videoName string, results chan VariantInfo) error {
	masterPath := filepath.Join("output", videoName, "master.m3u8")
	f, err := os.Create(masterPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		slog.Error("Failed to write to master playlist", "error", err)
		return err
	}

	resultsSlice := make([]VariantInfo, 0)

	for v := range results {
		resultsSlice = append(resultsSlice, v)
	}

	resultsSlice = sortVariantsByHeight(resultsSlice)

	if len(resultsSlice) == 0 {
		slog.Error("No variant info available to generate master playlist")
		return fmt.Errorf("no variant info available to generate master playlist")
	}

	for _, variant := range resultsSlice {
		line1 := fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n",
			variant.Bandwidth, variant.Width, variant.Height)
		line2 := fmt.Sprintf("%s/index.m3u8\n", variant.FolderName)

		if _, err := f.WriteString(line1); err != nil {
			slog.Error("Failed to write to master playlist", "error", err)
			return err
		}
		if _, err := f.WriteString(line2); err != nil {
			slog.Error("Failed to write to master playlist", "error", err)
			return err
		}
	}

	return nil
}

// StoreFile saves the uploaded file to a temporary location and returns the file path
func (s *transcodeService) StoreFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", err
	}

	fileName := uuid.New().String() + filepath.Ext(header.Filename)
	filePath := filepath.Join("uploads", fileName)
	out, err := os.Create(filePath)
	if err != nil {
		slog.Error("Failed to create file", "error", err)
		return "", err
	}
	defer out.Close()

	_, err = file.Seek(0, 0)
	if err != nil {
		slog.Error("Failed to seek file", "error", err)
		return "", err
	}

	_, err = io.Copy(out, file)
	if err != nil {
		slog.Error("Failed to save file", "error", err)
		return "", err
	}

	return filePath, nil
}

// GetExactBitrate fetches the exact bitrate of a transcoded segment
func (s *transcodeService) GetExactBitrate(segmentPath string) (int, error) {
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
		slog.Error("ffprobe command failed", "error", err)
		return 0, fmt.Errorf("ffprobe failed: %v", err)
	}

	bitrateStr := strings.TrimSpace(string(out))
	bitrate, err := strconv.Atoi(bitrateStr)
	if err != nil {
		slog.Error("Failed to parse bitrate", "bitrateStr", bitrateStr, "error", err)
		return 0, fmt.Errorf("failed to parse bitrate '%s': %v", bitrateStr, err)
	}

	return bitrate, nil
}

// GetVariantMetadata fetches the actual width and bitrate of a transcoded segment
// Update the return signature to include height
func (s *transcodeService) GetVariantMetadata(segmentPath string) (width int, height int, bitrate int, err error) {
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,bit_rate:format=bit_rate", // Added height here
		"-of", "default=noprint_wrappers=1",
		segmentPath,
	}

	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		slog.Error("ffprobe command failed", "error", err)
		return 0, 0, 0, fmt.Errorf("ffprobe failed: %v", err)
	}

	output := string(out)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "width=") {
			fmt.Sscanf(line, "width=%d", &width)
		}
		if strings.HasPrefix(line, "height=") {
			fmt.Sscanf(line, "height=%d", &height) // Extract height
		}
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

	// Fallback logic using the newly extracted height
	if bitrate == 0 {
		if height <= 360 {
			bitrate = 800000
		} else if height <= 720 {
			bitrate = 2500000
		} else if height <= 1080 {
			bitrate = 5000000
		} else if height <= 1440 {
			bitrate = 14000000
		} else {
			bitrate = 30000000
		}
	}

	return width, height, bitrate, nil
}

// StartTranscoding initiates the transcoding process for the given input file
func (s *transcodeService) StartTranscoding(inputFile, videoName string, resolutions map[string]int, duration float64) (chan VariantInfo, error) {
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
	go s.progressUI.StartUI(uiCtx, len(resolutions))

	for folder, height := range resolutions {
		folderName := folder
		targetHeight := height
		g.Go(func() error {

			sem <- struct{}{}
			defer func() { <-sem }()
			err := createDirectory(videoName, folderName)
			if err != nil {
				slog.Error("Failed to create directory", "folderName", folderName, "error", err)
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
				slog.Error("Failed to get stderr pipe", "folderName", folderName, "error", err)
				return fmt.Errorf("failed to get stderr pipe: %v", err)
			}

			if err := cmd.Start(); err != nil {
				slog.Error("Failed to start ffmpeg", "folderName", folderName, "error", err)
				return fmt.Errorf("failed to start ffmpeg for %s: %v", folderName, err)
			}

			go s.progressUI.MonitorProgress(folderName, stdErr, duration)

			if err := cmd.Wait(); err != nil {
				slog.Error("ffmpeg command failed", "folderName", folderName, "error", err)
				return fmt.Errorf("ffmpeg failed for %s: %v", folderName, err)
			}

			time.Sleep(500 * time.Millisecond)
			pattern := filepath.Join("output", videoName, folderName, "*.ts")
			matches, err := filepath.Glob(pattern)
			if err != nil || len(matches) == 0 {
				slog.Error("No segments found after transcoding", "folderName", folderName, "pattern", pattern, "error", err)
				return fmt.Errorf("metadata error: no segments found in %s (checked %s)", folderName, pattern)
			}

			width, _, bitrate, err := s.GetVariantMetadata(matches[0])
			if err != nil {
				slog.Error("Failed to get variant metadata", "folderName", folderName, "error", err)
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

// createDirectory ensures the output directory for a given resolution exists
func createDirectory(fileName, resolution string) error {
	targetDir := filepath.Join("output", fileName, resolution)

	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		slog.Error("Failed to create directory", "targetDir", targetDir, "error", err)
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
