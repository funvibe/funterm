package jobmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// JobManager manages the lifecycle of background tasks
type JobManager struct {
	jobs       map[JobID]*Job
	mu         sync.RWMutex
	semaphore  chan struct{}        // For concurrency limiting
	nextID     JobID                // Next job ID
	notifyChan chan JobNotification // Channel for job notifications
	ctx        context.Context      // Context for cancellation
	cancel     context.CancelFunc   // Cancel function for the context
	wg         sync.WaitGroup       // WaitGroup for tracking running jobs
}

// JobNotification represents a notification about a job status change
type JobNotification struct {
	JobID  JobID
	Status JobStatus
	Result interface{}
	Error  error
}

// NewJobManager creates a new JobManager with the specified concurrency limit
func NewJobManager(concurrencyLimit int) *JobManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobManager{
		jobs:       make(map[JobID]*Job),
		semaphore:  make(chan struct{}, concurrencyLimit),
		nextID:     1,
		notifyChan: make(chan JobNotification, 100), // Buffered channel
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Submit submits a job for execution and returns its ID
func (jm *JobManager) Submit(task func() (interface{}, error), command string) (JobID, error) {
	// Check if we've been cancelled
	select {
	case <-jm.ctx.Done():
		return 0, errors.New("job manager is shutting down")
	default:
	}

	// Try to acquire a semaphore slot (enforce concurrency limit)
	select {
	case jm.semaphore <- struct{}{}:
		// Got a slot, proceed
	default:
		// No slots available
		return 0, errors.New("concurrency limit reached, cannot submit more jobs")
	}

	// Generate a new job ID
	jm.mu.Lock()
	jobID := jm.nextID
	jm.nextID++
	jm.mu.Unlock()

	// Create a new job
	job := NewJob(jobID, command)

	// Store the job
	jm.mu.Lock()
	jm.jobs[jobID] = job
	jm.mu.Unlock()

	// Start the job in a goroutine
	jm.wg.Add(1)
	go jm.executeJob(job, task)

	return jobID, nil
}

// executeJob executes a job and handles its lifecycle
func (jm *JobManager) executeJob(job *Job, task func() (interface{}, error)) {
	defer jm.wg.Done()
	defer func() {
		// Release the semaphore slot
		<-jm.semaphore
	}()

	// Execute the task
	result, err := task()

	// Update job status based on execution result
	if err != nil {
		job.SetError(err)
		// Send failure notification
		select {
		case jm.notifyChan <- JobNotification{
			JobID:  job.ID,
			Status: StatusFailed,
			Result: nil,
			Error:  err,
		}:
		case <-jm.ctx.Done():
			// Manager is shutting down
		}
	} else {
		job.SetResult(result)
		job.SetStatus(StatusCompleted)
		// Send completion notification
		select {
		case jm.notifyChan <- JobNotification{
			JobID:  job.ID,
			Status: StatusCompleted,
			Result: result,
			Error:  nil,
		}:
		case <-jm.ctx.Done():
			// Manager is shutting down
		}
	}
}

// GetJobStatus returns the status of a specific job
func (jm *JobManager) GetJobStatus(id JobID) (JobStatus, error) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobs[id]
	if !exists {
		return "", fmt.Errorf("job with ID %d not found", id)
	}

	return job.GetStatus(), nil
}

// GetJob returns a specific job
func (jm *JobManager) GetJob(id JobID) (*Job, error) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job with ID %d not found", id)
	}

	return job, nil
}

// ListJobs returns all jobs
func (jm *JobManager) ListJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// GetNotificationChannel returns the channel for job notifications
func (jm *JobManager) GetNotificationChannel() <-chan JobNotification {
	return jm.notifyChan
}

// Shutdown gracefully shuts down the job manager
func (jm *JobManager) Shutdown() {
	// Cancel the context to signal shutdown
	jm.cancel()

	// Wait for all running jobs to complete
	jm.wg.Wait()

	// Close the notification channel
	close(jm.notifyChan)
}

// GetRunningJobsCount returns the number of currently running jobs
func (jm *JobManager) GetRunningJobsCount() int {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	count := 0
	for _, job := range jm.jobs {
		if job.GetStatus() == StatusRunning {
			count++
		}
	}

	return count
}

// GetConcurrencyLimit returns the current concurrency limit
func (jm *JobManager) GetConcurrencyLimit() int {
	return cap(jm.semaphore)
}

// SetConcurrencyLimit changes the concurrency limit
// Note: This doesn't affect already running jobs
func (jm *JobManager) SetConcurrencyLimit(limit int) {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	// Create a new semaphore with the new limit
	newSemaphore := make(chan struct{}, limit)

	// Try to fill the new semaphore with as many slots as possible
	// from the old semaphore without blocking
	oldSemaphore := jm.semaphore
	for i := 0; i < limit; i++ {
		select {
		case <-oldSemaphore:
			newSemaphore <- struct{}{}
		default:
			// No more slots available in old semaphore
			break
		}
	}

	jm.semaphore = newSemaphore
}

// CancelJob cancels a running job by ID
// Note: This is a best-effort cancellation and may not work for all types of tasks
func (jm *JobManager) CancelJob(id JobID) error {
	jm.mu.RLock()
	job, exists := jm.jobs[id]
	jm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job with ID %d not found", id)
	}

	if job.GetStatus() != StatusRunning {
		return fmt.Errorf("job %d is not running (status: %s)", id, job.GetStatus())
	}

	// Mark the job as failed with a cancellation error
	job.SetError(fmt.Errorf("job cancelled by user"))

	// Send cancellation notification
	select {
	case jm.notifyChan <- JobNotification{
		JobID:  id,
		Status: StatusFailed,
		Result: nil,
		Error:  fmt.Errorf("job cancelled by user"),
	}:
	default:
		// Channel is full or closed
	}

	return nil
}

// CleanCompletedJobs removes completed and failed jobs older than the specified duration
func (jm *JobManager) CleanCompletedJobs(olderThan time.Duration) int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	for id, job := range jm.jobs {
		if job.GetStatus() == StatusCompleted || job.GetStatus() == StatusFailed {
			if job.EndTime.Before(cutoff) {
				delete(jm.jobs, id)
				removed++
			}
		}
	}

	return removed
}
