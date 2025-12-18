package server

import (
	"encoding/json"
	"fmt"
	"go-transcoder/infrastructure/kafka"
	"go-transcoder/service"
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

type ServerService struct {
	transcoder    service.TranscodeService
	kafkaProducer kafka.ProducerInterface
	uiService     service.ProgressUIService
}

type ServerServiceInterface interface {
	Server()
}

func NewServerService(transcoder service.TranscodeService, kafkaProducer kafka.ProducerInterface, uiService service.ProgressUIService) ServerServiceInterface {
	return &ServerService{
		transcoder:    transcoder,
		kafkaProducer: kafkaProducer,
		uiService:     uiService,
	}
}

func (s *ServerService) Server() {
	mux := http.NewServeMux()

	mux.Handle("/videos/", http.StripPrefix("/videos/", http.FileServer(http.Dir("output"))))

	// 2. Upload Endpoint
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		var uploadHandler FileUpload

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
		filePath, err := s.transcoder.StoreFile(file, header)
		if err != nil {
			http.Error(w, "Failed to store file", http.StatusInternalServerError)
			return
		}

		// Prepare metadata for response
		videoName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		playbackURL := fmt.Sprintf("/videos/%s/master.m3u8", videoName)

		go func() {

			duration, _ := s.uiService.GetDuration(filePath)
			_, originalHeight, _, _ := s.transcoder.GetVariantMetadata(filePath)

			job := kafka.TranscodeJob{
				FilePath:  filePath,
				VideoName: videoName,
				Duration:  duration,
				MaxHeight: originalHeight,
				VideoID:   videoName,
			}

			jobBytes, _ := json.Marshal(job)
			if err := s.kafkaProducer.Produce("transcoding-jobs", []byte(videoName), jobBytes); err != nil {
				log.Printf("Failed to produce Kafka message for %s: %v", videoName, err)
			} else {
				log.Printf("Enqueued transcoding job for %s", videoName)
			}

			log.Printf("Successfully finished transcoding: %s", videoName)
		}()

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
