package transport

import (
	"time"

	"github.com/ds124wfegd/WB_L3/1/internal/service"
	"github.com/gin-gonic/gin"
)

func InitRoutes(usecase service.NotificationUseCase) *gin.Engine {
	router := gin.Default()

	handler := NewNotificationHandler(usecase)

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/notify", handler.CreateNotification)
		api.GET("/notify/:id", handler.GetNotification)
		api.DELETE("/notify/:id", handler.CancelNotification)
		api.GET("/notifications", handler.GetNotifications)

		router.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "healthy",
				"service":   "notification-service",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})

		router.GET("/", func(c *gin.Context) {
			c.Data(200, "text/html; charset=utf-8", []byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Notification Service</title>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <div class="container">
        <h1>Notification Service</h1>
        
        <div class="notification-form">
            <h2>Create New Notification</h2>
            <form id="notificationForm">
                <div class="form-group">
                    <label for="userId">User ID:</label>
                    <input type="text" id="userId" required>
                </div>
                <div class="form-group">
                    <label for="title">Title:</label>
                    <input type="text" id="title" required>
                </div>
                <div class="form-group">
                    <label for="message">Message:</label>
                    <textarea id="message" required></textarea>
                </div>
                <div class="form-group">
                    <label for="sendTime">Send Time:</label>
                    <input type="datetime-local" id="sendTime" required>
                </div>
                <button type="submit">Schedule Notification</button>
            </form>
        </div>

        <div class="notifications-list">
            <h2>Scheduled Notifications</h2>
            <button onclick="loadNotifications()">Refresh</button>
            <div id="notifications">Click "Refresh" to load notifications</div>
        </div>
    </div>

    <script>
        class NotificationService {
            constructor() {
                this.baseUrl = '/api/v1';
                this.init();
            }

            init() {
                document.getElementById('notificationForm').addEventListener('submit', (e) => {
                    e.preventDefault();
                    this.createNotification();
                });
            }

            async createNotification() {
                const notification = {
                    user_id: document.getElementById('userId').value,
                    title: document.getElementById('title').value,
                    message: document.getElementById('message').value,
                    send_time: new Date(document.getElementById('sendTime').value).toISOString()
                };

                try {
                    const response = await fetch(this.baseUrl + '/notify', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify(notification)
                    });

                    if (response.ok) {
                        alert('Notification scheduled successfully!');
                        document.getElementById('notificationForm').reset();
                        this.loadNotifications();
                    } else {
                        const error = await response.json();
                        alert('Error: ' + error.error);
                    }
                } catch (error) {
                    alert('Error creating notification: ' + error.message);
                }
            }

            async loadNotifications() {
                try {
                    const container = document.getElementById('notifications');
                    container.innerHTML = '<p>Loading notifications...</p>';
                    
                    const response = await fetch(this.baseUrl + '/notifications');
                    
                    if (response.ok) {
                        const data = await response.json();
                        if (data.notifications && data.notifications.length > 0) {
                            let html = '';
                            data.notifications.forEach(notification => {
                                const sendTime = new Date(notification.send_time);
                                const title = this.escapeHtml(notification.title);
                                const userId = this.escapeHtml(notification.user_id);
                                const message = this.escapeHtml(notification.message);
                                let status = this.escapeHtml(notification.status || 'scheduled');
								status = status.replace('_ Cancel', '');
                                const timeString = this.escapeHtml(sendTime.toLocaleString());
                                html += '<div class="notification-item">' +
                                        '<div class="notification-header">' +
                                        '<strong class="notification-title">' + title + '</strong>' +
                                        '<span class="notification-id">#' + notification.id + '</span>' +
                                        '</div>' +
                                        '<div class="notification-user">User: ' + userId + '</div>' +
                                        '<div class="notification-message">' + message + '</div>' +
                                        '<div class="notification-footer">' +
                                        '<span class="notification-time">Scheduled: ' + timeString + '</span>' +
                                        '<span class="notification-status">Status: ' + status + '</span>' +
                                        '</div>' +
                                        '</div>';
                            });
                            container.innerHTML = html;
                        } else {
                            container.innerHTML = '<p>No scheduled notifications found</p>';
                        }
                    } else {
                        container.innerHTML = '<p>Notification list feature would be implemented with a proper listing endpoint</p>';
                    }
                    
                } catch (error) {
                    console.error('Error loading notifications:', error);
                    container.innerHTML = '<p>Error loading notifications. Please try again.</p>';
                }
            }

            escapeHtml(text) {
                if (text === null || text === undefined) {
                    return '';
                }
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            }
        }

        let notificationService;

        document.addEventListener('DOMContentLoaded', () => {
            notificationService = new NotificationService();
        });

        function loadNotifications() {
            if (notificationService) {
                notificationService.loadNotifications();
            }
        }
    </script>
</body>
</html>
    `))
		})
	}

	return router
}
