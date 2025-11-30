package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
	"github.com/ds124wfegd/WB_L3/5/internal/service"
)

// TaskHandler обрабатывает задачи из очереди
type TaskHandler struct {
	bookingService service.BookingService
	eventService   service.EventService
	userService    service.UserService
	telegramBot    TelegramBot
}

// TelegramBot интерфейс для Telegram бота
type TelegramBot interface {
	SendMessage(chatID, text string) error
}

// NewTaskHandler создает новый обработчик задач
func NewTaskHandler(
	bookingService service.BookingService,
	eventService service.EventService,
	userService service.UserService,
	telegramBot TelegramBot,
) *TaskHandler {
	return &TaskHandler{
		bookingService: bookingService,
		eventService:   eventService,
		userService:    userService,
		telegramBot:    telegramBot,
	}
}

// HandleTask обрабатывает задачу
func (h *TaskHandler) HandleTask(task *Task) error {
	log.Printf("Обработка задачи %s типа %s (попытка %d/%d)",
		task.ID, task.Type, task.Attempts, task.MaxRetries)

	switch task.Type {
	case TaskTypeExpireBooking:
		return h.handleExpireBooking(task)
	case TaskTypeSendNotification:
		return h.handleSendNotification(task)
	case TaskTypeCleanupExpired:
		return h.handleCleanupExpired(task)
	case TaskTypeReminderNotification:
		return h.handleReminderNotification(task)
	case TaskTypeEventReminder:
		return h.handleEventReminder(task)
	default:
		return fmt.Errorf("неизвестный тип задачи: %s", task.Type)
	}
}

// handleExpireBooking обрабатывает истечение срока бронирования
func (h *TaskHandler) handleExpireBooking(task *Task) error {
	ctx := context.Background()

	bookingID, ok := task.Data["booking_id"].(float64)
	if !ok {
		return fmt.Errorf("неверный booking_id в данных задачи")
	}

	// Получаем информацию о бронировании
	booking, err := h.bookingService.GetBooking(ctx, int64(bookingID))
	if err != nil {
		return fmt.Errorf("не удалось получить бронирование %d: %v", int64(bookingID), err)
	}

	// Проверяем, что бронирование все еще в ожидании
	if booking.Status != entity.BookingStatusPending {
		log.Printf("Бронирование %d больше не в статусе ожидания (статус: %s), пропускаем истечение",
			booking.ID, booking.Status)
		return nil
	}

	// Проверяем, что срок действительно истек
	if time.Now().Before(booking.ExpiresAt) {
		log.Printf("Бронирование %d еще не истекло (истекает в: %s)",
			booking.ID, booking.ExpiresAt.Format(time.RFC3339))
		return nil
	}

	// Помечаем бронирование как истекшее
	if err := h.bookingService.ExpireBooking(ctx, booking.ID); err != nil {
		return fmt.Errorf("не удалось истечь бронирование %d: %v", booking.ID, err)
	}

	log.Printf("Бронирование %d успешно истекло", booking.ID)

	// Отправляем уведомление об истечении
	if err := h.sendExpirationNotification(ctx, booking); err != nil {
		log.Printf("Не удалось отправить уведомление об истечении для бронирования %d: %v", booking.ID, err)
	}

	return nil
}

// handleSendNotification обрабатывает отправку уведомлений
func (h *TaskHandler) handleSendNotification(task *Task) error {

	notificationType, ok := task.Data["notification_type"].(string)
	if !ok {
		return fmt.Errorf("неверный notification_type в данных задачи")
	}

	switch notificationType {
	case "booking_confirmed":
		return h.handleBookingConfirmedNotification(task)
	case "booking_created":
		return h.handleBookingCreatedNotification(task)
	case "event_cancelled":
		return h.handleEventCancelledNotification(task)
	case "custom_message":
		return h.handleCustomMessageNotification(task)
	default:
		return fmt.Errorf("неизвестный тип уведомления: %s", notificationType)
	}
}

// handleBookingConfirmedNotification отправляет уведомление о подтверждении бронирования
func (h *TaskHandler) handleBookingConfirmedNotification(task *Task) error {
	ctx := context.Background()

	bookingID, ok := task.Data["booking_id"].(float64)
	if !ok {
		return fmt.Errorf("неверный booking_id в данных задачи")
	}

	booking, err := h.bookingService.GetBooking(ctx, int64(bookingID))
	if err != nil {
		return fmt.Errorf("не удалось получить бронирование %d: %v", int64(bookingID), err)
	}

	eventWithAvailability, err := h.eventService.GetEvent(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("не удалось получить мероприятие %d: %v", booking.EventID, err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	user, err := h.userService.GetUserByID(ctx, booking.UserID)
	if err != nil {
		return fmt.Errorf("не удалось получить пользователя %d: %v", booking.UserID, err)
	}

	if user.TelegramID != "" && h.telegramBot != nil {
		message := fmt.Sprintf(
			"✅ Ваше бронирование подтверждено!\n\n"+
				"Мероприятие: %s\n"+
				"Дата: %s\n"+
				"Количество мест: %d\n"+
				"Номер брони: #%d\n\n"+
				"Ждем вас на мероприятии!",
			event.Title,
			event.Date.Format("02.01.2006 в 15:04"),
			booking.Seats,
			booking.ID,
		)

		if err := h.telegramBot.SendMessage(user.TelegramID, message); err != nil {
			return fmt.Errorf("не удалось отправить Telegram сообщение: %v", err)
		}
	}

	log.Printf("Отправлено уведомление о подтверждении для бронирования %d пользователю %d", booking.ID, user.ID)
	return nil
}

// handleBookingCreatedNotification отправляет уведомление о создании бронирования
func (h *TaskHandler) handleBookingCreatedNotification(task *Task) error {
	ctx := context.Background()

	bookingID, ok := task.Data["booking_id"].(float64)
	if !ok {
		return fmt.Errorf("неверный booking_id в данных задачи")
	}

	booking, err := h.bookingService.GetBooking(ctx, int64(bookingID))
	if err != nil {
		return fmt.Errorf("не удалось получить бронирование %d: %v", int64(bookingID), err)
	}

	eventWithAvailability, err := h.eventService.GetEvent(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("не удалось получить мероприятие %d: %v", booking.EventID, err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	user, err := h.userService.GetUserByID(ctx, booking.UserID)
	if err != nil {
		return fmt.Errorf("не удалось получить пользователя %d: %v", booking.UserID, err)
	}

	if user.TelegramID != "" && h.telegramBot != nil {
		expiresAt := booking.ExpiresAt.Format("02.01.2006 в 15:04")
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
			expiresAt,
		)

		if err := h.telegramBot.SendMessage(user.TelegramID, message); err != nil {
			return fmt.Errorf("не удалось отправить Telegram сообщение: %v", err)
		}
	}

	log.Printf("Отправлено уведомление о создании для бронирования %d пользователю %d", booking.ID, user.ID)
	return nil
}

// handleEventCancelledNotification отправляет уведомление об отмене мероприятия
func (h *TaskHandler) handleEventCancelledNotification(task *Task) error {
	ctx := context.Background()

	eventID, ok := task.Data["event_id"].(float64)
	if !ok {
		return fmt.Errorf("неверный event_id в данных задачи")
	}

	reason, _ := task.Data["reason"].(string)
	if reason == "" {
		reason = "по техническим причинам"
	}

	eventWithAvailability, err := h.eventService.GetEvent(ctx, int64(eventID))
	if err != nil {
		return fmt.Errorf("не удалось получить мероприятие %d: %v", int64(eventID), err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	// Получаем все бронирования для этого мероприятия
	bookings, err := h.bookingService.GetEventBookings(ctx, int64(eventID))
	if err != nil {
		return fmt.Errorf("не удалось получить бронирования для мероприятия %d: %v", int64(eventID), err)
	}

	// Отправляем уведомления всем пользователям с подтвержденными бронированиями
	sentCount := 0
	for _, booking := range bookings {
		if booking.Status == entity.BookingStatusConfirmed {
			user, err := h.userService.GetUserByID(ctx, booking.UserID)
			if err != nil {
				log.Printf("Не удалось получить пользователя %d для уведомления об отмене: %v", booking.UserID, err)
				continue
			}

			if user.TelegramID != "" && h.telegramBot != nil {
				message := fmt.Sprintf(
					"❌ Мероприятие отменено\n\n"+
						"Мероприятие: %s\n"+
						"Дата: %s\n"+
						"Причина: %s\n\n"+
						"Приносим извинения за доставленные неудобства. "+
						"Средства за билеты будут возвращены в течение 3-5 рабочих дней.",
					event.Title,
					event.Date.Format("02.01.2006 в 15:04"),
					reason,
				)

				if err := h.telegramBot.SendMessage(user.TelegramID, message); err != nil {
					log.Printf("Не удалось отправить уведомление об отмене пользователю %d: %v", user.ID, err)
				} else {
					sentCount++
				}
			}
		}
	}

	log.Printf("Отправлены уведомления об отмене мероприятия %d для %d пользователей", eventID, sentCount)
	return nil
}

// handleCustomMessageNotification отправляет кастомные сообщения
func (h *TaskHandler) handleCustomMessageNotification(task *Task) error {
	ctx := context.Background()

	messageText, ok := task.Data["message"].(string)
	if !ok {
		return fmt.Errorf("неверный message в данных задачи")
	}

	userIDsInterface, ok := task.Data["user_ids"].([]interface{})
	if !ok {
		return fmt.Errorf("неверный user_ids в данных задачи")
	}

	var userIDs []int64
	for _, id := range userIDsInterface {
		if idFloat, ok := id.(float64); ok {
			userIDs = append(userIDs, int64(idFloat))
		}
	}

	if len(userIDs) == 0 {
		log.Printf("Не указаны конкретные ID пользователей, пропускаем широковещательное сообщение")
		return nil
	}

	sentCount := 0
	for _, userID := range userIDs {
		user, err := h.userService.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Не удалось получить пользователя %d для кастомного уведомления: %v", userID, err)
			continue
		}

		if user.TelegramID != "" && h.telegramBot != nil {
			if err := h.telegramBot.SendMessage(user.TelegramID, messageText); err != nil {
				log.Printf("Не удалось отправить кастомное сообщение пользователю %d: %v", user.ID, err)
			} else {
				sentCount++
			}
		}
	}

	log.Printf("Отправлено кастомное сообщение %d/%d пользователям", sentCount, len(userIDs))
	return nil
}

// handleCleanupExpired выполняет массовую очистку истекших бронирований
func (h *TaskHandler) handleCleanupExpired(task *Task) error {
	ctx := context.Background()

	log.Printf("Начало массовой очистки истекших бронирований")

	expiredBefore, ok := task.Data["expired_before"].(string)
	if !ok {
		// По умолчанию 1 час назад для безопасности
		expiredBefore = time.Now().Add(-time.Hour).Format(time.RFC3339)
	}

	cutoffTime, err := time.Parse(time.RFC3339, expiredBefore)
	if err != nil {
		return fmt.Errorf("неверный формат expired_before: %v", err)
	}

	// Получаем истекшие бронирования
	expiredBookings, err := h.bookingService.GetExpiredBookings(ctx, cutoffTime)
	if err != nil {
		return fmt.Errorf("не удалось получить истекшие бронирования: %v", err)
	}

	log.Printf("Найдено %d истекших бронирований для очистки", len(expiredBookings))

	successCount := 0
	for _, expired := range expiredBookings {
		if err := h.bookingService.ExpireBooking(ctx, expired.BookingID); err != nil {
			log.Printf("Не удалось истечь бронирование %d: %v", expired.BookingID, err)
		} else {
			successCount++
		}
	}

	log.Printf("Успешно очищено %d/%d истекших бронирований", successCount, len(expiredBookings))
	return nil
}

// handleReminderNotification отправляет напоминания о бронированиях
func (h *TaskHandler) handleReminderNotification(task *Task) error {
	ctx := context.Background()

	bookingID, ok := task.Data["booking_id"].(float64)
	if !ok {
		return fmt.Errorf("неверный booking_id в данных задачи")
	}

	booking, err := h.bookingService.GetBooking(ctx, int64(bookingID))
	if err != nil {
		return fmt.Errorf("не удалось получить бронирование %d: %v", int64(bookingID), err)
	}

	// Проверяем, что бронирование все еще в ожидании
	if booking.Status != entity.BookingStatusPending {
		return nil // Напоминание не нужно
	}

	eventWithAvailability, err := h.eventService.GetEvent(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("не удалось получить мероприятие %d: %v", booking.EventID, err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	user, err := h.userService.GetUserByID(ctx, booking.UserID)
	if err != nil {
		return fmt.Errorf("не удалось получить пользователя %d: %v", booking.UserID, err)
	}

	if user.TelegramID != "" && h.telegramBot != nil {
		timeLeft := time.Until(booking.ExpiresAt)
		minutesLeft := int(timeLeft.Minutes())

		if minutesLeft <= 0 {
			return nil // Напоминание для истекших бронирований не нужно
		}

		message := fmt.Sprintf(
			"⏰ Напоминание о бронировании\n\n"+
				"Мероприятие: %s\n"+
				"Дата: %s\n"+
				"Количество мест: %d\n"+
				"Номер брони: #%d\n"+
				"Осталось времени: %d минут\n\n"+
				"Не забудьте подтвердить бронирование!",
			event.Title,
			event.Date.Format("02.01.2006 в 15:04"),
			booking.Seats,
			booking.ID,
			minutesLeft,
		)

		if err := h.telegramBot.SendMessage(user.TelegramID, message); err != nil {
			return fmt.Errorf("не удалось отправить сообщение напоминания: %v", err)
		}
	}

	log.Printf("Отправлено напоминание для бронирования %d пользователю %d", booking.ID, user.ID)
	return nil
}

// handleEventReminder отправляет напоминания о мероприятиях
func (h *TaskHandler) handleEventReminder(task *Task) error {
	ctx := context.Background()

	eventID, ok := task.Data["event_id"].(float64)
	if !ok {
		return fmt.Errorf("неверный event_id в данных задачи")
	}

	eventWithAvailability, err := h.eventService.GetEvent(ctx, int64(eventID))
	if err != nil {
		return fmt.Errorf("не удалось получить мероприятие %d: %v", int64(eventID), err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	// Получаем все подтвержденные бронирования для этого мероприятия
	bookings, err := h.bookingService.GetEventBookings(ctx, int64(eventID))
	if err != nil {
		return fmt.Errorf("не удалось получить бронирования для мероприятия %d: %v", int64(eventID), err)
	}

	reminderHours, ok := task.Data["reminder_hours"].(float64)
	if !ok {
		reminderHours = 24 // По умолчанию 24 часа
	}

	sentCount := 0
	for _, booking := range bookings {
		if booking.Status == entity.BookingStatusConfirmed {
			user, err := h.userService.GetUserByID(ctx, booking.UserID)
			if err != nil {
				log.Printf("Не удалось получить пользователя %d для напоминания о мероприятии: %v", booking.UserID, err)
				continue
			}

			if user.TelegramID != "" && h.telegramBot != nil {
				message := fmt.Sprintf(
					"🔔 Напоминание о мероприятии\n\n"+
						"Мероприятие: %s\n"+
						"Дата и время: %s\n"+
						"Количество мест: %d\n"+
						"Номер брони: #%d\n\n"+
						"Мероприятие начнется через %.0f часов. Ждем вас!",
					event.Title,
					event.Date.Format("02.01.2006 в 15:04"),
					booking.Seats,
					booking.ID,
					reminderHours,
				)

				if err := h.telegramBot.SendMessage(user.TelegramID, message); err != nil {
					log.Printf("Не удалось отправить напоминание о мероприятии пользователю %d: %v", user.ID, err)
				} else {
					sentCount++
				}
			}
		}
	}

	log.Printf("Отправлены напоминания о мероприятии %d для %d пользователей", eventID, sentCount)
	return nil
}

// sendExpirationNotification отправляет уведомление об истечении бронирования
func (h *TaskHandler) sendExpirationNotification(ctx context.Context, booking *entity.Booking) error {
	eventWithAvailability, err := h.eventService.GetEvent(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("не удалось получить мероприятие %d: %v", booking.EventID, err)
	}

	// Преобразуем в базовый Event
	event := &eventWithAvailability.Event

	user, err := h.userService.GetUserByID(ctx, booking.UserID)
	if err != nil {
		return fmt.Errorf("не удалось получить пользователя %d: %v", booking.UserID, err)
	}

	if user.TelegramID != "" && h.telegramBot != nil {
		message := fmt.Sprintf(
			"❌ Бронирование отменено\n\n"+
				"Мероприятие: %s\n"+
				"Дата: %s\n"+
				"Количество мест: %d\n"+
				"Номер брони: #%d\n\n"+
				"Бронирование было автоматически отменено, так как вы не подтвердили его вовремя.",
			event.Title,
			event.Date.Format("02.01.2006 в 15:04"),
			booking.Seats,
			booking.ID,
		)

		if err := h.telegramBot.SendMessage(user.TelegramID, message); err != nil {
			return fmt.Errorf("не удалось отправить уведомление об истечении: %v", err)
		}
	}

	return nil
}
