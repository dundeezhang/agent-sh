package shell

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// JobStatus represents the current state of a job.
type JobStatus int

const (
	// JobRunning means the job is executing.
	JobRunning JobStatus = iota
	// JobStopped means the job has been suspended (e.g. via Ctrl+Z).
	JobStopped
	// JobDone means the job has finished.
	JobDone
)

// String returns a human-readable label for a JobStatus.
func (s JobStatus) String() string {
	switch s {
	case JobRunning:
		return "Running"
	case JobStopped:
		return "Stopped"
	case JobDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// Job represents a tracked process in the shell's job table.
type Job struct {
	ID      int
	Command string
	Cmd     *exec.Cmd
	Status  JobStatus
}

// Pid returns the process ID of the job, or 0 if unavailable.
func (j *Job) Pid() int {
	if j.Cmd != nil && j.Cmd.Process != nil {
		return j.Cmd.Process.Pid
	}
	return 0
}

// JobTable tracks background and suspended processes.
type JobTable struct {
	mu     sync.Mutex
	jobs   map[int]*Job
	nextID int
	// current holds the job number of the foreground process while one is
	// running, so the SIGTSTP handler knows which job to stop. Zero means
	// no foreground job.
	current int
}

// NewJobTable creates an empty job table.
func NewJobTable() *JobTable {
	return &JobTable{
		jobs:   make(map[int]*Job),
		nextID: 1,
	}
}

// Add registers a new job and returns it.
func (jt *JobTable) Add(command string, cmd *exec.Cmd, status JobStatus) *Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	j := &Job{
		ID:      jt.nextID,
		Command: command,
		Cmd:     cmd,
		Status:  status,
	}
	jt.jobs[j.ID] = j
	jt.nextID++
	return j
}

// Remove deletes a job from the table.
func (jt *JobTable) Remove(id int) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	delete(jt.jobs, id)
}

// Get returns a job by ID, or nil if not found.
func (jt *JobTable) Get(id int) *Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	return jt.jobs[id]
}

// List returns all jobs sorted by ID.
func (jt *JobTable) List() []*Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	if len(jt.jobs) == 0 {
		return nil
	}

	// Find max ID to iterate in order.
	maxID := 0
	for id := range jt.jobs {
		if id > maxID {
			maxID = id
		}
	}

	result := make([]*Job, 0, len(jt.jobs))
	for i := 1; i <= maxID; i++ {
		if j, ok := jt.jobs[i]; ok {
			result = append(result, j)
		}
	}
	return result
}

// MostRecent returns the most recent job (highest ID) that is either
// stopped or running (not done). Returns nil if no such job exists.
func (jt *JobTable) MostRecent() *Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	var best *Job
	for _, j := range jt.jobs {
		if j.Status == JobDone {
			continue
		}
		if best == nil || j.ID > best.ID {
			best = j
		}
	}
	return best
}

// SetCurrent sets the current foreground job ID. Pass 0 to clear.
func (jt *JobTable) SetCurrent(id int) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jt.current = id
}

// Current returns the current foreground job ID (0 if none).
func (jt *JobTable) Current() int {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	return jt.current
}

// CheckCompleted polls for background jobs that have finished and returns
// them. The caller is responsible for removing them from the table or
// printing notifications.
func (jt *JobTable) CheckCompleted() []*Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	var completed []*Job
	for _, j := range jt.jobs {
		if j.Status != JobRunning {
			continue
		}
		if j.Cmd == nil || j.Cmd.Process == nil {
			continue
		}
		// Non-blocking wait: check if the process has exited.
		var ws syscall.WaitStatus
		pid, err := syscall.Wait4(j.Cmd.Process.Pid, &ws, syscall.WNOHANG, nil)
		if err != nil || pid == 0 {
			continue
		}
		j.Status = JobDone
		completed = append(completed, j)
	}
	return completed
}

// ReportAndClean checks for completed background jobs, prints notifications
// to stderr, and removes them from the table.
func (jt *JobTable) ReportAndClean() {
	completed := jt.CheckCompleted()
	for _, j := range completed {
		fmt.Fprintf(os.Stderr, "[%d]  Done                    %s\n", j.ID, j.Command)
		jt.Remove(j.ID)
	}
}

// SendSignal sends the given signal to the process group of the job.
func (j *Job) SendSignal(sig syscall.Signal) error {
	if j.Cmd == nil || j.Cmd.Process == nil {
		return fmt.Errorf("job has no process")
	}
	// Send to the process group (negative PID).
	return syscall.Kill(-j.Cmd.Process.Pid, sig)
}
