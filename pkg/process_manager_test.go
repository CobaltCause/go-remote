package process_manager

import (
	"sync"
	"testing"
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
	id, err := pm.Start("sleep", "sleep", "infinity")
	if err != nil {
		t.Error("Start unexpectedly failed:", err)
		return
	}

	st, err := pm.Status(id)
	if err != nil {
		t.Error("Status unexpectedly failed:", err)
		return
	}
	if st.State != RUNNING {
		t.Error("Unexpected State, got:", st.State)
	}

	stop := func() {
		if err := pm.Stop(id); err != nil {
			t.Error("Stop unexpectedly failed:", err)
		}
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
			stop()
		}()
	}
	wg.Wait()

	st, err = pm.Status(id)
	if err != nil {
		t.Error("Status unexpectedly failed:", err)
		return
	}
	if st.State != STOPPED {
		t.Error("Unexpected State, got:", st.State)
	}
}
