package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

func Timeout(seconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(seconds)*time.Second)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
