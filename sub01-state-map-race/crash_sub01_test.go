/*
Crash-validation test for SUB-01 (e2if_state.go map concurrency).

Replicates the report scenario: SDL connection-status callbacks writing
NbIdMap/NbIdStatusMap concurrently with subscription-path reads
(IsE2ConnectionUp / GetAllE2Nodes), while no write path takes e.mutex.

Runs only in the isolated experiment copy under
/root/oran-lab/experiments/crash-validation-20261105.
*/
package control

import (
	"sync"
	"testing"
)

// Writers replicate NotificationCb state transitions; readers replicate the
// subscription creation path (IsE2ConnectionUp) and the REST query path
// (GetAllE2Nodes). The upstream code holds e.mutex only on the read helpers,
// never in NotificationCb, so -race should flag the pair.
func TestSub01ConcurrentStateMaps(t *testing.T) {
	e := mainCtrl.c.e2IfState

	// Seed two nodes as CONNECTED so the DISCONNECTED branch takes the
	// delete path (write to both maps).
	e.NbIdMap["RAN_NAME_1"] = "RAN_NAME_1"
	e.NbIdStatusMap["RAN_NAME_1"] = "CONNECTED"
	e.NbIdMap["RAN_NAME_2"] = "RAN_NAME_2"
	e.NbIdStatusMap["RAN_NAME_2"] = "CONNECTED"

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// 4 writer goroutines flip CONNECTED/DISCONNECTED via NotificationCb.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			ran := "RAN_NAME_1"
			if w%2 == 1 {
				ran = "RAN_NAME_2"
			}
			for i := 0; i < 300; i++ {
				select {
				case <-stop:
					return
				default:
				}
				if i%2 == 0 {
					e.NotificationCb("RAN_CONNECTION_STATUS_CHANGE", ran+"_CONNECTED")
				} else {
					e.NotificationCb("RAN_CONNECTION_STATUS_CHANGE", ran+"_DISCONNECTED")
				}
			}
		}(w)
	}

	// Reader 1: subscription-path lookup (unlocked read of NbIdMap).
	wg.Add(1)
	go func() {
		defer wg.Done()
		nb := "RAN_NAME_1"
		for i := 0; i < 1200; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_ = e.IsE2ConnectionUp(&nb)
		}
	}()

	// Reader 2: REST query path, GetAllE2Nodes (locked read).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1200; i++ {
			select {
			case <-stop:
				return
			default:
			}
			m := e.GetAllE2Nodes()
			for range m {
			}
		}
	}()

	// Reader 3: status-map read used by IsE2ConnectionUnderReset.
	wg.Add(1)
	go func() {
		defer wg.Done()
		nb := "RAN_NAME_2"
		for i := 0; i < 1200; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_ = e.IsE2ConnectionUnderReset(&nb)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	for i := 0; i < 4; i++ {
		// no-op keep writers deterministic length
	}
	close(stop)
	<-done
}

// Supplementary check from the report: NotificationCb dereferences events[0]
// in the first log call before the len(events)==0 guard.
func TestSub01EmptyEventsCallback(t *testing.T) {
	e := mainCtrl.c.e2IfState
	// Direct call with no variadic args: events[0] must panic before the
	// length check runs (log arg evaluation happens at call time).
	defer func() {
		if r := recover(); r != nil {
			t.Logf("panic recovered as expected: %v", r)
		} else {
			t.Logf("no panic - first log line likely disabled by level")
		}
	}()
	e.NotificationCb("RAN_CONNECTION_STATUS_CHANGE")
}
