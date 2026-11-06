/*
SUB-02 isolated test: Registry.DeleteAllE2Subscriptions walks and mutates
r.register / r.restSubscriptions / r.subIds without taking r.mutex, while
the normal create path (AssignToSubscription -> allocateSubs) and query
paths hold the lock. Concurrent map traversal + writes = fatal error or
race report.

Uses a bare Registry (no network/SDL side effects), so any race reported
is attributable to the missing lock in DeleteAllE2Subscriptions itself.
*/
package control

import (
	"sync"
	"testing"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
)

func newSub(ran string) *Subscription {
	s := &Subscription{}
	s.Meid = &xapp.RMRMeid{RanName: ran}
	return s
}

func TestSub02DeleteAllVsAllocate(t *testing.T) {
	r := &Registry{}
	r.Initialize()

	// Pre-populate subscriptions for two RANs, mirroring production state.
	for i := uint32(1); i <= 300; i++ {
		ran := "RAN_A"
		if i%2 == 0 {
			ran = "RAN_B"
		}
		r.register[i] = newSub(ran)
	}
	// Keep an ID pool as Initialize does.
	r.subIds = r.subIds[:0]
	for i := uint32(400); i < 65535; i++ {
		r.subIds = append(r.subIds, i)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Cleanup goroutines: DeleteAllE2Subscriptions (no lock - the defect).
	for d := 0; d < 2; d++ {
		wg.Add(1)
		go func(d int) {
			defer wg.Done()
			ran := "RAN_A"
			if d == 1 {
				ran = "RAN_B"
			}
			for i := 0; i < 50; i++ {
				select {
				case <-stop:
					return
				default:
				}
				// Deletes matching entries without r.mutex; repopulate after
				// each pass so later iterations still traverse a full map.
				r.DeleteAllE2Subscriptions(ran, nil)
				for id := uint32(1); id <= 300; id++ {
					r.mutex.Lock()
					r.register[id] = newSub(ran)
					r.mutex.Unlock()
				}
			}
		}(d)
	}

	// Creator goroutines: locked map writes via subIds + register
	// (mirrors allocateSubs running under r.mutex).
	for c := 0; c < 2; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for i := 0; i < 400; i++ {
				select {
				case <-stop:
					return
				default:
				}
				r.mutex.Lock()
				if len(r.subIds) > 1 {
					id := r.subIds[0]
					r.subIds = r.subIds[1:]
					r.register[id] = newSub("RAN_C")
				}
				r.mutex.Unlock()
			}
		}(c)
	}

	// Reader goroutine: locked iteration (mirrors query endpoints).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 800; i++ {
			select {
			case <-stop:
				return
			default:
			}
			r.mutex.Lock()
			for _, s := range r.register {
				_ = s.Meid.RanName
				break
			}
			r.mutex.Unlock()
		}
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
	}
	close(stop)
	<-done
}
