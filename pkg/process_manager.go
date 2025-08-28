package process_manager

import (
	"errors"
	"io"
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
	// ExitStatus is only valid if `State` is `EXITED`.
	ExitStatus int
	State      ProcessState
}

type processStream struct {
	// TODO
}

func (s *processStream) Read(p []byte) (n int, err error) {
	panic("TODO")
}

type process struct {
	cmdLock *sync.RWMutex
	cmd     *exec.Cmd
}

type ProcessManager struct {
	processesLock sync.RWMutex
	processes     []process
}

var ErrUnknownID = errors.New("unknown process ID")

func (s *ProcessManager) acquireProcess(id int) (process, error) {
	s.processesLock.RLock()
	defer s.processesLock.RUnlock()

	if id >= len(s.processes) || id < 0 {
		return process{}, ErrUnknownID
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
	cmd := exec.Command(path)
	cmd.Args = args

	// Ideally you'd probably want to track running processes somewhere to wait
	// on them and/or kill them when the caller wants to shut down. Also, don't
	// need to acquire `process.cmdLock` because this can't be accessed by other
	// threads yet anyway.
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	new := process{
		cmdLock: new(sync.RWMutex),
		cmd:     cmd,
	}

	s.processesLock.Lock()
	defer s.processesLock.Unlock()
	s.processes = append(s.processes, new)
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

	p.cmdLock.Lock()
	defer p.cmdLock.Unlock()

	// Process has already stopped or exited. A read lock would be sufficient
	// for this check but upgrading locks is slightly more effort.
	if p.cmd.ProcessState != nil {
		return nil
	}

	if err := p.cmd.Process.Kill(); err != nil {
		return err
	}

	// `*exec.ExitError` is expected when the process was killed.
	err = p.cmd.Wait()
	_, exitError := err.(*exec.ExitError)
	if err != nil && !exitError {
		return err
	}

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

	state := RUNNING
	exitStatus := 0

	p.cmdLock.RLock()
	defer p.cmdLock.RUnlock()

	if processState := p.cmd.ProcessState; processState != nil {
		// Operating systems where this type assertion would fail are
		// unsupported.
		if processState.Sys().(syscall.WaitStatus).Signaled() {
			state = STOPPED
		} else if processState.Exited() {
			state = EXITED
			exitStatus = processState.ExitCode()
		}
	}

	return ProcessStatus{
			State:      state,
			ExitStatus: exitStatus,
		},
		nil
}

// Stream gets a reader of the process's stdout and stderr since it was started.
//
// Returns `nil, nil` if the process ID is not known.
func (s *ProcessManager) Stream(id int) (stdout io.Reader, stderr io.Reader) {
	// Would construct a `processStream` for stderr and stdout and return it.
	panic("TODO")
}
