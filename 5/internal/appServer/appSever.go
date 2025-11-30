package appServer

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ds124wfegd/WB_L3/5/config"
	repository "github.com/ds124wfegd/WB_L3/5/internal/database/postgres"
	"github.com/ds124wfegd/WB_L3/5/internal/service"
	"github.com/ds124wfegd/WB_L3/5/internal/transport"
	"github.com/ds124wfegd/WB_L3/5/internal/worker"

	"github.com/ds124wfegd/WB_L3/5/pkg/postgres"
	"github.com/ds124wfegd/WB_L3/5/pkg/queue"
	"github.com/ds124wfegd/WB_L3/5/pkg/redis"
	"github.com/ds124wfegd/WB_L3/5/pkg/scheduler"
	"github.com/ds124wfegd/WB_L3/5/pkg/telegram"

	"github.com/gin-gonic/gin"
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

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
	// Initialize database
	db, err := postgres.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := postgres.RunMigrations(db); err != nil {
		logrus.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	eventRepo := repository.NewEventRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize Telegram bot
	var telegramBot *telegram.Bot
	if cfg.Telegram.BotToken != "" {
		telegramBot = telegram.NewBot(cfg.Telegram.BotToken)
		logrus.Info("Telegram bot initialized")
	} else {
		logrus.Warn("Telegram bot token not provided, notifications disabled")
	}

	var redisQueue queue.Queue
	var taskPublisher service.TaskPublisher

	if cfg.Redis.URL != "" {
		redisConfig := &queue.RedisQueueConfig{
			Addr:     cfg.Redis.URL,
			Password: "",
			DB:       0,
		}

		retryManager := queue.NewRetryManager(3, 5*time.Second)
		redisClient := redis.NewRedisClient(&cfg.Redis)
		defer redisClient.Close()
		dlqHandler := queue.NewDefaultDLQHandler(redisClient, "event_booking:dlq")

		redisQueue, err = queue.NewRedisQueue(redisConfig, retryManager, dlqHandler)
		if err != nil {
			logrus.Errorf("Failed to initialize Redis queue: %v. Continuing without queue...", err)
		} else {
			logrus.Info("Redis queue initialized")
			// Создаем адаптер для очереди
			taskPublisher = service.NewQueueAdapter(redisQueue)
		}
	}

	// Initialize services
	bookingService := service.NewBookingService(bookingRepo, eventRepo, userRepo, taskPublisher, telegramBot)
	eventService := service.NewEventService(eventRepo, bookingRepo)
	userService := service.NewUserService(userRepo, bookingRepo)

	// Initialize task handler if queue is available
	if redisQueue != nil {
		taskHandler := queue.NewTaskHandler(bookingService, eventService, userService, telegramBot)

		// Start queue consumer
		go func() {
			ctx := context.Background()
			if err := redisQueue.Subscribe(ctx, taskHandler.HandleTask); err != nil {
				logrus.Errorf("Queue subscriber error: %v", err)
			}
		}()
		logrus.Info("Queue subscriber started")
	}

	// Initialize and start scheduler
	expirationScheduler := scheduler.NewScheduler(bookingService, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go expirationScheduler.Start(ctx)
	logrus.Info("Expiration scheduler started")

	// Initialize cleanup worker
	cleanupWorker := worker.NewBookingCleanupWorker(bookingService, 30*time.Minute)
	go cleanupWorker.Start(ctx)
	logrus.Info("Cleanup worker started")

	// Initialize handlers
	eventHandler := transport.NewEventHandler(eventService)
	bookingHandler := transport.NewBookingHandler(bookingService)
	userHandler := transport.NewUserHandler(userService)

	// Setup HTTP server
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	go cleanupWorker.Start(ctx)

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	srv := new(Server)
	go func() {
		if err := srv.Run(cfg, transport.InitRoutes(eventHandler, bookingHandler, userHandler)); err != nil {
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
