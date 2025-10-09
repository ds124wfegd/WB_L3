package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	SendMessage(topic string, message interface{}) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
}

func NewProducer(brokers string) Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers),
		Topic:        "image-processing",
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}

	log.Printf("Kafka producer configured for brokers: %s", brokers)

	// Проверяем подключение и создаем топик
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := kafka.DialContext(ctx, "tcp", brokers)
	if err != nil {
		log.Printf("Kafka connection failed: %v", err)
		log.Printf("Using mock producer instead")
		return &mockProducer{}
	}
	defer conn.Close()

	// Создаем топик если не существует
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             "image-processing",
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		log.Printf("Could not create topic (might already exist): %v", err)
	} else {
		log.Printf("Created topic: image-processing")
	}

	log.Printf("Connected to Kafka at %s", brokers)
	return &kafkaProducer{writer: writer}
}

func (p *kafkaProducer) SendMessage(topic string, message interface{}) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte("image-processor"),
		Value: messageBytes,
		Time:  time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("Failed to write message to Kafka: %v", err)
		return err
	}

	log.Printf("Message successfully sent to topic: %s", topic)
	return nil
}

func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}

// Mock producer для работы без Kafka
type mockProducer struct{}

func (m *mockProducer) SendMessage(topic string, message interface{}) error {
	log.Printf("MOCK: Message to topic %s: %v", topic, message)
	// Имитируем успешную обработку
	return nil
}

func (m *mockProducer) Close() error {
	return nil
}
