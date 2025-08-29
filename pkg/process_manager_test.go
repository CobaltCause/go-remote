package process_manager

import (
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
	assert.Equal(st.State, RUNNING)

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
	assert.Equal(st.State, STOPPED)
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
	assert.Equal(st.State, EXITED)
	assert.Equal(st.ExitStatus, 1)
}
