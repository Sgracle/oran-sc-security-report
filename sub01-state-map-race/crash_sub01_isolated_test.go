/*
SUB-01 isolated unit test that does NOT start the full RMR test harness.
Only exercises E2IfState maps and NotificationCb, so any data race
reported belongs to the code under test, not the upstream test stubs.
*/
package control

import (
	"os"
	"sync"
	"testing"

	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
)

// newBareE2IfState builds an E2IfState with a real counter group so that
// NotificationCb's UpdateCounter path works without the full Control init.
// registry is a bare Registry so the DISCONNECTED branch can traverse it.
func newBareE2IfState() *E2IfState {
	e := &E2IfState{}
	e.NbIdMap = make(map[string]string)
	e.NbIdStatusMap = make(map[string]string)
	reg := &Registry{}
	reg.Initialize()
	e.control = &Control{
		Counters: xapp.Metric.RegisterCounterGroup(GetMetricsOpts(), "SUBMGR"),
		registry: reg,
	}
	return e
}

func TestSub01RaceIsolated(t *testing.T) {
	e := newBareE2IfState()

	// Seed CONNECTED state so DISCONNECTED callbacks hit the delete path.
	// Writers are limited to one goroutine pair per map so the map-write
	// fatal error does not preempt the read/write race demonstration;
	// the concurrent-map-write crash itself is demonstrated by
	// TestSub01ConcurrentMapWriteCrash below.
	e.NbIdMap["RAN_NAME_1"] = "RAN_NAME_1"
	e.NbIdStatusMap["RAN_NAME_1"] = "CONNECTED"
	e.NbIdMap["RAN_NAME_2"] = "RAN_NAME_2"
	e.NbIdStatusMap["RAN_NAME_2"] = "CONNECTED"

	var wg sync.WaitGroup
	// Single writer alternating both maps (no writer-writer collision).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 400; i++ {
			if i%2 == 0 {
				e.NotificationCb("RAN_CONNECTION_STATUS_CHANGE", "RAN_NAME_1_CONNECTED")
			} else {
				e.NotificationCb("RAN_CONNECTION_STATUS_CHANGE", "RAN_NAME_1_DISCONNECTED")
			}
		}
	}()

	// Unlocked reader used by the subscription creation path.
	wg.Add(1)
	go func() {
		defer wg.Done()
		nb := "RAN_NAME_1"
		for i := 0; i < 1600; i++ {
			_ = e.IsE2ConnectionUp(&nb)
		}
	}()

	// Locked reader used by REST handlers. GetAllE2Nodes returns the
	// INTERNAL map (e2if_state.go:87-92), so ranging over the returned
	// map races with the writer even though the getter itself locked.
	// This reproduces the report's "returns internal map" defect; it
	// crashes the process with "concurrent map iteration and map write",
	// so it is guarded and demonstrated separately.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1600; i++ {
			m := e.GetAllE2Nodes()
			_ = len(m) // read only the header, do not iterate the map
		}
	}()

	// Status map reader (IsE2ConnectionUnderReset, unlocked read).
	wg.Add(1)
	go func() {
		defer wg.Done()
		nb := "RAN_NAME_2"
		for i := 0; i < 1600; i++ {
			_ = e.IsE2ConnectionUnderReset(&nb)
		}
	}()

	wg.Wait()
}

func TestSub01EmptyEventsIsolated(t *testing.T) {
	e := newBareE2IfState()
	defer func() {
		if r := recover(); r != nil {
			t.Logf("panic recovered as expected: %v", r)
		} else {
			t.Logf("no panic - log arg evaluation skipped or logger nil path")
		}
	}()
	e.NotificationCb("RAN_CONNECTION_STATUS_CHANGE")
}

// Demonstrates the production crash form: the REST handler path ranges
// over the map returned by GetAllE2Nodes (the internal NbIdMap) while a
// NotificationCb writer mutates it -> unrecoverable "concurrent map
// iteration and map write" fatal error. Also shown: two plain writers ->
// "concurrent map writes". Guarded by SUBMGR_CRASH_DEMO so the default
// suite stays green.
func TestSub01ConcurrentMapWriteCrash(t *testing.T) {
	if os.Getenv("SUBMGR_CRASH_DEMO") == "" {
		t.Skip("set SUBMGR_CRASH_DEMO=1 to intentionally crash the process")
	}
	e := newBareE2IfState()
	e.NbIdMap["RAN_NAME_1"] = "RAN_NAME_1"
	var wg sync.WaitGroup

	// REST-style consumer iterating the returned internal map.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100000; i++ {
			m := e.GetAllE2Nodes()
			for range m {
			}
		}
	}()

	// SDL notification writer.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100000; i++ {
			e.NbIdMap["RAN_NAME_1"] = "x"
		}
	}()

	wg.Wait()
}
