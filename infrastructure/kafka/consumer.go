package kafka

import (
	"encoding/json"
	"go-transcoder/service"
	"log"

	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type consumerService struct {
	transcoder service.TranscodeService
}

type Consumer interface {
	RunWorker()
}

func NewConsumer(transcoder service.TranscodeService) Consumer {
	return &consumerService{transcoder: transcoder}
}

func (c *consumerService) RunWorker() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  "localhost:9092",
		"group.id":           "transcoder-group",
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
		
	})

	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{"transcoding-jobs"}, nil); err != nil {
		log.Fatalf("Failed to subscribe to topics: %s", err)
	}

	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			slog.Error("Consumer error", "error", err, "message", msg)
			continue
		}

		var job TranscodeJob
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			continue
		}

		slog.Info(">>> Processing Job", "VideoName", job.VideoName, "FilePath", job.FilePath)
		targets := filterResolutions(job.MaxHeight)

		results, err := c.transcoder.StartTranscoding(job.FilePath, job.VideoName, targets, job.Duration)
		if err == nil {
			if err := c.transcoder.GenerateMasterPlaylist(job.VideoName, results); err == nil {
				slog.Info("SUCCESS: Finished", "VideoName", job.VideoName)
			}

			_, err := consumer.CommitMessage(msg)
			if err != nil {
				slog.Error("Failed to commit message", "error", err)
			}

			slog.Info("Successfully processed job", "VideoName", job.VideoName)
		} else {
			slog.Error("Transcoding failed", "VideoName", job.VideoName, "error", err)
		}
	}
}
