package service

import (
	"context"
	"fmt"
	"time"

	repository "github.com/ds124wfegd/WB_L3/5/internal/database/postgres"
	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

// RegisterUserRequest represents the data needed to register a user
type RegisterUserRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Name       string `json:"name" binding:"required,min=2,max=100"`
	TelegramID string `json:"telegram_id,omitempty" binding:"max=100"`
}

// UpdateUserRequest represents the data needed to update a user
type UpdateUserRequest struct {
	Name       *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	TelegramID *string `json:"telegram_id,omitempty" binding:"omitempty,max=100"`
}

// UserFilter represents filters for searching users
type UserFilter struct {
	Email  string `json:"email,omitempty"`
	Name   string `json:"name,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// UserStats represents statistics about a user
type UserStats struct {
	User              *entity.User         `json:"user"`
	TotalBookings     int                  `json:"total_bookings"`
	ConfirmedBookings int                  `json:"confirmed_bookings"`
	PendingBookings   int                  `json:"pending_bookings"`
	CancelledBookings int                  `json:"cancelled_bookings"`
	FavoriteEvents    []*EventBookingCount `json:"favorite_events"`
	TotalSeatsBooked  int                  `json:"total_seats_booked"`
}

type userService struct {
	userRepo    repository.UserRepository
	bookingRepo repository.BookingRepository
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepo repository.UserRepository,
	bookingRepo repository.BookingRepository,
) UserService {
	return &userService{
		userRepo:    userRepo,
		bookingRepo: bookingRepo,
	}
}

func (s *userService) RegisterUser(ctx context.Context, req *RegisterUserRequest) (*entity.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && err != entity.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	user := &entity.User{
		Email:      req.Email,
		Name:       req.Name,
		TelegramID: req.TelegramID,
		CreatedAt:  time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUser(ctx context.Context, id int64) (*entity.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, id int64, req *UpdateUserRequest) (*entity.User, error) {
	// Get existing user
	existingUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing user: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		existingUser.Name = *req.Name
	}
	if req.TelegramID != nil {
		existingUser.TelegramID = *req.TelegramID
	}

	// Update in repository
	if err := s.userRepo.Update(ctx, existingUser); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return existingUser, nil
}

func (s *userService) LinkTelegram(ctx context.Context, userID int64, telegramID string) error {
	if telegramID == "" {
		return fmt.Errorf("telegram ID cannot be empty")
	}

	// Check if telegram ID is already linked to another user
	existingUser, err := s.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil && err != entity.ErrUserNotFound {
		return fmt.Errorf("failed to check telegram ID: %w", err)
	}
	if existingUser != nil && existingUser.ID != userID {
		return fmt.Errorf("telegram ID is already linked to another user")
	}

	if err := s.userRepo.UpdateTelegramID(ctx, userID, telegramID); err != nil {
		return fmt.Errorf("failed to link telegram: %w", err)
	}

	return nil
}

func (s *userService) GetUserStats(ctx context.Context, userID int64) (*UserStats, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user's bookings
	bookings, err := s.bookingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user bookings: %w", err)
	}

	stats := &UserStats{
		User: user,
	}

	// Calculate statistics
	eventBookings := make(map[int64]int)
	eventTitles := make(map[int64]string)

	for _, booking := range bookings {
		// Count by status
		switch booking.Status {
		case entity.BookingStatusConfirmed:
			stats.ConfirmedBookings++
			stats.TotalSeatsBooked += booking.Seats
		case entity.BookingStatusPending:
			stats.PendingBookings++
		case entity.BookingStatusCancelled, entity.BookingStatusExpired:
			stats.CancelledBookings++
		}

		// Count bookings per event for favorite events
		eventBookings[booking.EventID]++

		// Store event title if not already stored
		if _, exists := eventTitles[booking.EventID]; !exists {
			// In a real implementation, you'd get event details from eventRepo
			eventTitles[booking.EventID] = fmt.Sprintf("Event #%d", booking.EventID)
		}
	}

	stats.TotalBookings = len(bookings)

	// Find favorite events (events with most bookings)
	for eventID, count := range eventBookings {
		stats.FavoriteEvents = append(stats.FavoriteEvents, &EventBookingCount{
			EventID:    eventID,
			EventTitle: eventTitles[eventID],
			Bookings:   int64(count),
		})
	}

	// Sort favorite events by booking count (descending)
	// Implementation would sort stats.FavoriteEvents

	return stats, nil
}

func (s *userService) SearchUsers(ctx context.Context, filter *UserFilter) ([]*entity.User, error) {
	if filter == nil {
		filter = &UserFilter{}
	}

	// Set default values
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	// This would typically call a specialized repository method
	// For now, we'll demonstrate with email search
	if filter.Email != "" {
		user, err := s.userRepo.GetByEmail(ctx, filter.Email)
		if err != nil {
			if err == entity.ErrUserNotFound {
				return []*entity.User{}, nil
			}
			return nil, fmt.Errorf("failed to search user by email: %w", err)
		}
		return []*entity.User{user}, nil
	}

	// For name search or general search, you would implement a proper search method
	// For now, return empty result
	return []*entity.User{}, nil
}

// Исправляем метод DeleteUser в userService
func (s *userService) DeleteUser(ctx context.Context, id int64) error {
	// Проверяем, есть ли у пользователя активные бронирования
	bookings, err := s.bookingRepo.GetByUserID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check user bookings: %w", err)
	}

	// Проверяем наличие активных бронирований (pending или confirmed)
	for _, booking := range bookings {
		if booking.Status == entity.BookingStatusPending || booking.Status == entity.BookingStatusConfirmed {
			return fmt.Errorf("cannot delete user with active bookings")
		}
	}

	// Удаляем пользователя
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// Добавляем метод для получения всех пользователей
func (s *userService) GetAllUsers(ctx context.Context) ([]*entity.User, error) {
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	return users, nil
}

// Добавляем метод для поиска пользователей по имени
func (s *userService) SearchUsersByName(ctx context.Context, name string) ([]*entity.User, error) {
	if name == "" {
		return s.userRepo.GetAll(ctx)
	}

	users, err := s.userRepo.SearchByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to search users by name: %w", err)
	}

	return users, nil
}

// Реализуем метод GetUserByID в userService
func (s *userService) GetUserByID(ctx context.Context, id int64) (*entity.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
