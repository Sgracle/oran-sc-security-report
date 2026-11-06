/*
MGR-02 isolated tests: two global maps in the models package lack a
consistent synchronization mechanism.

TestMgr02ProcedureMapCrossLockRace:
  models.UpdateProcedureType() writes ProcedureMap under the package-level
  models.procedureMapMutex, but ErrorIndicationHandler.HandleBasedOnProcedureType()
  reads it under a *different* lock (the handler's own procedureMapMutex
  field). The two locks exclude nothing, so writer vs reader race.

TestMgr02ExistingRanFunctionsMapUnlocked:
  ExistingRanFunctiuonsMap is written at managers/ric_service_update_manager.go:58
  and read at :70 with no lock at all. The test replicates those exact
  production statements concurrently.
*/
package rmrmsghandlers

import (
	"fmt"
	"sync"
	"testing"

	"e2mgr/logger"
	"e2mgr/models"

	"gerrit.o-ran-sc.org/r/ric-plt/nodeb-rnib.git/entities"
)

type noopRanDiscMgr struct{}

func (noopRanDiscMgr) DisconnectRan(string) error { return nil }

type noopRicSvcUpdMgr struct{}

func (noopRicSvcUpdMgr) RevertRanFunctions(string) error    { return nil }
func (noopRicSvcUpdMgr) StoreExistingRanFunctions(string) error { return nil }

func newTestErrorIndicationHandler(t *testing.T) *ErrorIndicationHandler {
	log, err := logger.InitLogger(1)
	if err != nil {
		t.Fatalf("logger init failed: %v", err)
	}
	return &ErrorIndicationHandler{
		logger:                  log,
		ranDisconnectionManager: noopRanDiscMgr{},
		RicServiceUpdateManager: noopRicSvcUpdMgr{},
	}
}

// Writer goroutines take the package-level lock; reader goroutines take
// the handler-level lock. No mutual exclusion => data race on ProcedureMap.
func TestMgr02ProcedureMapCrossLockRace(t *testing.T) {
	h := newTestErrorIndicationHandler(t)
	models.ProcedureMap = make(map[string]models.ProcedureType)

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				models.UpdateProcedureType(fmt.Sprintf("ran-%d", w), models.E2SetupProcedureOngoing)
			}
		}(w)
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func(r int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				_ = h.HandleBasedOnProcedureType(fmt.Sprintf("ran-%d", r))
			}
		}(r)
	}
	wg.Wait()
	t.Logf("survived cross-lock concurrent access; under -race this reports a data race on models.ProcedureMap")
}

// Write path replicates StoreExistingRanFunctions (line 58):
//   models.ExistingRanFunctiuonsMap[ranName] = nodebInfo.GetGnb().RanFunctions
// Read path replicates RevertRanFunctions (line 70):
//   nodebInfo.GetGnb().RanFunctions = models.ExistingRanFunctiuonsMap[ranName]
func TestMgr02ExistingRanFunctionsMapUnlocked(t *testing.T) {
	models.ExistingRanFunctiuonsMap = make(map[string][]*entities.RanFunction)

	rfs := []*entities.RanFunction{{}}

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				models.ExistingRanFunctiuonsMap[fmt.Sprintf("ran-%d", w)] = rfs // production line 58
			}
		}(w)
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func(r int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				_ = models.ExistingRanFunctiuonsMap[fmt.Sprintf("ran-%d", r)] // production line 70
			}
		}(r)
	}
	wg.Wait()
	t.Logf("survived unlocked concurrent access; under -race or natively this can fatal: concurrent map read and map write")
}
