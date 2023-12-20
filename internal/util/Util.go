package util

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func NewGinSlogger(level slog.Level, logger slog.Logger) func(*gin.Context) {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()
		latency := end.Sub(start)

		attributes := []slog.Attr{
			slog.Int("status", c.Writer.Status()),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("route", c.FullPath()),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", time.Duration(latency.Milliseconds())),
			slog.Time("time", end),
		}
		logger.LogAttrs(c.Request.Context(), level, "", attributes...)
	}
}
