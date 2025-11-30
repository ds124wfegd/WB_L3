package service

import (
	"context"
	"fmt"
	"log"
	"time"

	repository "github.com/ds124wfegd/WB_L3/5/internal/database/postgres"
	"github.com/ds124wfegd/WB_L3/5/internal/entity"
	"github.com/ds124wfegd/WB_L3/5/pkg/telegram"
)

// BookSeatsRequest представляет данные для бронирования мест
type BookSeatsRequest struct {
	EventID            int64 `json:"event_id" binding:"required"`
	UserID             int64 `json:"user_id" binding:"required"`
	Seats              int   `json:"seats" binding:"required,min=1,max=50"`
	ReservationTimeout int   `json:"reservation_timeout" binding:"min=1,max=1440"`
}

// BookingStats представляет статистику по бронированиям
type BookingStats struct {
	TotalBookings    int64                          `json:"total_bookings"`
	BookingsByStatus map[entity.BookingStatus]int64 `json:"bookings_by_status"`
	AverageSeats     float64                        `json:"average_seats"`
	PopularEvents    []*EventBookingCount           `json:"popular_events"`
	DailyBookings    int64                          `json:"daily_bookings"`
	WeeklyBookings   int64                          `json:"weekly_bookings"`
	MonthlyBookings  int64                          `json:"monthly_bookings"`
	Revenue          float64                        `json:"revenue"`
}

// EventBookingCount представляет мероприятие с количеством бронирований
type EventBookingCount struct {
	EventID    int64  `json:"event_id"`
	EventTitle string `json:"event_title"`
	Bookings   int64  `json:"bookings"`
	Seats      int64  `json:"seats"`
}

// BookingDetails представляет детальную информацию о бронировании
type BookingDetails struct {
	Booking    *entity.Booking `json:"booking"`
	Event      *entity.Event   `json:"event"`
	User       *entity.User    `json:"user"`
	TimeLeft   time.Duration   `json:"time_left,omitempty"`
	IsExpired  bool            `json:"is_expired"`
	CanConfirm bool            `json:"can_confirm"`
}

// TaskPublisher интерфейс для публикации задач в очередь
type TaskPublisher interface {
	Publish(ctx context.Context, task *Task) error
}

// Task представляет задачу для очереди
type Task struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data"`
	ExecuteAt  time.Time              `json:"execute_at"`
	MaxRetries int                    `json:"max_retries"`
	Attempts   int                    `json:"attempts"`
}

// Константы типов задач
const (
	TaskTypeExpireBooking        = "expire_booking"
	TaskTypeSendNotification     = "send_notification"
	TaskTypeCleanupExpired       = "cleanup_expired"
	TaskTypeReminderNotification = "reminder_notification"
	TaskTypeEventReminder        = "event_reminder"
)

type bookingService struct {
	bookingRepo repository.BookingRepository
	eventRepo   repository.EventRepository
	userRepo    repository.UserRepository
	queue       TaskPublisher
	telegramBot *telegram.Bot
}

// NewBookingService создает новый экземпляр BookingService
func NewBookingService(
	bookingRepo repository.BookingRepository,
	eventRepo repository.EventRepository,
	userRepo repository.UserRepository,
	queue TaskPublisher,
	telegramBot *telegram.Bot,
) BookingService {
	return &bookingService{
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
		userRepo:    userRepo,
		queue:       queue,
		telegramBot: telegramBot,
	}
}

// BookSeats создает новое бронирование мест
func (s *bookingService) BookSeats(ctx context.Context, req *BookSeatsRequest) (*entity.Booking, error) {
	// Валидация мероприятия
	eventWithAvailability, err := s.eventRepo.GetByID(ctx, req.EventID)
	if err != nil {
		return nil, fmt.Errorf("мероприятие не найдено: %w", err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	if event.Date.Before(time.Now()) {
		return nil, fmt.Errorf("невозможно забронировать места на прошедшее мероприятие")
	}

	if eventWithAvailability.AvailableSeats < req.Seats {
		return nil, fmt.Errorf("недостаточно доступных мест: запрошено %d, доступно %d",
			req.Seats, eventWithAvailability.AvailableSeats)
	}

	// Валидация пользователя
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден: %w", err)
	}

	// Проверка существующего бронирования
	existingBooking, err := s.bookingRepo.GetByEventAndUser(ctx, req.EventID, req.UserID)
	if err != nil && err != entity.ErrBookingNotFound {
		return nil, fmt.Errorf("ошибка при проверке существующих бронирований: %w", err)
	}

	if existingBooking != nil {
		switch existingBooking.Status {
		case entity.BookingStatusPending:
			return nil, fmt.Errorf("у вас уже есть ожидающее бронирование на это мероприятие")
		case entity.BookingStatusConfirmed:
			return nil, fmt.Errorf("у вас уже есть подтвержденное бронирование на это мероприятие")
		}
	}

	// Установка времени резервирования по умолчанию
	timeout := req.ReservationTimeout
	if timeout == 0 {
		timeout = 30
	}

	// Создание бронирования
	booking := &entity.Booking{
		EventID:            req.EventID,
		UserID:             req.UserID,
		Seats:              req.Seats,
		Status:             entity.BookingStatusPending,
		ReservationTimeout: timeout,
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, fmt.Errorf("ошибка при создании бронирования: %w", err)
	}

	log.Printf("Бронирование создано: ID=%d, Event=%d, User=%d, Seats=%d",
		booking.ID, booking.EventID, booking.UserID, booking.Seats)

	// Планирование задач через очередь, если доступна
	if s.queue != nil {
		if err := s.scheduleBookingTasks(ctx, booking); err != nil {
			log.Printf("Ошибка при планировании задач бронирования: %v", err)
		}
	}

	// Отправка уведомления через Telegram
	if s.telegramBot != nil && user.TelegramID != "" {
		go s.sendBookingCreatedNotification(booking, event, user)
	}

	return booking, nil
}

// scheduleBookingTasks планирует задачи для бронирования
func (s *bookingService) scheduleBookingTasks(ctx context.Context, booking *entity.Booking) error {
	// Задача на истечение срока бронирования
	expirationTask := &Task{
		ID:   fmt.Sprintf("expire_booking_%d_%d", booking.ID, time.Now().Unix()),
		Type: TaskTypeExpireBooking,
		Data: map[string]interface{}{
			"booking_id": booking.ID,
			"event_id":   booking.EventID,
			"user_id":    booking.UserID,
			"expires_at": booking.ExpiresAt.Format(time.RFC3339),
		},
		ExecuteAt:  booking.ExpiresAt,
		MaxRetries: 3,
	}

	if err := s.queue.Publish(ctx, expirationTask); err != nil {
		return fmt.Errorf("ошибка при планировании задачи истечения: %w", err)
	}

	// Задача напоминания за 15 минут до истечения
	reminderTime := booking.ExpiresAt.Add(-15 * time.Minute)
	if reminderTime.After(time.Now()) {
		reminderTask := &Task{
			ID:   fmt.Sprintf("reminder_booking_%d_%d", booking.ID, time.Now().Unix()),
			Type: TaskTypeReminderNotification,
			Data: map[string]interface{}{
				"booking_id": booking.ID,
				"event_id":   booking.EventID,
				"user_id":    booking.UserID,
			},
			ExecuteAt:  reminderTime,
			MaxRetries: 2,
		}

		if err := s.queue.Publish(ctx, reminderTask); err != nil {
			return fmt.Errorf("ошибка при планировании задачи напоминания: %w", err)
		}
	}

	// Уведомление о создании бронирования
	notificationTask := &Task{
		ID:   fmt.Sprintf("notification_booking_created_%d_%d", booking.ID, time.Now().Unix()),
		Type: TaskTypeSendNotification,
		Data: map[string]interface{}{
			"notification_type": "booking_created",
			"booking_id":        booking.ID,
			"event_id":          booking.EventID,
			"user_id":           booking.UserID,
		},
		ExecuteAt:  time.Now().Add(5 * time.Second),
		MaxRetries: 3,
	}

	if err := s.queue.Publish(ctx, notificationTask); err != nil {
		return fmt.Errorf("ошибка при планировании задачи уведомления: %w", err)
	}

	return nil
}

// sendBookingCreatedNotification отправляет уведомление о создании бронирования
func (s *bookingService) sendBookingCreatedNotification(booking *entity.Booking, event *entity.Event, user *entity.User) {
	message := fmt.Sprintf(
		"🎫 Бронирование создано!\n\n"+
			"Мероприятие: %s\n"+
			"Дата: %s\n"+
			"Количество мест: %d\n"+
			"Номер брони: #%d\n"+
			"Статус: Ожидание оплаты\n"+
			"Подтвердите бронирование до: %s\n\n"+
			"Не забудьте подтвердить бронирование вовремя!",
		event.Title,
		event.Date.Format("02.01.2006 в 15:04"),
		booking.Seats,
		booking.ID,
		booking.ExpiresAt.Format("02.01.2006 в 15:04"),
	)

	if err := s.telegramBot.SendMessage(user.TelegramID, message); err != nil {
		log.Printf("Ошибка при отправке Telegram уведомления пользователю %d: %v", user.ID, err)
	}
}

// ConfirmBooking подтверждает бронирование
func (s *bookingService) ConfirmBooking(ctx context.Context, bookingID int64) error {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("бронирование не найдено: %w", err)
	}

	if booking.Status != entity.BookingStatusPending {
		return fmt.Errorf("бронирование не в статусе ожидания")
	}

	if time.Now().After(booking.ExpiresAt) {
		if err := s.bookingRepo.UpdateStatus(ctx, bookingID, entity.BookingStatusExpired); err != nil {
			return fmt.Errorf("ошибка при обновлении статуса истекшего бронирования: %w", err)
		}
		return fmt.Errorf("бронирование истекло")
	}

	eventWithAvailability, err := s.eventRepo.GetByID(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("ошибка при получении информации о мероприятии: %w", err)
	}

	if eventWithAvailability.AvailableSeats < booking.Seats {
		return fmt.Errorf("недостаточно доступных мест для подтверждения")
	}

	if err := s.bookingRepo.UpdateStatus(ctx, bookingID, entity.BookingStatusConfirmed); err != nil {
		return fmt.Errorf("ошибка при подтверждении бронирования: %w", err)
	}

	log.Printf("Бронирование подтверждено: ID=%d", bookingID)

	// Отправка уведомления о подтверждении
	if s.queue != nil {
		notificationTask := &Task{
			ID:   fmt.Sprintf("notification_booking_confirmed_%d_%d", bookingID, time.Now().Unix()),
			Type: TaskTypeSendNotification,
			Data: map[string]interface{}{
				"notification_type": "booking_confirmed",
				"booking_id":        bookingID,
				"event_id":          booking.EventID,
				"user_id":           booking.UserID,
			},
			ExecuteAt:  time.Now().Add(2 * time.Second),
			MaxRetries: 3,
		}

		if err := s.queue.Publish(ctx, notificationTask); err != nil {
			log.Printf("Ошибка при планировании уведомления о подтверждении: %v", err)
		}
	}

	return nil
}

// CancelBooking отменяет бронирование
func (s *bookingService) CancelBooking(ctx context.Context, bookingID int64, reason string) error {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("бронирование не найдено: %w", err)
	}

	if booking.Status == entity.BookingStatusCancelled || booking.Status == entity.BookingStatusExpired {
		return fmt.Errorf("бронирование уже отменено")
	}

	if err := s.bookingRepo.UpdateStatus(ctx, bookingID, entity.BookingStatusCancelled); err != nil {
		return fmt.Errorf("ошибка при отмене бронирования: %w", err)
	}

	log.Printf("Бронирование отменено: ID=%d, Причина: %s", bookingID, reason)

	// Отправка уведомления об отмене
	if s.telegramBot != nil {
		user, err := s.userRepo.GetByID(ctx, booking.UserID)
		if err == nil && user.TelegramID != "" {
			eventWithAvailability, err := s.eventRepo.GetByID(ctx, booking.EventID)
			if err == nil {
				// Преобразуем в базовый Event
				event := &eventWithAvailability.Event
				message := fmt.Sprintf(
					"❌ Бронирование отменено\n\n"+
						"Мероприятие: %s\n"+
						"Дата: %s\n"+
						"Количество мест: %d\n"+
						"Причина: %s\n\n"+
						"Если это ошибка, свяжитесь с поддержкой.",
					event.Title,
					event.Date.Format("02.01.2006 в 15:04"),
					booking.Seats,
					reason,
				)

				go s.telegramBot.SendMessage(user.TelegramID, message)
			}
		}
	}

	return nil
}

// GetBooking возвращает бронирование по ID
func (s *bookingService) GetBooking(ctx context.Context, id int64) (*entity.Booking, error) {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бронирования: %w", err)
	}
	return booking, nil
}

// GetUserBookings возвращает все бронирования пользователя
func (s *bookingService) GetUserBookings(ctx context.Context, userID int64) ([]*entity.Booking, error) {
	bookings, err := s.bookingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бронирований пользователя: %w", err)
	}
	return bookings, nil
}

// GetEventBookings возвращает все бронирования мероприятия
func (s *bookingService) GetEventBookings(ctx context.Context, eventID int64) ([]*entity.Booking, error) {
	bookings, err := s.bookingRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бронирований мероприятия: %w", err)
	}
	return bookings, nil
}

// CancelExpiredBookings отменяет все истекшие бронирования
func (s *bookingService) CancelExpiredBookings(ctx context.Context) error {
	expiredBookings, err := s.bookingRepo.GetExpiredBookings(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка при получении истекших бронирований: %w", err)
	}

	cancelledCount := 0
	for _, expired := range expiredBookings {
		if err := s.bookingRepo.UpdateStatus(ctx, expired.BookingID, entity.BookingStatusExpired); err != nil {
			log.Printf("Ошибка при отмене истекшего бронирования %d: %v", expired.BookingID, err)
			continue
		}

		if s.telegramBot != nil && expired.TelegramID != "" {
			message := fmt.Sprintf(
				"⏰ Бронирование истекло\n\n"+
					"Мероприятие: %s\n"+
					"Бронирование #%d было автоматически отменено.\n\n"+
					"Вы можете создать новое бронирование, если места еще доступны.",
				expired.EventTitle,
				expired.BookingID,
			)

			go s.telegramBot.SendMessage(expired.TelegramID, message)
		}

		cancelledCount++
	}

	log.Printf("Отменено %d истекших бронирований", cancelledCount)
	return nil
}

// GetExpiredBookings возвращает список истекших бронирований
func (s *bookingService) GetExpiredBookings(ctx context.Context, before time.Time) ([]*entity.BookingExpiration, error) {
	bookings, err := s.bookingRepo.GetExpiredBookings(ctx, before)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении истекших бронирований: %w", err)
	}
	return bookings, nil
}

// ExpireBooking помечает бронирование как истекшее
func (s *bookingService) ExpireBooking(ctx context.Context, bookingID int64) error {
	return s.bookingRepo.UpdateStatus(ctx, bookingID, entity.BookingStatusExpired)
}

// GetBookingsByStatus возвращает бронирования по статусу
func (s *bookingService) GetBookingsByStatus(ctx context.Context, status entity.BookingStatus) ([]*entity.Booking, error) {
	bookings, err := s.bookingRepo.GetByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бронирований по статусу: %w", err)
	}
	return bookings, nil
}

// UpdateBookingSeats обновляет количество мест в бронировании
func (s *bookingService) UpdateBookingSeats(ctx context.Context, bookingID int64, seats int) error {
	if seats <= 0 {
		return fmt.Errorf("количество мест должно быть положительным")
	}

	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("бронирование не найдено: %w", err)
	}

	if booking.Status != entity.BookingStatusPending {
		return fmt.Errorf("можно обновлять места только для бронирований в статусе ожидания")
	}

	eventWithAvailability, err := s.eventRepo.GetByID(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("ошибка при получении информации о мероприятии: %w", err)
	}

	seatDifference := seats - booking.Seats
	if eventWithAvailability.AvailableSeats+seatDifference < 0 {
		return fmt.Errorf("недостаточно доступных мест")
	}

	booking.Seats = seats
	if err := s.bookingRepo.Update(ctx, booking); err != nil {
		return fmt.Errorf("ошибка при обновлении количества мест: %w", err)
	}

	return nil
}

// UpdateBookingStatus обновляет статус бронирования
func (s *bookingService) UpdateBookingStatus(ctx context.Context, bookingID int64, status entity.BookingStatus) error {
	switch status {
	case entity.BookingStatusPending, entity.BookingStatusConfirmed,
		entity.BookingStatusCancelled, entity.BookingStatusExpired:
		// Valid status
	default:
		return fmt.Errorf("неверный статус бронирования")
	}

	if err := s.bookingRepo.UpdateStatus(ctx, bookingID, status); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса бронирования: %w", err)
	}
	return nil
}

// GetBookingStats возвращает статистику по бронированиям
func (s *bookingService) GetBookingStats(ctx context.Context) (*BookingStats, error) {
	allBookings, err := s.bookingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бронирований для статистики: %w", err)
	}

	stats := &BookingStats{
		TotalBookings:    int64(len(allBookings)),
		BookingsByStatus: make(map[entity.BookingStatus]int64),
		PopularEvents:    make([]*EventBookingCount, 0),
	}

	totalSeats := 0
	eventBookings := make(map[int64]*EventBookingCount)
	eventTitles := make(map[int64]string)

	now := time.Now()
	dailyCount := int64(0)
	weeklyCount := int64(0)
	monthlyCount := int64(0)

	for _, booking := range allBookings {
		stats.BookingsByStatus[booking.Status]++
		totalSeats += booking.Seats

		if _, exists := eventBookings[booking.EventID]; !exists {
			eventBookings[booking.EventID] = &EventBookingCount{
				EventID:  booking.EventID,
				Bookings: 0,
				Seats:    0,
			}
		}
		eventBookings[booking.EventID].Bookings++
		eventBookings[booking.EventID].Seats += int64(booking.Seats)

		if _, exists := eventTitles[booking.EventID]; !exists {
			event, err := s.eventRepo.GetByID(ctx, booking.EventID)
			if err == nil {
				eventTitles[booking.EventID] = event.Title
			}
		}

		if booking.CreatedAt.After(now.AddDate(0, 0, -1)) {
			dailyCount++
		}
		if booking.CreatedAt.After(now.AddDate(0, 0, -7)) {
			weeklyCount++
		}
		if booking.CreatedAt.After(now.AddDate(0, -1, 0)) {
			monthlyCount++
		}
	}

	for eventID, eventCount := range eventBookings {
		eventCount.EventTitle = eventTitles[eventID]
		stats.PopularEvents = append(stats.PopularEvents, eventCount)
	}

	stats.sortPopularEvents()

	if len(allBookings) > 0 {
		stats.AverageSeats = float64(totalSeats) / float64(len(allBookings))
	}

	stats.DailyBookings = dailyCount
	stats.WeeklyBookings = weeklyCount
	stats.MonthlyBookings = monthlyCount
	stats.Revenue = float64(totalSeats) * 1000.0

	return stats, nil
}

// sortPopularEvents сортирует популярные мероприятия по количеству бронирований
func (s *BookingStats) sortPopularEvents() {
	for i := 0; i < len(s.PopularEvents)-1; i++ {
		for j := i + 1; j < len(s.PopularEvents); j++ {
			if s.PopularEvents[i].Bookings < s.PopularEvents[j].Bookings {
				s.PopularEvents[i], s.PopularEvents[j] = s.PopularEvents[j], s.PopularEvents[i]
			}
		}
	}
}

// GetAllBookings возвращает все бронирования
func (s *bookingService) GetAllBookings(ctx context.Context) ([]*entity.Booking, error) {
	bookings, err := s.bookingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении всех бронирований: %w", err)
	}
	return bookings, nil
}

// DeleteBooking удаляет бронирование
func (s *bookingService) DeleteBooking(ctx context.Context, bookingID int64) error {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("бронирование не найдено: %w", err)
	}

	if booking.Status == entity.BookingStatusConfirmed {
		return fmt.Errorf("невозможно удалить подтвержденное бронирование")
	}

	if err := s.bookingRepo.Delete(ctx, bookingID); err != nil {
		return fmt.Errorf("ошибка при удалении бронирования: %w", err)
	}
	return nil
}

// GetRecentBookings возвращает последние бронирования
func (s *bookingService) GetRecentBookings(ctx context.Context, limit int) ([]*entity.Booking, error) {
	if limit <= 0 {
		limit = 50
	}

	bookings, err := s.bookingRepo.GetRecentBookings(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении последних бронирований: %w", err)
	}
	return bookings, nil
}

// GetBookingWithDetails возвращает детальную информацию о бронировании
func (s *bookingService) GetBookingWithDetails(ctx context.Context, bookingID int64) (*BookingDetails, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бронирования: %w", err)
	}

	eventWithAvailability, err := s.eventRepo.GetByID(ctx, booking.EventID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении информации о мероприятии: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, booking.UserID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении информации о пользователе: %w", err)
	}

	details := &BookingDetails{
		Booking: booking,
		Event:   &eventWithAvailability.Event, // Преобразуем в базовый Event
		User:    user,
	}

	if booking.Status == entity.BookingStatusPending {
		details.TimeLeft = time.Until(booking.ExpiresAt)
		details.IsExpired = details.TimeLeft <= 0
		details.CanConfirm = !details.IsExpired
	}

	return details, nil
}

// CheckBookingAvailability проверяет доступность мест для бронирования
func (s *bookingService) CheckBookingAvailability(ctx context.Context, eventID int64, seats int) (bool, error) {
	if seats <= 0 {
		return false, fmt.Errorf("количество мест должно быть положительным")
	}

	eventWithAvailability, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("ошибка при получении информации о мероприятии: %w", err)
	}

	if eventWithAvailability.Date.Before(time.Now()) {
		return false, fmt.Errorf("мероприятие уже прошло")
	}

	available := eventWithAvailability.AvailableSeats >= seats
	return available, nil
}
