package transport

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
	"github.com/ds124wfegd/WB_L3/5/internal/service"
	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	bookingService service.BookingService
}

func NewBookingHandler(bookingService service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

// SuccessResponse представляет успешный ответ
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// CancelBookingRequest представляет запрос на отмену бронирования
type CancelBookingRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

func (h *BookingHandler) BookSeats(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

	var req service.BookSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EventID = eventID

	booking, err := h.bookingService.BookSeats(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, booking)
}

func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	eventIDStr := c.Param("id")
	_, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

	var req struct {
		BookingID int64 `json:"booking_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.bookingService.ConfirmBooking(c.Request.Context(), req.BookingID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "booking confirmed"})
}

func (h *BookingHandler) GetUserBookings(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	bookings, err := h.bookingService.GetUserBookings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bookings)
}

// GetAllBookings возвращает все бронирования
func (h *BookingHandler) GetAllBookings(c *gin.Context) {
	// Получаем параметры пагинации
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	// Получаем фильтр по статусу
	status := c.Query("status")

	ctx := c.Request.Context()

	// Если указан статус, получаем бронирования по статусу
	if status != "" {
		bookingStatus, err := h.parseBookingStatus(status)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error:   "Invalid booking status",
			})
			return
		}

		bookings, err := h.bookingService.GetBookingsByStatus(ctx, bookingStatus)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error:   "Failed to get bookings by status: " + err.Error(),
			})
			return
		}

		// Применяем пагинацию
		start := offset
		if start > len(bookings) {
			start = len(bookings)
		}
		end := start + limit
		if end > len(bookings) {
			end = len(bookings)
		}

		paginatedBookings := bookings[start:end]

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Message: "Bookings retrieved successfully",
			Data:    paginatedBookings,
			Meta: map[string]interface{}{
				"total":    len(bookings),
				"limit":    limit,
				"offset":   offset,
				"has_more": end < len(bookings),
			},
		})
		return
	}

	// Если статус не указан, получаем все бронирования
	bookings, err := h.bookingService.GetAllBookings(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to get all bookings: " + err.Error(),
		})
		return
	}

	// Применяем пагинацию
	start := offset
	if start > len(bookings) {
		start = len(bookings)
	}
	end := start + limit
	if end > len(bookings) {
		end = len(bookings)
	}

	paginatedBookings := bookings[start:end]

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Bookings retrieved successfully",
		Data:    paginatedBookings,
		Meta: map[string]interface{}{
			"total":    len(bookings),
			"limit":    limit,
			"offset":   offset,
			"has_more": end < len(bookings),
		},
	})
}

// GetEventBookings возвращает все бронирования для конкретного мероприятия
func (h *BookingHandler) GetEventBookings(c *gin.Context) {
	// Получаем ID мероприятия из пути
	eventID, err := strconv.ParseInt(c.Param("event_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid event ID",
		})
		return
	}

	// Получаем параметры пагинации
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	// Получаем фильтр по статусу
	status := c.Query("status")

	ctx := c.Request.Context()

	// Получаем все бронирования мероприятия
	bookings, err := h.bookingService.GetEventBookings(ctx, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to get event bookings: " + err.Error(),
		})
		return
	}

	// Фильтруем по статусу если указан
	if status != "" {
		bookingStatus, err := h.parseBookingStatus(status)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error:   "Invalid booking status",
			})
			return
		}

		filteredBookings := make([]*entity.Booking, 0)
		for _, booking := range bookings {
			if booking.Status == bookingStatus {
				filteredBookings = append(filteredBookings, booking)
			}
		}
		bookings = filteredBookings
	}

	// Применяем пагинацию
	start := offset
	if start > len(bookings) {
		start = len(bookings)
	}
	end := start + limit
	if end > len(bookings) {
		end = len(bookings)
	}

	paginatedBookings := bookings[start:end]

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Event bookings retrieved successfully",
		Data:    paginatedBookings,
		Meta: map[string]interface{}{
			"event_id": eventID,
			"total":    len(bookings),
			"limit":    limit,
			"offset":   offset,
			"has_more": end < len(bookings),
		},
	})
}

// CancelBooking отменяет бронирование
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	// Получаем ID бронирования из пути
	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid booking ID",
		})
		return
	}

	// Парсим тело запроса
	var req CancelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Валидация причины отмены
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Cancellation reason is required",
		})
		return
	}

	if len(req.Reason) > 500 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Cancellation reason too long (max 500 characters)",
		})
		return
	}

	ctx := c.Request.Context()

	// Выполняем отмену бронирования
	err = h.bookingService.CancelBooking(ctx, bookingID, req.Reason)
	if err != nil {
		// Проверяем тип ошибки для возврата соответствующего статуса
		switch {
		case err.Error() == "booking not found":
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error:   "Booking not found",
			})
		case err.Error() == "booking already cancelled":
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error:   "Booking is already cancelled",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error:   "Failed to cancel booking: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Booking cancelled successfully",
		Meta: map[string]interface{}{
			"booking_id": bookingID,
			"reason":     req.Reason,
		},
	})
}

// parseBookingStatus парсит строку в статус бронирования
func (h *BookingHandler) parseBookingStatus(status string) (entity.BookingStatus, error) {
	switch status {
	case "pending":
		return entity.BookingStatusPending, nil
	case "confirmed":
		return entity.BookingStatusConfirmed, nil
	case "cancelled":
		return entity.BookingStatusCancelled, nil
	case "expired":
		return entity.BookingStatusExpired, nil
	default:
		return "", fmt.Errorf("invalid booking status: %s", status)
	}
}
