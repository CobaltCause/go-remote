package process_manager

import "io"

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

func (*processStream) Read(p []byte) (n int, err error) {
	panic("TODO")
}

type ProcessManager struct {
	// TODO
}

func (*ProcessManager) Start() uint32 {
	panic("TODO")
}

// Stop stops a process by its ID.
//
// If the error is not `nil`, then the process was stopped. An error is returned
// if the process ID does not exist.
func (*ProcessManager) Stop(id uint32) error {
	panic("TODO")
}

// Status returns the status of a process by its ID.
//
// An error is returned if the process ID is not known.
func (*ProcessManager) Status(id uint32) (ProcessStatus, error) {
	panic("TODO")
}

// Stream gets a reader of the process's stdout and stderr since it was started.
//
// Returns `nil, nil` if the process ID is not known.
func (*ProcessManager) Stream(id uint32) (stdout io.Reader, stderr io.Reader) {
	// Would construct a `processStream` for stderr and stdout and return it.
	panic("TODO")
}
