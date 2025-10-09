.PHONY: build, run, docker-build, docker-run, up, down

# Сборка приложения
build:
	go build -o bin/image-processor-service ./cmd/app

# Запуск приложения
run:
	go run ./cmd/app

# Сборка Docker образа
docker-build:
	docker build -t app .

# Запуск Docker контейнера
docker-run:
	docker run -p 8081:8081 --env-file .env app

# Launching PostgreSQL, Kafka
up:
	docker-compose up -d

# Stopping PostgreSQL, Kafka
down:
	docker-compose down

# Check docker status
status:
	docker-compose ps

# logs
logs:
	docker-compose logs -f
