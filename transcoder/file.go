package transcoder

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type FileUpload struct {
	Filename multipart.File
}

func (f *FileUpload) ValidateFile(file multipart.File, header *multipart.FileHeader) error {
	// 1. Check Size
	const maxFileSize = 800 * 1024 * 1024 // 800MB
	if header.Size > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max 800MB)", header.Size)
	}

	// 2. Check Content Type (Sniffing)
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return err
	}
	file.Seek(0, 0)

	contentType := http.DetectContentType(buffer)

	// Check if it's a video OR if it has a common video extension as a fallback
	isVid := strings.HasPrefix(contentType, "video/")

	// Fallback: Check extension if sniffing is ambiguous
	ext := strings.ToLower(filepath.Ext(header.Filename))
	isValidExt := ext == ".mov" || ext == ".mp4" || ext == ".mkv" || ext == ".avi"

	if !isVid && !isValidExt {
		return fmt.Errorf("invalid file type: %s (extension: %s)", contentType, ext)
	}

	return nil
}

func GenerateMasterPlaylist(videoName string, results chan VariantInfo) error {
	masterPath := filepath.Join("output", videoName, "master.m3u8")
	f, err := os.Create(masterPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		return err
	}

	resultsSlice := make([]VariantInfo, 0)

	for v := range results {
		resultsSlice = append(resultsSlice, v)
	}

	resultsSlice = sortVariantsByHeight(resultsSlice)

	if len(resultsSlice) == 0 {
		return fmt.Errorf("no variant info available to generate master playlist")
	}

	for _, variant := range resultsSlice {
		line1 := fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n",
			variant.Bandwidth, variant.Width, variant.Height)
		line2 := fmt.Sprintf("%s/index.m3u8\n", variant.FolderName)

		if _, err := f.WriteString(line1); err != nil {
			return err
		}
		if _, err := f.WriteString(line2); err != nil {
			return err
		}
	}

	return nil
}

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

func StoreFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", err
	}

	fileName := uuid.New().String() + filepath.Ext(header.Filename)
	filePath := filepath.Join("uploads", fileName)
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = file.Seek(0, 0)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(out, file)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
