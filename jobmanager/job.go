package jobmanager

import (
	"fmt"
	"sync"
	"time"
)

// JobID is a unique identifier for a job
type JobID int64

// JobStatus represents the current status of a job
type JobStatus string

const (
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job represents a background task
type Job struct {
	ID        JobID        // Unique job identifier
	Status    JobStatus    // Current status of the job
	Command   string       // Command string that was executed
	Result    interface{}  // Result of the execution (if any)
	Error     error        // Error that occurred during execution (if any)
	StartTime time.Time    // When the job started
	EndTime   time.Time    // When the job ended (zero value if still running)
	mu        sync.RWMutex // For thread-safe access to job fields
}

// NewJob creates a new job with the given command and ID
func NewJob(id JobID, command string) *Job {
	return &Job{
		ID:        id,
		Status:    StatusRunning,
		Command:   command,
		StartTime: time.Now(),
	}
}

// GetStatus returns the current status of the job
func (j *Job) GetStatus() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Status
}

// SetStatus updates the status of the job
func (j *Job) SetStatus(status JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	if status == StatusCompleted || status == StatusFailed {
		j.EndTime = time.Now()
	}
}

// GetResult returns the result of the job
func (j *Job) GetResult() interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Result
}

// SetResult sets the result of the job
func (j *Job) SetResult(result interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Result = result
}

// SetEndTime sets the end time of the job (for testing purposes)
func (j *Job) SetEndTime(endTime time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.EndTime = endTime
}

// GetError returns the error of the job
func (j *Job) GetError() error {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Error
}

// SetError sets the error of the job and marks it as failed
func (j *Job) SetError(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Error = err
	j.Status = StatusFailed
	j.EndTime = time.Now()
}

// GetDuration returns the duration of the job
func (j *Job) GetDuration() time.Duration {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if j.EndTime.IsZero() {
		return time.Since(j.StartTime)
	}
	return j.EndTime.Sub(j.StartTime)
}

// ToMap returns a map representation of the job for serialization
func (j *Job) ToMap() map[string]interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()

	result := map[string]interface{}{
		"id":         j.ID,
		"status":     j.Status,
		"command":    j.Command,
		"start_time": j.StartTime.Format(time.RFC3339),
	}

	if !j.EndTime.IsZero() {
		result["end_time"] = j.EndTime.Format(time.RFC3339)
		result["duration"] = j.GetDuration().String()
	}

	if j.Result != nil {
		result["result"] = j.Result
	}

	if j.Error != nil {
		result["error"] = j.Error.Error()
	}

	return result
}

// String returns a string representation of the job
func (j *Job) String() string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	duration := "running"
	if !j.EndTime.IsZero() {
		duration = j.GetDuration().String()
	}

	return fmt.Sprintf("Job[%d] %s - %s (%s)", j.ID, j.Command, j.Status, duration)
}
