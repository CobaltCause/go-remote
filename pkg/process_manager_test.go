package process_manager

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartStatusStopStatus(t *testing.T) {
	pm := new(ProcessManager)

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
