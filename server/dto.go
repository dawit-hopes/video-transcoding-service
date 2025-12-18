package server

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	
)

type FileUpload struct {
	Filename multipart.File
}

func (f *FileUpload) ValidateFile(file multipart.File, header *multipart.FileHeader) error {
	const maxFileSize = 800 * 1024 * 1024 // 800MB
	if header.Size > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max 800MB)", header.Size)
	}

	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	contentType := http.DetectContentType(buffer)

	isVid := strings.HasPrefix(contentType, "video/")
	ext := strings.ToLower(filepath.Ext(header.Filename))
	isValidExt := ext == ".mov" || ext == ".mp4" || ext == ".mkv" || ext == ".avi"

	if !isVid && !isValidExt {
		return fmt.Errorf("invalid file type: %s (extension: %s)", contentType, ext)
	}

	return nil
}
