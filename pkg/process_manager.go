package process_manager

import (
	"context"
	"errors"
	"io"
	"log"
	"os/exec"
	"sync"
	"syscall"
)

type ProcessState int

const (
	// RUNNING indicates that the process has not yet terminated.
	RUNNING ProcessState = iota

	// EXITED indicates that the process has exited normally.
	EXITED

	// STOPPED indicates that the process has stopped.
	//
	// This state is typically due to `(*ProcessManager) Stop` being called.
	STOPPED
)

type ProcessStatus struct {
	// ExitCode is only valid if `State` is `EXITED`.
	ExitCode int
	State    ProcessState
}

type processStream struct {
	// TODO
}

func (s *processStream) Read(p []byte) (n int, err error) {
	panic("TODO")
}

type process struct {
	cancel       context.CancelFunc
	statusLock   sync.RWMutex
	status       ProcessStatus
	terminatedCh chan struct{}
}

func (s *process) markTerminated() {
	close(s.terminatedCh)
}

func (s *process) waitTerminated() {
	<-s.terminatedCh
}

type ProcessManager struct {
	processesLock sync.RWMutex
	processes     []*process
}

var ErrUnknownID = errors.New("unknown process ID")

func (s *ProcessManager) acquireProcess(id int) (*process, error) {
	s.processesLock.RLock()
	defer s.processesLock.RUnlock()

	if id >= len(s.processes) || id < 0 {
		return &process{}, ErrUnknownID
	}

	return s.processes[id], nil
}

// Start starts a process and returns its ID.
//
// If `path` contains no path separators, the location is resolved from `$PATH`.
// Note that `args` includes `argv[0]`.
//
// This function fails if starting the process fails.
func (s *ProcessManager) Start(path string, args ...string) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, path)
	cmd.Args = args

	if err := cmd.Start(); err != nil {
		cancel()
		return 0, err
	}

	p := &process{
		cancel:       cancel,
		terminatedCh: make(chan struct{}),
	}

	go func() {
		// Log unexpected errors only.
		err := cmd.Wait()
		_, ok := err.(*exec.ExitError)
		if err != nil && !ok {
			log.Print("failed to wait for process: ", err)
		}

		processStatus := ProcessStatus{}
		if cmd.ProcessState != nil {
			// Operating systems where this type assertion would fail are
			// unsupported.
			if cmd.ProcessState.Sys().(syscall.WaitStatus).Signaled() {
				processStatus.State = STOPPED
			} else if cmd.ProcessState.Exited() {
				processStatus.State = EXITED
				processStatus.ExitCode = cmd.ProcessState.ExitCode()
			}
		}

		p.statusLock.Lock()
		defer p.statusLock.Unlock()
		p.status = processStatus

		p.markTerminated()
	}()

	s.processesLock.Lock()
	defer s.processesLock.Unlock()
	s.processes = append(s.processes, p)
	return len(s.processes) - 1, nil
}

// Stop stops a process by its ID.
//
// If the error is not `nil`, then the process was stopped. An error is returned
// if the process ID does not exist or the process could not be killed.
func (s *ProcessManager) Stop(id int) error {
	p, err := s.acquireProcess(id)
	if err != nil {
		return err
	}

	p.cancel()
	p.waitTerminated()

	return nil
}

// Wait waits for a process to terminate.
//
// An error is returned if the process ID is not known or waiting failed.
func (s *ProcessManager) Wait(id int) error {
	p, err := s.acquireProcess(id)
	if err != nil {
		return err
	}

	p.waitTerminated()

	return nil
}

// Status returns the status of a process by its ID.
//
// An error is returned if the process ID is not known.
func (s *ProcessManager) Status(id int) (ProcessStatus, error) {
	p, err := s.acquireProcess(id)
	if err != nil {
		return ProcessStatus{}, err
	}

	p.statusLock.RLock()
	defer p.statusLock.RUnlock()

	return p.status, nil
}

// Stream gets a reader of the process's stdout and stderr since it was started.
//
// Returns `nil, nil` if the process ID is not known.
func (s *ProcessManager) Stream(id int) (stdout io.Reader, stderr io.Reader) {
	// Would construct a `processStream` for stderr and stdout and return it.
	panic("TODO")
}
