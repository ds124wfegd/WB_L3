package processor

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/ds124wfegd/WB_L3/4/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResizeOperation тестирует операцию изменения размера
func TestResizeOperation(t *testing.T) {
	tests := []struct {
		name           string
		originalWidth  int
		originalHeight int
		targetWidth    int
		targetHeight   int
	}{
		{
			name:           "resize to smaller dimensions",
			originalWidth:  800,
			originalHeight: 600,
			targetWidth:    400,
			targetHeight:   300,
		},
		{
			name:           "resize to larger dimensions",
			originalWidth:  200,
			originalHeight: 150,
			targetWidth:    400,
			targetHeight:   300,
		},
		{
			name:           "resize to square",
			originalWidth:  800,
			originalHeight: 600,
			targetWidth:    200,
			targetHeight:   200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовое изображение
			original := image.NewRGBA(image.Rect(0, 0, tt.originalWidth, tt.originalHeight))
			fillImageWithColor(original, color.RGBA{R: 100, G: 150, B: 200, A: 255})

			// Выполняем операцию ресайза
			resized := imaging.Resize(original, tt.targetWidth, tt.targetHeight, imaging.Lanczos)

			// Проверяем результаты
			require.NotNil(t, resized)
			assert.Equal(t, tt.targetWidth, resized.Bounds().Dx())
			assert.Equal(t, tt.targetHeight, resized.Bounds().Dy())
		})
	}
}

// TestThumbnailOperation тестирует операцию генерации миниатюр
func TestThumbnailOperation(t *testing.T) {
	tests := []struct {
		name           string
		originalWidth  int
		originalHeight int
		maxWidth       int
		maxHeight      int
	}{
		{
			name:           "landscape thumbnail",
			originalWidth:  800,
			originalHeight: 600,
			maxWidth:       100,
			maxHeight:      100,
		},
		{
			name:           "portrait thumbnail",
			originalWidth:  600,
			originalHeight: 800,
			maxWidth:       100,
			maxHeight:      100,
		},
		{
			name:           "square thumbnail",
			originalWidth:  500,
			originalHeight: 500,
			maxWidth:       150,
			maxHeight:      150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовое изображение
			original := image.NewRGBA(image.Rect(0, 0, tt.originalWidth, tt.originalHeight))
			fillImageWithColor(original, color.RGBA{R: 50, G: 100, B: 150, A: 255})

			// Генерируем миниатюру
			thumbnail := imaging.Thumbnail(original, tt.maxWidth, tt.maxHeight, imaging.Lanczos)

			// Проверяем, что миниатюра не превышает максимальные размеры
			require.NotNil(t, thumbnail)
			assert.True(t, thumbnail.Bounds().Dx() <= tt.maxWidth)
			assert.True(t, thumbnail.Bounds().Dy() <= tt.maxHeight)
		})
	}
}

// TestWatermarkOperation тестирует операцию добавления водяных знаков
func TestWatermarkOperation(t *testing.T) {
	processor := &imageProcessor{storagePath: "./test_storage"}

	tests := []struct {
		name          string
		imageWidth    int
		imageHeight   int
		watermarkText string
	}{
		{
			name:          "watermark on small image",
			imageWidth:    100,
			imageHeight:   100,
			watermarkText: "TEST",
		},
		{
			name:          "watermark on large image",
			imageWidth:    800,
			imageHeight:   600,
			watermarkText: "COPYRIGHT",
		},
		{
			name:          "watermark with empty text",
			imageWidth:    500,
			imageHeight:   500,
			watermarkText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовое изображение
			original := image.NewRGBA(image.Rect(0, 0, tt.imageWidth, tt.imageHeight))
			fillImageWithColor(original, color.RGBA{R: 200, G: 100, B: 50, A: 255})

			// Добавляем водяной знак
			watermarked := processor.addWatermark(original, tt.watermarkText)

			// Проверяем результаты
			require.NotNil(t, watermarked)
			assert.Equal(t, tt.imageWidth, watermarked.Bounds().Dx())
			assert.Equal(t, tt.imageHeight, watermarked.Bounds().Dy())
		})
	}
}

// TestMultipleResizeOperations тестирует последовательное выполнение операций ресайза
func TestMultipleResizeOperations(t *testing.T) {
	tests := []struct {
		name       string
		operations []entity.Operation
	}{
		{
			name: "multiple resize operations",
			operations: []entity.Operation{
				{Type: "resize", Width: 800, Height: 600},
				{Type: "resize", Width: 400, Height: 300},
			},
		},
		{
			name: "resize then thumbnail",
			operations: []entity.Operation{
				{Type: "resize", Width: 1024, Height: 768},
				{Type: "thumbnail", Width: 100, Height: 100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём оригинальное изображение
			original := image.NewRGBA(image.Rect(0, 0, 2000, 1500))
			if original == nil {
				t.Fatal("не удалось создать исходное изображение")
			}
			fillImageWithColor(original, color.RGBA{R: 150, G: 200, B: 100, A: 255})

			currentImage := original

			// Последовательно применяем операции
			for _, op := range tt.operations {
				var processed image.Image

				switch op.Type {
				case "resize":
					processed = imaging.Resize(currentImage, op.Width, op.Height, imaging.Lanczos)
				case "thumbnail":
					processed = imaging.Thumbnail(currentImage, op.Width, op.Height, imaging.Lanczos)
				}

				// Проверяем, что операция не вернула nil
				if processed == nil {
					t.Errorf("операция %q вернула nil для изображения", op.Type)
					return
				}

				// Приводим тип с проверкой успешности
				currentImage = convertToRGBA(processed)
			}

			// Финальная проверка
			assert.NotNil(t, currentImage)
		})
	}
}

// TestWatermarkWithConvertedImage тестирует водяные знаки на преобразованных изображениях
func TestWatermarkWithConvertedImage(t *testing.T) {
	processor := &imageProcessor{storagePath: "./test_storage"}

	tests := []struct {
		name          string
		imageWidth    int
		imageHeight   int
		watermarkText string
	}{
		{
			name:          "watermark on resized image",
			imageWidth:    800,
			imageHeight:   600,
			watermarkText: "RESIZED",
		},
		{
			name:          "watermark on thumbnail image",
			imageWidth:    200,
			imageHeight:   200,
			watermarkText: "THUMBNAIL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем оригинальное изображение
			original := image.NewRGBA(image.Rect(0, 0, tt.imageWidth, tt.imageHeight))
			fillImageWithColor(original, color.RGBA{R: 100, G: 150, B: 200, A: 255})

			// Преобразуем изображение (имитируем результат операции)
			var processed image.Image
			if tt.name == "watermark on resized image" {
				processed = imaging.Resize(original, 400, 300, imaging.Lanczos)
			} else {
				processed = imaging.Thumbnail(original, 100, 100, imaging.Lanczos)
			}

			// Преобразуем обратно в *image.RGBA для watermark
			rgba := convertToRGBA(processed)

			// Добавляем водяной знак
			watermarked := processor.addWatermark(rgba, tt.watermarkText)

			// Проверяем результаты
			require.NotNil(t, watermarked)
			assert.NotNil(t, watermarked)
		})
	}
}

// TestEdgeCases тестирует граничные случаи
func TestEdgeCases(t *testing.T) {
	processor := &imageProcessor{storagePath: "./test_storage"}

	tests := []struct {
		name        string
		operation   entity.Operation
		imageWidth  int
		imageHeight int
	}{
		{
			name: "resize very small image",
			operation: entity.Operation{
				Type:   "resize",
				Width:  100,
				Height: 100,
			},
			imageWidth:  10,
			imageHeight: 10,
		},
		{
			name: "thumbnail from large image",
			operation: entity.Operation{
				Type:   "thumbnail",
				Width:  50,
				Height: 50,
			},
			imageWidth:  2000,
			imageHeight: 1500,
		},
		{
			name: "watermark on single pixel",
			operation: entity.Operation{
				Type: "watermark",
				Text: "TEST",
			},
			imageWidth:  1,
			imageHeight: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := image.NewRGBA(image.Rect(0, 0, tt.imageWidth, tt.imageHeight))
			fillImageWithColor(original, color.RGBA{R: 100, G: 100, B: 100, A: 255})

			var processed image.Image

			switch tt.operation.Type {
			case "resize":
				processed = imaging.Resize(original, tt.operation.Width, tt.operation.Height, imaging.Lanczos)
			case "thumbnail":
				processed = imaging.Thumbnail(original, tt.operation.Width, tt.operation.Height, imaging.Lanczos)
			case "watermark":
				processed = processor.addWatermark(original, tt.operation.Text)
			}

			assert.NotNil(t, processed)
		})
	}
}

// TestOperationTypes тестирует разные типы операций
func TestOperationTypes(t *testing.T) {
	processor := &imageProcessor{storagePath: "./test_storage"}

	tests := []struct {
		name      string
		operation entity.Operation
		check     func(*testing.T, image.Image)
	}{
		{
			name: "resize operation",
			operation: entity.Operation{
				Type:   "resize",
				Width:  300,
				Height: 300,
			},
			check: func(t *testing.T, img image.Image) {
				assert.Equal(t, 300, img.Bounds().Dx())
				assert.Equal(t, 300, img.Bounds().Dy())
			},
		},
		{
			name: "thumbnail operation",
			operation: entity.Operation{
				Type:   "thumbnail",
				Width:  100,
				Height: 100,
			},
			check: func(t *testing.T, img image.Image) {
				assert.True(t, img.Bounds().Dx() <= 100)
				assert.True(t, img.Bounds().Dy() <= 100)
			},
		},
		{
			name: "watermark operation",
			operation: entity.Operation{
				Type: "watermark",
				Text: "WATERMARK",
			},
			check: func(t *testing.T, img image.Image) {
				assert.Equal(t, 500, img.Bounds().Dx())
				assert.Equal(t, 500, img.Bounds().Dy())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовое изображение для каждого теста
			original := image.NewRGBA(image.Rect(0, 0, 500, 500))
			fillImageWithColor(original, color.RGBA{R: 255, G: 255, B: 255, A: 255})

			var result image.Image

			switch tt.operation.Type {
			case "resize":
				result = imaging.Resize(original, tt.operation.Width, tt.operation.Height, imaging.Lanczos)
			case "thumbnail":
				result = imaging.Thumbnail(original, tt.operation.Width, tt.operation.Height, imaging.Lanczos)
			case "watermark":
				result = processor.addWatermark(original, tt.operation.Text)
			}

			require.NotNil(t, result)
			tt.check(t, result)
		})
	}
}

// fillImageWithColor заполняет изображение одним цветом
func fillImageWithColor(img *image.RGBA, color color.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, y, color)
		}
	}
}

// convertToRGBA преобразует image.Image в *image.RGBA
func convertToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}
