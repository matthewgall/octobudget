// Copyright 2025 Matthew Gall <me@matthewgall.dev>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
)

// APIError represents an API-related error
type APIError struct {
	StatusCode int
	Endpoint   string
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("API error at %s (status %d): %s: %v", e.Endpoint, e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("API error at %s (status %d): %s", e.Endpoint, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if this error should be retried
func (e *APIError) IsRetryable() bool {
	return isRetryableStatus(e.StatusCode)
}

func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// AuthError represents an authentication or authorization error
type AuthError struct {
	Code    string
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("authentication error [%s]: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("authentication error: %s", e.Message)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// ValidationError represents a configuration or input validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation error for %s (%s): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// StorageError represents a storage operation error
type StorageError struct {
	Operation string
	Path      string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error during %s at %s: %v", e.Operation, e.Path, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// DataError represents insufficient or missing data error
type DataError struct {
	DataType string
	Message  string
}

func (e *DataError) Error() string {
	return fmt.Sprintf("data error for %s: %s", e.DataType, e.Message)
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error for %s: %s", e.Field, e.Message)
}
