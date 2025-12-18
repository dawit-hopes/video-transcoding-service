package main

import (
	"flag"
	"go-transcoder/infrastructure/kafka"
	"go-transcoder/server"
	"go-transcoder/service"
	"log"
	"log/slog"

)

func main() {
	// 1. Define flags to choose mode
	// Usage: go run main.go -mode=api  OR  go run main.go -mode=worker
	mode := flag.String("mode", "all", "Mode to run the app in: api, worker, or all")
	flag.Parse()

	services := service.InitService()

	switch *mode {
	case "api":
		runAPI(services)
	case "worker":
		runWorker(services)
	case "all":
		slog.Info("Starting in 'all' mode (API + Worker)...")
		go runWorker(services)
		runAPI(services)
	default:
		log.Fatalf("Invalid mode: %s. Use 'api', 'worker', or 'all'", *mode)
	}
}

func runAPI(services *service.Service) {
	kafkaProducer := kafka.NewProducer(services.Transcode)
	s := server.NewServerService(services.Transcode, kafkaProducer, services.ProgressUI)

	slog.Info("Initializing API Server...")
	s.Server()
}

func runWorker(services *service.Service) {
	kafkaConsumer := kafka.NewConsumer(services.Transcode)

	slog.Info("Initializing Transcoder Worker...")
	kafkaConsumer.RunWorker()

	select {}
}
