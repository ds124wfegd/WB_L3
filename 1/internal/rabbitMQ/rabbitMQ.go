package rabbitMQ

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Queue interface {
	Publish(ctx context.Context, message interface{}) error
	PublishWithDelay(ctx context.Context, message interface{}, delay time.Duration) error
	Consume(ctx context.Context, handler func(message []byte) error) error
	Close() error
}

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
	config  RabbitMQConfig
}

type RabbitMQConfig struct {
	URL          string
	QueueName    string
	ExchangeName string
	RetryCount   int
}

func NewRabbitMQ(config RabbitMQConfig) (*RabbitMQ, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Объявляем основную очередь
	q, err := channel.QueueDeclare(
		config.QueueName, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		amqp.Table{
			"x-queue-mode": "lazy",
		},
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	rabbitMQ := &RabbitMQ{
		conn:    conn,
		channel: channel,
		queue:   q,
		config:  config,
	}

	return rabbitMQ, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.PublishWithContext(
		ctx,
		"",           // exchange
		r.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (r *RabbitMQ) PublishWithDelay(ctx context.Context, message interface{}, delay time.Duration) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Используем подход с TTL и DLX (более надежный)
	return r.publishWithTTLAndDLX(ctx, body, delay)
}

func (r *RabbitMQ) publishWithTTLAndDLX(ctx context.Context, body []byte, delay time.Duration) error {
	// Создаем уникальное имя для временной очереди
	delayedQueueName := fmt.Sprintf("%s_delayed_%d", r.config.QueueName, time.Now().UnixNano())

	// Создаем очередь с TTL и DLX
	_, err := r.channel.QueueDeclare(
		delayedQueueName,
		true,  // durable
		false, // delete when unused
		true,  // exclusive (автоудаление при отключении потребителя)
		false, // no-wait
		amqp.Table{
			"x-message-ttl":             delay.Milliseconds(),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": r.config.QueueName,
			"x-expires":                 delay.Milliseconds() + 60000, // Удалить очередь через 1 минуту после TTL
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare delayed queue: %w", err)
	}

	// Публикуем сообщение в очередь с TTL
	err = r.channel.PublishWithContext(
		ctx,
		"",
		delayedQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)

	return err
}

func (r *RabbitMQ) Consume(ctx context.Context, handler func(message []byte) error) error {
	// Настраиваем QoS
	err := r.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Начинаем потребление сообщений
	msgs, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	go r.handleMessages(ctx, msgs, handler)
	return nil
}

func (r *RabbitMQ) handleMessages(ctx context.Context, msgs <-chan amqp.Delivery, handler func(message []byte) error) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			if err := handler(msg.Body); err != nil {
				fmt.Printf("Failed to process message: %v. Message will be retried.\n", err)
				msg.Nack(false, true) // requeue
			} else {
				msg.Ack(false)
			}
		}
	}
}

func (r *RabbitMQ) Close() error {
	var errs []error

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors while closing RabbitMQ: %v", errs)
	}

	return nil
}

// HealthCheck проверяет соединение с RabbitMQ
func (r *RabbitMQ) HealthCheck() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}

	testChannel, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("RabbitMQ health check failed: %w", err)
	}
	testChannel.Close()

	return nil
}
