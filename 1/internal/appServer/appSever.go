// launching the server, DB, kafka, postgres
package appServer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"

	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ds124wfegd/WB_L3/1/config"
	"github.com/ds124wfegd/WB_L3/1/internal/database"
	"github.com/ds124wfegd/WB_L3/1/internal/rabbitMQ"
	"github.com/ds124wfegd/WB_L3/1/internal/service"
	"github.com/ds124wfegd/WB_L3/1/internal/transport"
	"github.com/go-redis/redis/v8"

	"github.com/sirupsen/logrus"
)

type Server struct {
	httpServer *http.Server
}

func (s *Server) Run(cfg *config.Config, handler http.Handler) error {
	s.httpServer = &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           handler,
		MaxHeaderBytes:    1 << 20,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      cfg.Server.Timeout,
		IdleTimeout:       cfg.Server.Idle_timeout,
		ReadHeaderTimeout: 3 * time.Second,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},           // ban on outdate TLS certificate
		ErrorLog:          log.New(os.Stderr, "SERVER ERROR: ", log.LstdFlags), // os.Stderr can be replaced with ElsasticSearch in the feature
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func NewServer(cfg *config.Config) {

	logrus.SetFormatter(new(logrus.JSONFormatter))

	redisClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxRetries:   cfg.Redis.MaxRetries,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		PoolTimeout:  cfg.Redis.PoolTimeout,
		IdleTimeout:  cfg.Redis.IdleTimeout,
	})

	var rabbitMQURL string
	if cfg.Rabbit.URL != "" {
		rabbitMQURL = cfg.Rabbit.URL
	} else {
		rabbitMQURL = fmt.Sprintf("amqp://%s:%s@%s:%d/",
			cfg.Rabbit.Username,
			cfg.Rabbit.Password,
			cfg.Rabbit.Host,
			cfg.Rabbit.Port)
	}

	rabbitMQConfig := rabbitMQ.RabbitMQConfig{
		URL:          rabbitMQURL,
		QueueName:    cfg.Rabbit.QueueName,
		ExchangeName: cfg.Rabbit.ExchangeName,
		RetryCount:   3,
	}

	// Логирование для отладки
	fmt.Printf("Using RabbitMQ URL: %s\n", rabbitMQConfig.URL)

	rabbitMQ, err := rabbitMQ.NewRabbitMQ(rabbitMQConfig)
	if err != nil {
		logrus.Fatalf("Failed to connect to RabbitMQ:: %s", err.Error())
	}
	defer rabbitMQ.Close()

	notificationRepo := database.NewRedisRepository(redisClient)

	notificationUseCase := service.NewNotificationUseCase(notificationRepo, rabbitMQ, 3)

	ctx := context.Background()
	go startBackgroundProcessor(ctx, notificationUseCase)

	srv := new(Server)
	go func() {
		if err := srv.Run(cfg, transport.InitRoutes(notificationUseCase)); err != nil {
			logrus.Fatalf("error occured while running http server: %s", err.Error())
		}
	}()

	logrus.Print("App Started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Print("App Shutting Down")

	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Errorf("error occured on server shutting down: %s", err.Error())
	}

}

func startBackgroundProcessor(ctx context.Context, useCase service.NotificationUseCase) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := useCase.ProcessScheduledNotifications(ctx); err != nil {
				log.Printf("Error processing scheduled notifications: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
