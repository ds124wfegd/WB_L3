package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/ds124wfegd/WB_L3/4/internal/entity"
	"github.com/segmentio/kafka-go"
)

type ImageProcessor interface {
	Process(task entity.ProcessingTask) error
}

type imageProcessor struct {
	storagePath string
}

func NewImageProcessor() ImageProcessor {
	return &imageProcessor{storagePath: "./storage"}
}

func (p *imageProcessor) Process(task entity.ProcessingTask) error {
	log.Printf("Processing image: %s", task.ImageID)

	// Загружаем оригинальное изображение
	originalPath := filepath.Join(p.storagePath, "original", task.ImageID)
	img, format, err := p.loadImage(originalPath)
	if err != nil {
		return fmt.Errorf("failed to load image: %v", err)
	}

	// Обрабатываем каждую операцию
	results := make(map[string]string)
	for _, op := range task.Operations {
		var processed image.Image
		var outputFormat string

		switch op.Type {
		case "resize":
			processed = imaging.Resize(img, op.Width, op.Height, imaging.Lanczos)
			outputFormat = "resized"
		case "thumbnail":
			processed = imaging.Thumbnail(img, op.Width, op.Height, imaging.Lanczos)
			outputFormat = "thumbnail"
		case "watermark":
			processed = p.addWatermark(img, op.Text)
			outputFormat = "watermark"
		default:
			log.Printf("Unknown operation: %s", op.Type)
			continue
		}

		// Сохраняем обработанное изображение
		outputPath := filepath.Join(p.storagePath, "processed", task.ImageID, outputFormat)
		if err := p.saveImage(processed, outputPath, format); err != nil {
			log.Printf("Failed to save %s: %v", outputFormat, err)
			continue
		}

		results[outputFormat] = outputPath
	}

	// Обновляем статус
	if err := p.updateStatus(task.ImageID, "completed", results); err != nil {
		return fmt.Errorf("failed to update status: %v", err)
	}

	log.Printf("Completed processing image: %s", task.ImageID)
	return nil
}

func (p *imageProcessor) loadImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	// Определяем формат по расширению
	ext := filepath.Ext(path)
	switch ext {
	case ".jpg", ".jpeg":
		img, err := jpeg.Decode(file)
		return img, "jpeg", err
	case ".png":
		img, err := png.Decode(file)
		return img, "png", err
	case ".gif":
		return p.processGif(path)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", ext)
	}
}

func (p *imageProcessor) processGif(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	// Декодируем GIF
	gifImg, err := gif.DecodeAll(file)
	if err != nil {
		return nil, "", err
	}

	// Возвращаем первый кадр
	if len(gifImg.Image) > 0 {
		return gifImg.Image[0], "gif", nil
	}

	return nil, "", fmt.Errorf("no frames in GIF")
}

func (p *imageProcessor) addWatermark(img image.Image, text string) image.Image {
	// Простая реализация водяного знака
	dst := imaging.Clone(img)
	// Здесь можно добавить более сложную логику наложения текста
	return dst
}

func (p *imageProcessor) updateStatus(imageID string, status string, formats map[string]string) error {
	metadataPath := filepath.Join(p.storagePath, "metadata", imageID+".json")

	file, err := os.OpenFile(metadataPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var imageData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&imageData); err != nil {
		return err
	}

	imageData["status"] = status
	imageData["formats"] = formats

	file.Seek(0, 0)
	file.Truncate(0)

	encoder := json.NewEncoder(file)
	return encoder.Encode(imageData)
}

func (p *imageProcessor) saveImage(img image.Image, path string, format string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(file, img)
	case "gif":
		// Для GIF сохраняем как PNG, так как обработка может изменить изображение
		return png.Encode(file, img)
	default:
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	}
}

func StartImageProcessorConsumer(brokers []string, topic, groupID string) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset, //-2 FirstOffset

	})

	defer reader.Close()

	processor := NewImageProcessor()

	log.Println("Image processor consumer started...")
	log.Printf("Connected to Kafka brokers: %s", brokers)

	for {
		ctx := context.Background()
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading message from Kafka: %v", err)
			continue
		}

		log.Printf("Received message from topic %s [partition %d, offset %d]: %s\n",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Value))

		var task entity.ProcessingTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			log.Printf("Failed to parse task: %v\n", err)
			continue
		}

		go func(t entity.ProcessingTask) {
			if err := processor.Process(t); err != nil {
				log.Printf("Processing failed for %s: %v\n", t.ImageID, err)
			} else {
				log.Printf("Successfully processed image: %s", t.ImageID)
			}
		}(task)
	}
}
