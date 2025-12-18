package kafka

import (
	"encoding/json"
	"go-transcoder/service"
	"log"

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
		"bootstrap.servers": "localhost:9092",
		"group.id":          "transcoder-group",
		"auto.offset.reset": "earliest",
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
			log.Printf("Consumer error: %v (%v)\n", err, msg)
			continue
		}

		var job TranscodeJob
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			log.Printf("Failed to unmarshal message: %s", err)
			continue
		}

		log.Printf(">>> Processing Job: %s (File: %s)", job.VideoName, job.FilePath)

		targets := filterResolutions(job.MaxHeight)

		results, err := c.transcoder.StartTranscoding(job.FilePath, job.VideoName, targets, job.Duration)
		if err == nil {
			if err := c.transcoder.GenerateMasterPlaylist(job.VideoName, results); err == nil {
				log.Printf("SUCCESS: Finished %s", job.VideoName)
			}
		} else {
			log.Printf("ERROR: Transcoding failed for %s: %v", job.VideoName, err)
		}
	}
}
