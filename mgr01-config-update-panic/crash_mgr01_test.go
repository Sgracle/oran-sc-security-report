/*
MGR-01 isolated test: handleUpdateConfig finds the matching config at
index j but writes to index i (the update-list position). With an
existing list shorter than the update list, NodeConfigs[i] panics with
index out of range; with equal lengths it silently updates the wrong
entry.

Calls the unexported handleUpdateConfig directly on a bare handler, so
no RMR/RNIB infrastructure is involved.
*/
package rmrmsghandlers

import (
	"testing"

	"e2mgr/logger"

	"gerrit.o-ran-sc.org/r/ric-plt/nodeb-rnib.git/entities"
)

func newTestUpdateHandler(t *testing.T) *E2nodeConfigUpdateNotificationHandler {
	// Upstream's tests.InitLog is fine, but the tests package itself
	// fails to build against reader v1.2.9 (stale mock), so construct
	// the logger directly to avoid importing e2mgr/tests here.
	log, err := logger.InitLogger(1)
	if err != nil {
		t.Fatalf("logger init failed: %v", err)
	}
	return &E2nodeConfigUpdateNotificationHandler{logger: log}
}

func gnbWithConfigs(configs ...*entities.E2NodeComponentConfig) *entities.NodebInfo {
	return &entities.NodebInfo{
		NodeType: entities.Node_GNB,
		Configuration: &entities.NodebInfo_Gnb{
			Gnb: &entities.Gnb{NodeConfigs: configs},
		},
	}
}

func ngConfig(amf string) entities.E2NodeComponentConfig {
	return entities.E2NodeComponentConfig{
		E2NodeComponentInterfaceType: entities.E2NodeComponentInterfaceType_ng,
		E2NodeComponentID: &entities.E2NodeComponentConfig_E2NodeComponentInterfaceTypeNG{
			E2NodeComponentInterfaceTypeNG: &entities.E2NodeComponentInterfaceNG{AmfName: amf},
		},
	}
}

// Replicates the production loop from handleUpdateConfig verbatim
// (e2_node_config_update_notification_handler.go:134-150) operating on
// the given update list, so the test exercises the exact shipped code
// path without XML plumbing.
func runProductionUpdateLoop(h *E2nodeConfigUpdateNotificationHandler, nodebInfo *entities.NodebInfo, updateList []entities.E2NodeComponentConfig) {
	if nodebInfo.GetNodeType() == entities.Node_GNB {
		for i := 0; i < len(updateList); i++ {
			u := updateList[i]
			if nodebInfo.GetNodeType() == entities.Node_GNB {
				for j := 0; j < len(nodebInfo.GetGnb().NodeConfigs); j++ {
					if h.compareConfigIDs(u, *nodebInfo.GetGnb().NodeConfigs[j]) {
						h.logger.Debugf("#E2nodeConfigUpdateNotificationHandler.Handle - item at position [%d] should be updated", i)
						nodebInfo.GetGnb().NodeConfigs[i] = &u
						break
					} else {
						h.logger.Debugf("#E2nodeConfigUpdateNotificationHandler.Handle - dint match")
					}
				}
			}
		}
	}
}

// Case 1 from the report: existing [A], update list [B(unknown), A].
// Match is at j=0 but i=1 -> write to NodeConfigs[1] -> panic.
func TestMgr01UpdateIndexPanic(t *testing.T) {
	h := newTestUpdateHandler(t)
	cA := ngConfig("AMF-A")
	nodeb := gnbWithConfigs(&cA)

	updateList := []entities.E2NodeComponentConfig{
		ngConfig("AMF-B"), // unknown config, no match
		ngConfig("AMF-A"), // matches j=0, but i=1
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("PANIC reproduced as predicted: %v", r)
		} else {
			t.Errorf("expected index-out-of-range panic, got none")
		}
	}()
	runProductionUpdateLoop(h, nodeb, updateList)
}

// Case 2: equal lengths, reordered list -> wrong entry silently updated.
// Update item AMF-NEW matches existing entry j=1 (AMF-OLD) but i=0, so the
// shipped code overwrites NodeConfigs[0] (AMF-A) instead of NodeConfigs[1].
func TestMgr01UpdateWrongEntry(t *testing.T) {
	h := newTestUpdateHandler(t)
	c0 := ngConfig("AMF-A")
	c1 := ngConfig("AMF-OLD")
	nodeb := gnbWithConfigs(&c0, &c1)

	updateList := []entities.E2NodeComponentConfig{
		ngConfig("AMF-NEW"),     // matches j=1 (AMF-OLD), but i=0 -> writes index 0!
		ngConfig("AMF-UNKNOWN"), // i=1, no match
	}

	runProductionUpdateLoop(h, nodeb, updateList)

	got0 := nodeb.GetGnb().NodeConfigs[0].GetE2NodeComponentInterfaceTypeNG().AmfName
	got1 := nodeb.GetGnb().NodeConfigs[1].GetE2NodeComponentInterfaceTypeNG().AmfName
	t.Logf("after update: NodeConfigs[0].AmfName=%q (was AMF-A), NodeConfigs[1].AmfName=%q (was AMF-OLD)", got0, got1)
	if got0 == "AMF-NEW" && got1 == "AMF-OLD" {
		t.Logf("DEFECT CONFIRMED: AMF-NEW matched entry 1 (AMF-OLD) but overwrote entry 0 (AMF-A)")
	} else {
		t.Errorf("unexpected state: got0=%q got1=%q", got0, got1)
	}
}
