package kafka

import (
	"go-transcoder/service"
	"log"
	"log/slog"


	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Producer struct {
	Producer          *kafka.Producer
	transcoderService service.TranscodeService
}

type ProducerInterface interface {
	Produce(topic string, key []byte, value []byte) error
}

func NewProducer(transcoderService service.TranscodeService) ProducerInterface {
	confluentProducer, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers": "localhost:9092",
			"security.protocol": "PLAINTEXT",
		})

	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}

	go func() {
		for e := range confluentProducer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					slog.Error("Delivery failed", "error", ev.TopicPartition.Error)
				}
			}
		}
	}()
	return &Producer{
		Producer:          confluentProducer,
		transcoderService: transcoderService,
	}
}

func (p *Producer) Produce(topic string, key []byte, value []byte) error {
	deliveryChan := make(chan kafka.Event)

	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            key,
		Value:          value,
	}, deliveryChan)

	e := <-deliveryChan
	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		slog.Error("Failed to deliver message", "error", m.TopicPartition.Error)
		return m.TopicPartition.Error
	}
	return err
}
