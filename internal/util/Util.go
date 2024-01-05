// Copyright 2024 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
