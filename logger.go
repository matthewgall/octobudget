// Copyright 2025 Matthew Gall <me@matthewgall.dev>
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

package main

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with domain-specific methods
type Logger struct {
	*slog.Logger
}

// NewLogger creates a text-formatted logger
func NewLogger(debug bool) *Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return &Logger{slog.New(handler)}
}

// NewJSONLogger creates a JSON-formatted logger
func NewJSONLogger(debug bool) *Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stderr, opts)
	return &Logger{slog.New(handler)}
}

// WithComponent adds a component field to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{l.With("component", component)}
}

// WithAccountID adds a masked account ID field to the logger
func (l *Logger) WithAccountID(accountID string) *Logger {
	masked := accountID
	if len(accountID) > 5 {
		masked = accountID[:5] + "***"
	}
	return &Logger{l.With("account_id", masked)}
}

// LogAPIRequest logs an API request
func (l *Logger) LogAPIRequest(method, endpoint string) {
	l.Debug("API request",
		"method", method,
		"endpoint", endpoint,
	)
}

// LogAPIError logs an API error
func (l *Logger) LogAPIError(endpoint string, statusCode int, err error) {
	l.Error("API request failed",
		"endpoint", endpoint,
		"status_code", statusCode,
		"error", err,
	)
}

// LogDataCollection logs data collection progress
func (l *Logger) LogDataCollection(dataType string, count int) {
	l.Info("Data collected",
		"type", dataType,
		"count", count,
	)
}

// LogAnalysisStage logs analysis stage completion
func (l *Logger) LogAnalysisStage(stage string) {
	l.Info("Analysis stage completed",
		"stage", stage,
	)
}

// LogAnomalyDetected logs detected anomaly
func (l *Logger) LogAnomalyDetected(date, anomalyType string, deviation float64) {
	l.Warn("Anomaly detected",
		"date", date,
		"type", anomalyType,
		"deviation", fmt.Sprintf("%.1f%%", deviation),
	)
}

// LogStorageOperation logs storage operations
func (l *Logger) LogStorageOperation(operation, path string) {
	l.Debug("Storage operation",
		"operation", operation,
		"path", path,
	)
}

// UserMessage outputs a message directly to stdout (bypassing structured logging)
func (l *Logger) UserMessage(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
