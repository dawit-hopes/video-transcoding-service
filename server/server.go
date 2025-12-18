package server

import (
	"encoding/json"
	"fmt"
	"go-transcoder/transcoder"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Response structure for JSON communication
type UploadResponse struct {
	Message     string `json:"message"`
	VideoName   string `json:"video_name"`
	PlaybackURL string `json:"playback_url"`
}

func Server() {
	mux := http.NewServeMux()

	// 1. Static file server: Serves the 'output' folder via the '/videos/' URL
	// Example: http://localhost:8080/videos/myvideo/master.m3u8
	mux.Handle("/videos/", http.StripPrefix("/videos/", http.FileServer(http.Dir("output"))))

	// 2. Upload Endpoint
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		var uploadHandler transcoder.FileUpload

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limit upload size to 800MB
		if err := r.ParseMultipartForm(800 << 20); err != nil {
			http.Error(w, "Failed to parse multipart form (Max 800MB)", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file from request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file type and size
		if err := uploadHandler.ValidateFile(file, header); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Save the file to the local disk temporarily
		filePath, err := transcoder.StoreFile(file, header)
		if err != nil {
			http.Error(w, "Failed to store file", http.StatusInternalServerError)
			return
		}

		// Prepare metadata for response
		videoName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		playbackURL := fmt.Sprintf("/videos/%s/master.m3u8", videoName)

		// ASYNC BLOCK: Transcoding happens in the background
		go func() {
			defer os.Remove(filePath) // Clean up the original uploaded file when done

			duration, _ := transcoder.GetDuration(filePath)
			_, originalHeight, _ := transcoder.GetVariantMetadata(filePath)

			filteredResolutions := make(map[string]int)
			for folder, height := range transcoder.Resolutions {
				if height <= originalHeight {
					filteredResolutions[folder] = height
				} else {
					log.Printf("Skipping %s (%dp) as it exceeds original height (%dp)", folder, height, originalHeight)
				}
			}

			if len(filteredResolutions) == 0 {
				filteredResolutions["original"] = originalHeight
			}
			log.Printf("Starting background transcode for: %s", videoName)

			resultChan, err := transcoder.StartTranscoding(filePath, videoName, filteredResolutions, duration)
			if err != nil {
				log.Printf("Transcoding error for %s: %v", videoName, err)
				return
			}

			if err := transcoder.GenerateMasterPlaylist(videoName, resultChan); err != nil {
				log.Printf("Playlist error for %s: %v", videoName, err)
				return
			}
			log.Printf("Successfully finished transcoding: %s", videoName)
		}()

		// Respond to client immediately with JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)

		resp := UploadResponse{
			Message:     "Video accepted and processing started.",
			VideoName:   videoName,
			PlaybackURL: playbackURL,
		}
		json.NewEncoder(w).Encode(resp)
	})

	// 3. List Endpoint: Shows all processed videos
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir("output")
		if err != nil {
			http.Error(w, "Could not read output directory", http.StatusInternalServerError)
			return
		}

		var videos []string
		for _, e := range entries {
			if e.IsDir() {
				videos = append(videos, e.Name())
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(videos)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server is running on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
