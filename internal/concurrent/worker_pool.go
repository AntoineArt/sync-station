// Package concurrent provides utilities for concurrent operations in syncstation
package concurrent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Task represents a unit of work to be executed
type Task interface {
	Execute(ctx context.Context) error
	ID() string
	Priority() int // Higher number = higher priority
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID    string
	Error     error
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// WorkerPool manages a pool of workers that execute tasks concurrently
type WorkerPool struct {
	workers     int
	taskQueue   chan Task
	resultQueue chan TaskResult
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	statistics  *PoolStatistics
}

// PoolStatistics tracks worker pool performance metrics
type PoolStatistics struct {
	mutex           sync.RWMutex
	TasksSubmitted  int64         `json:"tasksSubmitted"`
	TasksCompleted  int64         `json:"tasksCompleted"`
	TasksFailed     int64         `json:"tasksFailed"`
	TotalDuration   time.Duration `json:"totalDuration"`
	AverageDuration time.Duration `json:"averageDuration"`
	WorkersActive   int           `json:"workersActive"`
	WorkersIdle     int           `json:"workersIdle"`
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if queueSize <= 0 {
		queueSize = workers * 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers:     workers,
		taskQueue:   make(chan Task, queueSize),
		resultQueue: make(chan TaskResult, queueSize),
		ctx:         ctx,
		cancel:      cancel,
		statistics:  &PoolStatistics{},
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool and waits for all workers to finish
func (wp *WorkerPool) Stop() {
	close(wp.taskQueue)
	wp.wg.Wait()
	wp.cancel()
	close(wp.resultQueue)
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.taskQueue <- task:
		wp.statistics.mutex.Lock()
		wp.statistics.TasksSubmitted++
		wp.statistics.mutex.Unlock()
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// Results returns a channel that receives task results
func (wp *WorkerPool) Results() <-chan TaskResult {
	return wp.resultQueue
}

// worker processes tasks from the task queue
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return // Channel closed, worker should exit
			}

			wp.statistics.mutex.Lock()
			wp.statistics.WorkersActive++
			wp.statistics.WorkersIdle--
			wp.statistics.mutex.Unlock()

			result := wp.executeTask(task)

			wp.statistics.mutex.Lock()
			wp.statistics.WorkersActive--
			wp.statistics.WorkersIdle++
			wp.statistics.TasksCompleted++
			if result.Error != nil {
				wp.statistics.TasksFailed++
			}
			wp.statistics.TotalDuration += result.Duration
			if wp.statistics.TasksCompleted > 0 {
				wp.statistics.AverageDuration = time.Duration(int64(wp.statistics.TotalDuration) / wp.statistics.TasksCompleted)
			}
			wp.statistics.mutex.Unlock()

			select {
			case wp.resultQueue <- result:
			case <-wp.ctx.Done():
				return
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// executeTask executes a single task and returns the result
func (wp *WorkerPool) executeTask(task Task) TaskResult {
	startTime := time.Now()
	err := task.Execute(wp.ctx)
	endTime := time.Now()

	return TaskResult{
		TaskID:    task.ID(),
		Error:     err,
		Duration:  endTime.Sub(startTime),
		StartTime: startTime,
		EndTime:   endTime,
	}
}

// GetStatistics returns current worker pool statistics
func (wp *WorkerPool) GetStatistics() PoolStatistics {
	wp.statistics.mutex.RLock()
	defer wp.statistics.mutex.RUnlock()

	// Create a copy to avoid race conditions
	return PoolStatistics{
		TasksSubmitted:  wp.statistics.TasksSubmitted,
		TasksCompleted:  wp.statistics.TasksCompleted,
		TasksFailed:     wp.statistics.TasksFailed,
		TotalDuration:   wp.statistics.TotalDuration,
		AverageDuration: wp.statistics.AverageDuration,
		WorkersActive:   wp.statistics.WorkersActive,
		WorkersIdle:     wp.statistics.WorkersIdle,
	}
}

// BatchExecutor executes multiple tasks concurrently and waits for all to complete
type BatchExecutor struct {
	pool    *WorkerPool
	timeout time.Duration
}

// NewBatchExecutor creates a new batch executor
func NewBatchExecutor(workers int, timeout time.Duration) *BatchExecutor {
	return &BatchExecutor{
		pool:    NewWorkerPool(workers, workers*2),
		timeout: timeout,
	}
}

// Execute executes all tasks concurrently and returns results
func (be *BatchExecutor) Execute(tasks []Task) ([]TaskResult, error) {
	if len(tasks) == 0 {
		return []TaskResult{}, nil
	}

	be.pool.Start()
	defer be.pool.Stop()

	// Submit all tasks
	for _, task := range tasks {
		if err := be.pool.Submit(task); err != nil {
			return nil, fmt.Errorf("failed to submit task %s: %w", task.ID(), err)
		}
	}

	// Collect results
	results := make([]TaskResult, 0, len(tasks))
	resultsChannel := be.pool.Results()

	// Set up timeout context
	ctx, cancel := context.WithTimeout(context.Background(), be.timeout)
	defer cancel()

	for i := 0; i < len(tasks); i++ {
		select {
		case result := <-resultsChannel:
			results = append(results, result)
		case <-ctx.Done():
			return results, fmt.Errorf("batch execution timed out after %v", be.timeout)
		}
	}

	return results, nil
}

// SyncTask represents a file synchronization task
type SyncTask struct {
	id        string
	priority  int
	operation func(ctx context.Context) error
}

// NewSyncTask creates a new sync task
func NewSyncTask(id string, priority int, operation func(ctx context.Context) error) *SyncTask {
	return &SyncTask{
		id:        id,
		priority:  priority,
		operation: operation,
	}
}

func (st *SyncTask) Execute(ctx context.Context) error {
	return st.operation(ctx)
}

func (st *SyncTask) ID() string {
	return st.id
}

func (st *SyncTask) Priority() int {
	return st.priority
}

// FileOperationTask represents a file operation task
type FileOperationTask struct {
	id          string
	priority    int
	srcPath     string
	dstPath     string
	operationType string // "copy", "move", "delete", "hash"
	operation   func(ctx context.Context, src, dst string) error
}

// NewFileOperationTask creates a new file operation task
func NewFileOperationTask(id string, priority int, srcPath, dstPath, operationType string, 
	operation func(ctx context.Context, src, dst string) error) *FileOperationTask {
	return &FileOperationTask{
		id:            id,
		priority:      priority,
		srcPath:       srcPath,
		dstPath:       dstPath,
		operationType: operationType,
		operation:     operation,
	}
}

func (fot *FileOperationTask) Execute(ctx context.Context) error {
	return fot.operation(ctx, fot.srcPath, fot.dstPath)
}

func (fot *FileOperationTask) ID() string {
	return fot.id
}

func (fot *FileOperationTask) Priority() int {
	return fot.priority
}

func (fot *FileOperationTask) SrcPath() string {
	return fot.srcPath
}

func (fot *FileOperationTask) DstPath() string {
	return fot.dstPath
}

func (fot *FileOperationTask) OperationType() string {
	return fot.operationType
}

// ProgressTracker tracks progress of concurrent operations
type ProgressTracker struct {
	mutex     sync.RWMutex
	total     int
	completed int
	failed    int
	callbacks []func(completed, total, failed int)
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		callbacks: make([]func(completed, total, failed int), 0),
	}
}

// AddCallback adds a progress callback function
func (pt *ProgressTracker) AddCallback(callback func(completed, total, failed int)) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.callbacks = append(pt.callbacks, callback)
}

// Update updates progress and notifies callbacks
func (pt *ProgressTracker) Update(success bool) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.completed++
	if !success {
		pt.failed++
	}

	// Notify all callbacks
	for _, callback := range pt.callbacks {
		go callback(pt.completed, pt.total, pt.failed)
	}
}

// Progress returns current progress
func (pt *ProgressTracker) Progress() (completed, total, failed int) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()
	return pt.completed, pt.total, pt.failed
}

// IsComplete returns true if all tasks are completed
func (pt *ProgressTracker) IsComplete() bool {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()
	return pt.completed >= pt.total
}

// Success rate returns the success rate as a percentage
func (pt *ProgressTracker) SuccessRate() float64 {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	if pt.completed == 0 {
		return 0
	}

	successful := pt.completed - pt.failed
	return (float64(successful) / float64(pt.completed)) * 100
}