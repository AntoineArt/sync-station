// Package errors provides custom error types and utilities for syncstation
package errors

import (
	"fmt"
	"strings"
)

// SyncError represents an error that occurred during sync operations
type SyncError struct {
	Operation string // The operation that failed (push, pull, sync, etc.)
	ItemName  string // The sync item name
	FilePath  string // The file path involved
	Cause     error  // The underlying error
	Code      ErrorCode
}

func (e *SyncError) Error() string {
	var parts []string
	
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation: %s", e.Operation))
	}
	if e.ItemName != "" {
		parts = append(parts, fmt.Sprintf("item: %s", e.ItemName))
	}
	if e.FilePath != "" {
		parts = append(parts, fmt.Sprintf("path: %s", e.FilePath))
	}
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %s", e.Cause.Error()))
	}
	
	return fmt.Sprintf("sync error [%s]: %s", e.Code, strings.Join(parts, ", "))
}

func (e *SyncError) Unwrap() error {
	return e.Cause
}

// ErrorCode represents different categories of errors
type ErrorCode string

const (
	ErrCodeFileNotFound     ErrorCode = "FILE_NOT_FOUND"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeHashMismatch     ErrorCode = "HASH_MISMATCH"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeInvalidPath      ErrorCode = "INVALID_PATH"
	ErrCodeGitOperation     ErrorCode = "GIT_OPERATION"
	ErrCodeConfigLoad       ErrorCode = "CONFIG_LOAD"
	ErrCodeConfigSave       ErrorCode = "CONFIG_SAVE"
	ErrCodeNetworkError     ErrorCode = "NETWORK_ERROR"
	ErrCodeIOError          ErrorCode = "IO_ERROR"
	ErrCodeValidation       ErrorCode = "VALIDATION"
	ErrCodeInternal         ErrorCode = "INTERNAL"
)

// ConfigError represents configuration-related errors
type ConfigError struct {
	ConfigType string // Type of config (local, sync-items, metadata, etc.)
	FilePath   string // Path to config file
	Cause      error  // Underlying error
	Code       ErrorCode
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error [%s]: %s at %s: %v", 
		e.Code, e.ConfigType, e.FilePath, e.Cause)
}

func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// ValidationError represents input validation errors
type ValidationError struct {
	Field   string // Field that failed validation
	Value   string // Invalid value
	Message string // Human-readable message
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s (field: %s, value: %s)", 
		e.Message, e.Field, e.Value)
}

// ConflictError represents sync conflicts
type ConflictError struct {
	ItemName     string    // Name of the conflicted item
	LocalPath    string    // Local file path
	CloudPath    string    // Cloud file path
	LocalHash    string    // Hash of local file
	CloudHash    string    // Hash of cloud file
	ConflictType ConflictType
}

type ConflictType string

const (
	ConflictBothModified ConflictType = "BOTH_MODIFIED"
	ConflictSameTime     ConflictType = "SAME_TIME_DIFFERENT_CONTENT"
	ConflictHashMismatch ConflictType = "HASH_MISMATCH"
)

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict error [%s]: item '%s' has conflicts between local (%s) and cloud (%s)", 
		e.ConflictType, e.ItemName, e.LocalPath, e.CloudPath)
}

// ErrorCollector collects multiple errors and provides batch error handling
type ErrorCollector struct {
	errors []error
}

func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return "no errors"
	}
	
	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}
	
	var messages []string
	for i, err := range ec.errors {
		messages = append(messages, fmt.Sprintf("%d: %s", i+1, err.Error()))
	}
	
	return fmt.Sprintf("multiple errors (%d): %s", len(ec.errors), strings.Join(messages, "; "))
}

// Constructor functions for common errors

func NewSyncError(operation, itemName, filePath string, cause error, code ErrorCode) *SyncError {
	return &SyncError{
		Operation: operation,
		ItemName:  itemName,
		FilePath:  filePath,
		Cause:     cause,
		Code:      code,
	}
}

func NewConfigError(configType, filePath string, cause error, code ErrorCode) *ConfigError {
	return &ConfigError{
		ConfigType: configType,
		FilePath:   filePath,
		Cause:      cause,
		Code:       code,
	}
}

func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

func NewConflictError(itemName, localPath, cloudPath, localHash, cloudHash string, conflictType ConflictType) *ConflictError {
	return &ConflictError{
		ItemName:     itemName,
		LocalPath:    localPath,
		CloudPath:    cloudPath,
		LocalHash:    localHash,
		CloudHash:    cloudHash,
		ConflictType: conflictType,
	}
}

// Error checking utilities

func IsFileNotFoundError(err error) bool {
	var syncErr *SyncError
	if As(err, &syncErr) {
		return syncErr.Code == ErrCodeFileNotFound
	}
	return false
}

func IsPermissionError(err error) bool {
	var syncErr *SyncError
	if As(err, &syncErr) {
		return syncErr.Code == ErrCodePermissionDenied
	}
	return false
}

func IsConflictError(err error) bool {
	var conflictErr *ConflictError
	return As(err, &conflictErr)
}

func IsConfigError(err error) bool {
	var configErr *ConfigError
	return As(err, &configErr)
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return As(err, &validationErr)
}

// As is a wrapper around errors.As for convenience
func As(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	
	// Simple type assertion approach
	switch target := target.(type) {
	case **SyncError:
		if syncErr, ok := err.(*SyncError); ok {
			*target = syncErr
			return true
		}
	case **ConfigError:
		if configErr, ok := err.(*ConfigError); ok {
			*target = configErr
			return true
		}
	case **ValidationError:
		if validationErr, ok := err.(*ValidationError); ok {
			*target = validationErr
			return true
		}
	case **ConflictError:
		if conflictErr, ok := err.(*ConflictError); ok {
			*target = conflictErr
			return true
		}
	}
	
	return false
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted additional context
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}