package server

import (
	"go-transcoder/transcoder"
	"log"
	"net/http"
)

func Server() {
	mux := http.NewServeMux()

	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		var uploadHandler transcoder.FileUpload
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file from request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		uploadHandler.Filename = file

		if err := uploadHandler.ValidateFile(file, nil); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Proceed with transcoding...
		w.Write([]byte("File uploaded and validated successfully"))
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server is running on :8080")
	log.Fatal(server.ListenAndServe())
}
