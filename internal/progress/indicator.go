// Package progress provides progress indication and reporting capabilities
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Indicator represents a progress indicator
type Indicator interface {
	Start()
	Update(current, total int)
	SetMessage(message string)
	Finish()
	Close()
}

// BarIndicator displays a progress bar
type BarIndicator struct {
	writer      io.Writer
	width       int
	current     int
	total       int
	message     string
	startTime   time.Time
	lastUpdate  time.Time
	finished    bool
	mutex       sync.RWMutex
	showPercent bool
	showETA     bool
	showSpeed   bool
	prefix      string
	suffix      string
}

// NewBarIndicator creates a new progress bar indicator
func NewBarIndicator(total int, options ...BarOption) *BarIndicator {
	bar := &BarIndicator{
		writer:      os.Stdout,
		width:       50,
		total:       total,
		showPercent: true,
		showETA:     true,
		showSpeed:   true,
		prefix:      "",
		suffix:      "",
	}

	// Apply options
	for _, option := range options {
		option(bar)
	}

	return bar
}

// BarOption represents a configuration option for the progress bar
type BarOption func(*BarIndicator)

// WithWriter sets the output writer
func WithWriter(w io.Writer) BarOption {
	return func(b *BarIndicator) {
		b.writer = w
	}
}

// WithWidth sets the progress bar width
func WithWidth(width int) BarOption {
	return func(b *BarIndicator) {
		b.width = width
	}
}

// WithShowPercent controls percentage display
func WithShowPercent(show bool) BarOption {
	return func(b *BarIndicator) {
		b.showPercent = show
	}
}

// WithShowETA controls ETA display
func WithShowETA(show bool) BarOption {
	return func(b *BarIndicator) {
		b.showETA = show
	}
}

// WithShowSpeed controls speed display
func WithShowSpeed(show bool) BarOption {
	return func(b *BarIndicator) {
		b.showSpeed = show
	}
}

// WithPrefix sets a prefix for the progress bar
func WithPrefix(prefix string) BarOption {
	return func(b *BarIndicator) {
		b.prefix = prefix
	}
}

// WithSuffix sets a suffix for the progress bar
func WithSuffix(suffix string) BarOption {
	return func(b *BarIndicator) {
		b.suffix = suffix
	}
}

// Start starts the progress indicator
func (b *BarIndicator) Start() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.startTime = time.Now()
	b.lastUpdate = b.startTime
	b.finished = false
	b.render()
}

// Update updates the progress
func (b *BarIndicator) Update(current, total int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.current = current
	if total > 0 {
		b.total = total
	}
	b.lastUpdate = time.Now()

	if !b.finished {
		b.render()
	}
}

// SetMessage sets the progress message
func (b *BarIndicator) SetMessage(message string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.message = message
	if !b.finished {
		b.render()
	}
}

// Finish marks the progress as complete
func (b *BarIndicator) Finish() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.current = b.total
	b.finished = true
	b.render()
	fmt.Fprint(b.writer, "\n")
}

// Close cleans up the progress indicator
func (b *BarIndicator) Close() {
	if !b.finished {
		b.Finish()
	}
}

// render renders the progress bar
func (b *BarIndicator) render() {
	// Clear the line
	fmt.Fprint(b.writer, "\r\033[K")

	// Build the progress bar
	var parts []string

	// Add prefix
	if b.prefix != "" {
		parts = append(parts, b.prefix)
	}

	// Progress bar
	var percentage float64
	if b.total > 0 {
		percentage = float64(b.current) / float64(b.total)
	}

	completed := int(float64(b.width) * percentage)
	remaining := b.width - completed

	bar := fmt.Sprintf("[%s%s]",
		strings.Repeat("█", completed),
		strings.Repeat("░", remaining))
	parts = append(parts, bar)

	// Progress numbers
	parts = append(parts, fmt.Sprintf("%d/%d", b.current, b.total))

	// Percentage
	if b.showPercent {
		parts = append(parts, fmt.Sprintf("%.1f%%", percentage*100))
	}

	// Speed
	if b.showSpeed && time.Since(b.startTime) > 0 {
		elapsed := time.Since(b.startTime)
		speed := float64(b.current) / elapsed.Seconds()
		parts = append(parts, fmt.Sprintf("%.1f/s", speed))
	}

	// ETA
	if b.showETA && b.current > 0 && percentage < 1.0 {
		elapsed := time.Since(b.startTime)
		avgTimePerItem := elapsed / time.Duration(b.current)
		remaining := time.Duration(b.total-b.current) * avgTimePerItem
		parts = append(parts, fmt.Sprintf("ETA: %s", formatDuration(remaining)))
	}

	// Message
	if b.message != "" {
		parts = append(parts, b.message)
	}

	// Add suffix
	if b.suffix != "" {
		parts = append(parts, b.suffix)
	}

	// Print the progress bar
	fmt.Fprint(b.writer, strings.Join(parts, " "))
}

// SpinnerIndicator displays a spinning indicator
type SpinnerIndicator struct {
	writer   io.Writer
	frames   []string
	current  int
	message  string
	ticker   *time.Ticker
	done     chan bool
	finished bool
	mutex    sync.RWMutex
}

// NewSpinnerIndicator creates a new spinner indicator
func NewSpinnerIndicator(options ...SpinnerOption) *SpinnerIndicator {
	spinner := &SpinnerIndicator{
		writer: os.Stdout,
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:   make(chan bool, 1),
	}

	// Apply options
	for _, option := range options {
		option(spinner)
	}

	return spinner
}

// SpinnerOption represents a configuration option for the spinner
type SpinnerOption func(*SpinnerIndicator)

// WithSpinnerWriter sets the output writer
func WithSpinnerWriter(w io.Writer) SpinnerOption {
	return func(s *SpinnerIndicator) {
		s.writer = w
	}
}

// WithSpinnerFrames sets custom spinner frames
func WithSpinnerFrames(frames []string) SpinnerOption {
	return func(s *SpinnerIndicator) {
		s.frames = frames
	}
}

// Start starts the spinner
func (s *SpinnerIndicator) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.finished = false
	s.ticker = time.NewTicker(100 * time.Millisecond)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.mutex.Lock()
				if !s.finished {
					s.render()
					s.current = (s.current + 1) % len(s.frames)
				}
				s.mutex.Unlock()
			case <-s.done:
				return
			}
		}
	}()
}

// Update updates the spinner (no-op for spinner)
func (s *SpinnerIndicator) Update(current, total int) {
	// Spinners don't show specific progress
}

// SetMessage sets the spinner message
func (s *SpinnerIndicator) SetMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.message = message
}

// Finish stops the spinner
func (s *SpinnerIndicator) Finish() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.finished = true
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true

	// Clear the line and print completion
	fmt.Fprint(s.writer, "\r\033[K✓")
	if s.message != "" {
		fmt.Fprintf(s.writer, " %s", s.message)
	}
	fmt.Fprint(s.writer, "\n")
}

// Close cleans up the spinner
func (s *SpinnerIndicator) Close() {
	if !s.finished {
		s.Finish()
	}
}

// render renders the spinner
func (s *SpinnerIndicator) render() {
	fmt.Fprint(s.writer, "\r\033[K")
	fmt.Fprint(s.writer, s.frames[s.current])
	if s.message != "" {
		fmt.Fprintf(s.writer, " %s", s.message)
	}
}

// MultiIndicator manages multiple progress indicators
type MultiIndicator struct {
	indicators []Indicator
	mutex      sync.RWMutex
}

// NewMultiIndicator creates a new multi-indicator
func NewMultiIndicator() *MultiIndicator {
	return &MultiIndicator{
		indicators: make([]Indicator, 0),
	}
}

// Add adds an indicator to the multi-indicator
func (m *MultiIndicator) Add(indicator Indicator) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.indicators = append(m.indicators, indicator)
}

// Start starts all indicators
func (m *MultiIndicator) Start() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, indicator := range m.indicators {
		indicator.Start()
	}
}

// Update updates all indicators
func (m *MultiIndicator) Update(current, total int) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, indicator := range m.indicators {
		indicator.Update(current, total)
	}
}

// SetMessage sets the message for all indicators
func (m *MultiIndicator) SetMessage(message string) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, indicator := range m.indicators {
		indicator.SetMessage(message)
	}
}

// Finish finishes all indicators
func (m *MultiIndicator) Finish() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, indicator := range m.indicators {
		indicator.Finish()
	}
}

// Close closes all indicators
func (m *MultiIndicator) Close() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, indicator := range m.indicators {
		indicator.Close()
	}
}

// ProgressTracker tracks operation progress with callbacks
type ProgressTracker struct {
	total     int
	current   int
	message   string
	startTime time.Time
	callbacks []func(current, total int, message string)
	mutex     sync.RWMutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		startTime: time.Now(),
		callbacks: make([]func(current, total int, message string), 0),
	}
}

// AddCallback adds a progress callback
func (pt *ProgressTracker) AddCallback(callback func(current, total int, message string)) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.callbacks = append(pt.callbacks, callback)
}

// Update updates the progress and notifies callbacks
func (pt *ProgressTracker) Update(current int, message string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.current = current
	pt.message = message

	// Notify all callbacks
	for _, callback := range pt.callbacks {
		go callback(pt.current, pt.total, pt.message)
	}
}

// Increment increments the progress by 1
func (pt *ProgressTracker) Increment(message string) {
	pt.Update(pt.current+1, message)
}

// Progress returns current progress information
func (pt *ProgressTracker) Progress() (current, total int, message string, elapsed time.Duration) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	return pt.current, pt.total, pt.message, time.Since(pt.startTime)
}

// IsComplete returns true if progress is complete
func (pt *ProgressTracker) IsComplete() bool {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	return pt.current >= pt.total
}

// Percentage returns completion percentage
func (pt *ProgressTracker) Percentage() float64 {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	if pt.total == 0 {
		return 0
	}

	return float64(pt.current) / float64(pt.total) * 100
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-60*d.Minutes())
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) - hours*60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}

// NoOpIndicator is a progress indicator that does nothing (for testing or quiet mode)
type NoOpIndicator struct{}

func NewNoOpIndicator() *NoOpIndicator {
	return &NoOpIndicator{}
}

func (n *NoOpIndicator) Start()                      {}
func (n *NoOpIndicator) Update(current, total int)  {}
func (n *NoOpIndicator) SetMessage(message string)  {}
func (n *NoOpIndicator) Finish()                    {}
func (n *NoOpIndicator) Close()                     {}

// Global progress management
var (
	globalIndicator Indicator
	globalMutex     sync.RWMutex
)

// SetGlobalIndicator sets the global progress indicator
func SetGlobalIndicator(indicator Indicator) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalIndicator != nil {
		globalIndicator.Close()
	}
	globalIndicator = indicator
}

// UpdateGlobalProgress updates the global progress indicator
func UpdateGlobalProgress(current, total int) {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	if globalIndicator != nil {
		globalIndicator.Update(current, total)
	}
}

// SetGlobalMessage sets the global progress message
func SetGlobalMessage(message string) {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	if globalIndicator != nil {
		globalIndicator.SetMessage(message)
	}
}

// StartGlobalProgress starts the global progress indicator
func StartGlobalProgress() {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	if globalIndicator != nil {
		globalIndicator.Start()
	}
}

// FinishGlobalProgress finishes the global progress indicator
func FinishGlobalProgress() {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	if globalIndicator != nil {
		globalIndicator.Finish()
	}
}