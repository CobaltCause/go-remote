package process_manager

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartStatusStopStatus(t *testing.T) {
	pm := new(ProcessManager)
	defer pm.StopAll()

	var wg sync.WaitGroup

	// TODO: Use wg.Go when 1.25 is available in nixpkgs.
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			innerStartStatusStopStatus(t, pm)
		}()
	}

	wg.Wait()
}

func innerStartStatusStopStatus(t *testing.T, pm *ProcessManager) {
	assert := assert.New(t)

	id, err := pm.Start("sleep", "sleep", "infinity")
	if !assert.Nil(err) {
		return
	}

	st, err := pm.Status(id)
	if !assert.Nil(err) {
		return
	}
	assert.Equal(RUNNING, st.State, "process should be running")

	// Do this to try ensure repeated/concurrent calls to `Stop` for the same
	// process ID don't cause problems.
	//
	// TODO: Use wg.Go when 1.25 is available in nixpkgs.
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.Nil(pm.Stop(id))
		}()
	}
	wg.Wait()

	st, err = pm.Status(id)
	if !assert.Nil(err) {
		return
	}
	assert.Equal(STOPPED, st.State, "process should be stopped")
}

func TestStartWaitStatus(t *testing.T) {
	pm := new(ProcessManager)
	defer pm.StopAll()

	var wg sync.WaitGroup

	// TODO: Use wg.Go when 1.25 is available in nixpkgs.
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			innerStartWaitStatus(t, pm)
		}()
	}

	wg.Wait()
}

func innerStartWaitStatus(t *testing.T, pm *ProcessManager) {
	assert := assert.New(t)

	id, err := pm.Start("false", "false")
	if !assert.Nil(err) {
		return
	}

	// Do this to try ensure repeated/concurrent calls to `Stop` for the same
	// process ID don't cause problems.
	//
	// TODO: Use wg.Go when 1.25 is available in nixpkgs.
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.Nil(pm.Wait(id))
		}()
	}
	wg.Wait()

	st, err := pm.Status(id)
	if !assert.Nil(err) {
		return
	}
	assert.Equal(EXITED, st.State)
	assert.Equal(1, st.ExitCode)
}

func TestStream(t *testing.T) {
	assert := assert.New(t)

	pm := new(ProcessManager)
	defer pm.StopAll()

	// Ideally this test would use an IPC mechanism for synchronization, as time
	// is not a synchronization primitive, and relying on timing for
	// synchronization causes tests to become unnecessarily slow and unreliable.

	id, err := pm.Start("bash", "bash", "-c", `
		for x in $(seq 1 3); do
			sleep 0.25
			echo "test $x"
		done
	`)
	if !assert.Nil(err) {
		return
	}

	readStream := func(postWait bool) {
		stdout, stderr := pm.Stream(id)
		defer func() {
			if stdout.Close() != nil {
				panic("unreachable")
			}
		}()
		defer func() {
			if stderr.Close() != nil {
				panic("unreachable")
			}
		}()

		iter := 1
		for {
			buf := make([]byte, 1024)

			n, err := stdout.Read(buf)

			// This conversion should be fine since we control the started
			// process's output.
			t.Logf("Read: %#v", string(buf[:n]))

			if errors.Is(err, io.EOF) {
				t.Log("Last read was EOF")
				assert.Equal("", string(buf[:n]))
				break
			}

			assert.Nil(err, "unexpected error")

			if postWait {
				// Should be able to fetch the entire history in one read.
				assert.Equal("test 1\ntest 2\ntest 3\n", string(buf[:n]))
			} else {
				// Should fetch each new output as it comes in.
				assert.Equal(fmt.Sprintf("test %v\n", iter), string(buf[:n]))
			}

			iter += 1
		}
	}

	readStream(false)

	assert.Nil(pm.Wait(id))

	readStream(true)
}
