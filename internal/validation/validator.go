// Package validation provides input validation and security utilities
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/AntoineArt/syncstation/internal/errors"
)

// PathValidator validates file and directory paths
type PathValidator struct {
	allowedRoots    []string
	blockedPaths    []string
	maxPathLength   int
	allowSymlinks   bool
	allowHiddenFiles bool
}

// NewPathValidator creates a new path validator with default settings
func NewPathValidator() *PathValidator {
	return &PathValidator{
		allowedRoots:     []string{}, // Empty means allow all roots
		blockedPaths:     []string{"/etc/passwd", "/etc/shadow", "/etc/hosts"},
		maxPathLength:    4096,
		allowSymlinks:    false,
		allowHiddenFiles: false,
	}
}

// WithAllowedRoots sets the allowed root directories
func (pv *PathValidator) WithAllowedRoots(roots []string) *PathValidator {
	pv.allowedRoots = roots
	return pv
}

// WithBlockedPaths sets paths that should be blocked
func (pv *PathValidator) WithBlockedPaths(paths []string) *PathValidator {
	pv.blockedPaths = paths
	return pv
}

// WithMaxPathLength sets the maximum allowed path length
func (pv *PathValidator) WithMaxPathLength(length int) *PathValidator {
	pv.maxPathLength = length
	return pv
}

// WithAllowSymlinks enables or disables symlink following
func (pv *PathValidator) WithAllowSymlinks(allow bool) *PathValidator {
	pv.allowSymlinks = allow
	return pv
}

// WithAllowHiddenFiles enables or disables hidden file access
func (pv *PathValidator) WithAllowHiddenFiles(allow bool) *PathValidator {
	pv.allowHiddenFiles = allow
	return pv
}

// ValidatePath validates a file or directory path
func (pv *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return errors.NewValidationError("path", path, "path cannot be empty")
	}

	// Check path length
	if len(path) > pv.maxPathLength {
		return errors.NewValidationError("path", path, 
			fmt.Sprintf("path length exceeds maximum of %d characters", pv.maxPathLength))
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return errors.NewValidationError("path", path, 
			fmt.Sprintf("failed to resolve absolute path: %v", err))
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return errors.NewValidationError("path", path, "path traversal detected")
	}

	// Check for null bytes (security risk)
	if strings.Contains(path, "\x00") {
		return errors.NewValidationError("path", path, "null bytes not allowed in path")
	}

	// Check against blocked paths
	for _, blocked := range pv.blockedPaths {
		if strings.HasPrefix(absPath, blocked) || absPath == blocked {
			return errors.NewValidationError("path", path, "access to this path is blocked")
		}
	}

	// Check allowed roots if specified
	if len(pv.allowedRoots) > 0 {
		allowed := false
		for _, root := range pv.allowedRoots {
			absRoot, err := filepath.Abs(root)
			if err != nil {
				continue
			}
			if strings.HasPrefix(absPath, absRoot) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.NewValidationError("path", path, "path is outside allowed directories")
		}
	}

	// Check for hidden files if not allowed
	if !pv.allowHiddenFiles {
		parts := strings.Split(cleanPath, string(filepath.Separator))
		for _, part := range parts {
			if strings.HasPrefix(part, ".") && part != "." && part != ".." {
				return errors.NewValidationError("path", path, "hidden files are not allowed")
			}
		}
	}

	// Check for symlinks if not allowed
	if !pv.allowSymlinks {
		if err := pv.checkSymlinks(absPath); err != nil {
			return err
		}
	}

	return nil
}

// checkSymlinks checks if path contains symlinks
func (pv *PathValidator) checkSymlinks(path string) error {
	// Walk up the path and check each component
	current := path
	for current != filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				// Path doesn't exist yet, continue checking parent
				current = filepath.Dir(current)
				continue
			}
			return errors.NewValidationError("path", path, 
				fmt.Sprintf("failed to check path: %v", err))
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return errors.NewValidationError("path", path, "symlinks are not allowed")
		}

		current = filepath.Dir(current)
	}
	return nil
}

// NameValidator validates sync item names and other identifiers
type NameValidator struct {
	maxLength       int
	allowedChars    *regexp.Regexp
	reservedNames   []string
	caseSensitive   bool
}

// NewNameValidator creates a new name validator
func NewNameValidator() *NameValidator {
	// Allow alphanumeric, spaces, hyphens, underscores, and dots
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.]+$`)
	
	return &NameValidator{
		maxLength:     100,
		allowedChars:  allowedChars,
		reservedNames: []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", 
			"COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", 
			"LPT8", "LPT9"}, // Windows reserved names
		caseSensitive: false,
	}
}

// ValidateName validates a sync item name or identifier
func (nv *NameValidator) ValidateName(name string) error {
	if name == "" {
		return errors.NewValidationError("name", name, "name cannot be empty")
	}

	// Check length
	if len(name) > nv.maxLength {
		return errors.NewValidationError("name", name, 
			fmt.Sprintf("name length exceeds maximum of %d characters", nv.maxLength))
	}

	// Check allowed characters
	if !nv.allowedChars.MatchString(name) {
		return errors.NewValidationError("name", name, 
			"name contains invalid characters. Only letters, numbers, spaces, hyphens, underscores, and dots are allowed")
	}

	// Check for leading/trailing whitespace
	if strings.TrimSpace(name) != name {
		return errors.NewValidationError("name", name, "name cannot have leading or trailing whitespace")
	}

	// Check for reserved names
	checkName := name
	if !nv.caseSensitive {
		checkName = strings.ToUpper(name)
	}
	
	for _, reserved := range nv.reservedNames {
		if checkName == reserved {
			return errors.NewValidationError("name", name, 
				fmt.Sprintf("'%s' is a reserved name and cannot be used", name))
		}
	}

	// Check for consecutive spaces
	if strings.Contains(name, "  ") {
		return errors.NewValidationError("name", name, "name cannot contain consecutive spaces")
	}

	return nil
}

// ConfigValidator validates configuration values
type ConfigValidator struct {
	pathValidator *PathValidator
	nameValidator *NameValidator
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		pathValidator: NewPathValidator(),
		nameValidator: NewNameValidator(),
	}
}

// ValidateComputerID validates a computer identifier
func (cv *ConfigValidator) ValidateComputerID(computerID string) error {
	if computerID == "" {
		return errors.NewValidationError("computer_id", computerID, "computer ID cannot be empty")
	}

	// Check length (reasonable limit for computer names)
	if len(computerID) > 63 {
		return errors.NewValidationError("computer_id", computerID, 
			"computer ID cannot exceed 63 characters")
	}

	// Check for valid hostname characters
	validHostname := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
	if !validHostname.MatchString(computerID) {
		return errors.NewValidationError("computer_id", computerID, 
			"computer ID must be a valid hostname (letters, numbers, hyphens, no leading/trailing hyphens)")
	}

	return nil
}

// ValidateSyncItemName validates a sync item name
func (cv *ConfigValidator) ValidateSyncItemName(name string) error {
	return cv.nameValidator.ValidateName(name)
}

// ValidateLocalPath validates a local file path
func (cv *ConfigValidator) ValidateLocalPath(path string) error {
	return cv.pathValidator.ValidatePath(path)
}

// ValidateCloudPath validates a cloud storage path
func (cv *ConfigValidator) ValidateCloudPath(path string) error {
	// Cloud paths have additional restrictions
	cloudValidator := NewPathValidator().
		WithAllowHiddenFiles(false). // Don't allow hidden files in cloud
		WithAllowSymlinks(false)     // Don't allow symlinks in cloud
	
	return cloudValidator.ValidatePath(path)
}

// ValidateExcludePatterns validates exclude patterns
func (cv *ConfigValidator) ValidateExcludePatterns(patterns []string) error {
	for i, pattern := range patterns {
		if err := cv.validateExcludePattern(pattern); err != nil {
			return errors.NewValidationError("exclude_pattern", pattern, 
				fmt.Sprintf("invalid exclude pattern at index %d: %v", i, err))
		}
	}
	return nil
}

// validateExcludePattern validates a single exclude pattern
func (cv *ConfigValidator) validateExcludePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("exclude pattern cannot be empty")
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{"*", "**", "/", "/*", "/**"}
	for _, dangerous := range dangerousPatterns {
		if pattern == dangerous {
			return fmt.Errorf("pattern '%s' is too broad and would exclude everything", pattern)
		}
	}

	// Check for path traversal in patterns
	if strings.Contains(pattern, "..") {
		return fmt.Errorf("path traversal not allowed in exclude patterns")
	}

	// Validate that it's a reasonable glob pattern
	_, err := filepath.Match(pattern, "test")
	if err != nil {
		return fmt.Errorf("invalid glob pattern: %v", err)
	}

	return nil
}

// SanitizerOptions configures input sanitization
type SanitizerOptions struct {
	RemoveControlChars bool
	NormalizeUnicode   bool
	TrimWhitespace     bool
	MaxLength          int
}

// InputSanitizer sanitizes user input
type InputSanitizer struct {
	options SanitizerOptions
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer(options SanitizerOptions) *InputSanitizer {
	return &InputSanitizer{
		options: options,
	}
}

// SanitizeString sanitizes a string input
func (is *InputSanitizer) SanitizeString(input string) string {
	result := input

	// Remove control characters
	if is.options.RemoveControlChars {
		result = is.removeControlChars(result)
	}

	// Trim whitespace
	if is.options.TrimWhitespace {
		result = strings.TrimSpace(result)
	}

	// Truncate if too long
	if is.options.MaxLength > 0 && len(result) > is.options.MaxLength {
		result = result[:is.options.MaxLength]
	}

	return result
}

// removeControlChars removes control characters from a string
func (is *InputSanitizer) removeControlChars(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1 // Remove the character
		}
		return r
	}, input)
}

// SecurityChecker provides security-related validation
type SecurityChecker struct {
	suspiciousPatterns []*regexp.Regexp
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker() *SecurityChecker {
	// Patterns that might indicate malicious input
	suspiciousPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{.*\}`),                    // Variable substitution
		regexp.MustCompile(`\$\(.*\)`),                    // Command substitution
		regexp.MustCompile("`.*`"),                        // Backtick execution
		regexp.MustCompile(`^\s*\|`),                      // Pipe at start
		regexp.MustCompile(`[;&|><]\s*\w`),                // Command chaining
		regexp.MustCompile(`\b(rm|del|format|mkfs)\b`),    // Dangerous commands
		regexp.MustCompile(`\b(eval|exec|system)\b`),      // Code execution
	}

	return &SecurityChecker{
		suspiciousPatterns: suspiciousPatterns,
	}
}

// CheckForSuspiciousContent checks input for potentially malicious content
func (sc *SecurityChecker) CheckForSuspiciousContent(input string) error {
	for _, pattern := range sc.suspiciousPatterns {
		if pattern.MatchString(input) {
			return errors.NewValidationError("security", input, 
				"input contains potentially malicious content")
		}
	}
	return nil
}

// Default validators for common use
var (
	DefaultPathValidator   = NewPathValidator()
	DefaultNameValidator   = NewNameValidator()
	DefaultConfigValidator = NewConfigValidator()
	DefaultSecurityChecker = NewSecurityChecker()
	DefaultInputSanitizer  = NewInputSanitizer(SanitizerOptions{
		RemoveControlChars: true,
		NormalizeUnicode:   true,
		TrimWhitespace:     true,
		MaxLength:          1000,
	})
)

// Convenience functions using default validators

// ValidatePath validates a path using the default validator
func ValidatePath(path string) error {
	return DefaultPathValidator.ValidatePath(path)
}

// ValidateName validates a name using the default validator
func ValidateName(name string) error {
	return DefaultNameValidator.ValidateName(name)
}

// ValidateComputerID validates a computer ID using the default validator
func ValidateComputerID(computerID string) error {
	return DefaultConfigValidator.ValidateComputerID(computerID)
}

// ValidateSyncItemName validates a sync item name using the default validator
func ValidateSyncItemName(name string) error {
	return DefaultConfigValidator.ValidateSyncItemName(name)
}

// SanitizeInput sanitizes input using the default sanitizer
func SanitizeInput(input string) string {
	return DefaultInputSanitizer.SanitizeString(input)
}

// CheckSecurity checks input for security issues using the default checker
func CheckSecurity(input string) error {
	return DefaultSecurityChecker.CheckForSuspiciousContent(input)
}