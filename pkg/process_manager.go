package process_manager

import (
	"context"
	"errors"
	"io"
	"log"
	"os/exec"
	"sync"
	"sync/atomic"
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

// sharedBufWriter is the write half of a shared buffer that allows multiple
// readers.
type sharedBufWriter struct {
	bufLock sync.RWMutex
	buf     []byte

	closed atomic.Bool

	readersLock sync.Mutex
	readers     map[chan struct{}]struct{}
}

func (s *sharedBufWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	func() {
		s.bufLock.Lock()
		defer s.bufLock.Unlock()
		s.buf = append(s.buf, p...)
	}()

	s.readersLock.Lock()
	defer s.readersLock.Unlock()
	for output := range s.readers {
		// Non-blocking send.
		select {
		case output <- struct{}{}:
		default:
		}
	}

	return len(p), nil
}

// Close informs all associated readers (including readers that haven't been
// created yet) that the buffer will no longer receive new data.
//
// This method always returns a `nil` error.
func (s *sharedBufWriter) Close() error {
	s.readersLock.Lock()
	defer s.readersLock.Unlock()
	for output := range s.readers {
		close(output)
	}

	s.closed.Store(true)

	return nil
}

func (s *sharedBufWriter) newReader() *sharedBufReader {
	s.readersLock.Lock()
	defer s.readersLock.Unlock()

	// The writer needs to be able to buffer at least one notification to avoid
	// allowing Read to block waiting for new writes when it shouldn't.
	output := make(chan struct{}, 1)

	// If the writer is closed, immediately close the channel to indicate that
	// there will never be new data to wait for.
	if s.closed.Load() {
		close(output)
	}

	s.readers[output] = struct{}{}
	return &sharedBufReader{
		writer: s,
		output: output,
	}
}

// sharedBufReader is the read half of a shared buffer that allows multiple
// readers.
type sharedBufReader struct {
	writer *sharedBufWriter

	// pos tracks how far into buf has been read already.
	pos int

	// caughtUp indicates whether Read has read up to the end of buf at least
	// once.
	caughtUp bool

	// output can be received from to wait for new data to be written by the
	// writer.
	output chan struct{}
}

func (s *sharedBufReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Whether the channel is still open.
	open := true
	select {
	case _, open = <-s.output:
	default:
		// Wait for new data if the channel is open and we're caught up.
		if s.caughtUp {
			<-s.output
		}
	}

	func() {
		s.writer.bufLock.RLock()
		defer s.writer.bufLock.RUnlock()
		n = copy(p, s.writer.buf[s.pos:])
	}()

	if n == 0 {
		s.caughtUp = true
		if open {
			// Try again to wait for more data.
			return s.Read(p)
		} else {
			// There won't be any more data, this is the end of the buffer.
			return 0, io.EOF
		}
	}
	s.pos += n

	return n, nil
}

// Close unregisters this reader with its writer, freeing resources.
//
// This method always returns a `nil` error.
func (s *sharedBufReader) Close() error {
	s.writer.readersLock.Lock()
	defer s.writer.readersLock.Unlock()
	delete(s.writer.readers, s.output)

	return nil
}

type process struct {
	cancel       context.CancelFunc
	statusLock   sync.RWMutex
	status       ProcessStatus
	terminatedCh chan struct{}
	stdout       *sharedBufWriter
	stderr       *sharedBufWriter
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
	stdout := &sharedBufWriter{readers: make(map[chan struct{}]struct{})}
	stderr := &sharedBufWriter{readers: make(map[chan struct{}]struct{})}

	cmd := exec.CommandContext(ctx, path)
	cmd.Args = args
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return 0, err
	}

	p := &process{
		cancel:       cancel,
		terminatedCh: make(chan struct{}),
		stdout:       stdout,
		stderr:       stderr,
	}

	go func() {
		// Log unexpected errors only.
		err := cmd.Wait()
		_, ok := err.(*exec.ExitError)
		if err != nil && !ok {
			log.Print("failed to wait for process: ", err)
		}

		if stdout.Close() != nil {
			panic("unreachable")
		}
		if stderr.Close() != nil {
			panic("unreachable")
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

// StopAll stops all still-running process and waits for them to exit.
//
// This function should be called to to release system resources.
func (s *ProcessManager) StopAll() {
	s.processesLock.Lock()
	defer s.processesLock.Unlock()

	var wg sync.WaitGroup

	for _, p := range s.processes {
		// TODO: Use wg.Go when 1.25 is available in nixpkgs.
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.cancel()
			p.waitTerminated()
		}()
	}

	wg.Wait()
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
func (s *ProcessManager) Stream(id int) (
	stdout io.ReadCloser,
	stderr io.ReadCloser,
) {
	p, err := s.acquireProcess(id)
	if err != nil {
		return nil, nil
	}

	return p.stdout.newReader(), p.stderr.newReader()
}
