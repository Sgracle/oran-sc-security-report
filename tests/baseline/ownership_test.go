package control

import (
	"encoding/json"
	"ownershiplab/e2ap"
	"ownershiplab/xapp"
	"reflect"
	"sync"
	"testing"
)

type State struct {
	Present, Valid, RemoteActive bool
	Owner                        string
	Endpoints, OngoingDelete     int
	DeleteFromDb                 bool
	DBOwner                      string
	RouteEndpoints, Recycled     int
}

func fixture() (*Control, map[uint32]*Subscription) {
	c := &Control{registry: &Registry{mutex: new(sync.Mutex), register: map[uint32]*Subscription{}}, tracker: &Tracker{}, e2ap: &E2ap{}, e2IfState: &E2IfState{connected: map[string]bool{"A": true, "B": true}}, db: map[uint32]string{}, routes: map[uint32]int{}, remote: map[uint32]bool{}, counters: map[string]int{}, remoteMode: "success"}
	c.tracker.Init()
	c.registry.rtmgrClient = &RtmgrClient{control: c}
	for _, p := range []struct {
		id    uint32
		owner string
	}{{101, "A"}, {102, "B"}} {
		s := &Subscription{valid: true, registry: c.registry, Meid: &xapp.RMRMeid{RanName: p.owner}, ReqId: RequestId{RequestId: e2ap.RequestId{Id: 900, InstanceId: p.id}}, SubReqMsg: &e2ap.E2APSubscriptionRequest{RequestId: e2ap.RequestId{Id: 900, InstanceId: 777}, FunctionId: 2}, RMRRouteCreated: true}
		s.EpList.AddEndpoint(xapp.NewRmrEndpoint("kpm-" + p.owner + ":4560"))
		c.registry.register[p.id] = s
		c.db[p.id] = p.owner
		c.routes[p.id] = 1
		c.remote[p.id] = true
	}
	return c, map[uint32]*Subscription{101: c.registry.register[101], 102: c.registry.register[102]}
}
func item(id uint32) e2ap.E2APSubscriptionDeleteRequired {
	return e2ap.E2APSubscriptionDeleteRequired{RequestId: e2ap.RequestId{Id: 123, InstanceId: id}, FunctionId: 2}
}
func message(source string, items ...e2ap.E2APSubscriptionDeleteRequired) *xapp.RMRParams {
	b, _ := json.Marshal(e2ap.SubscriptionDeleteRequiredList{E2APSubscriptionDeleteRequiredRequests: items})
	return &xapp.RMRParams{Meid: &xapp.RMRMeid{RanName: source}, Payload: b, PayloadLen: len(b), Mtype: 12023}
}
func snapshot(c *Control, s *Subscription) State {
	// Wait for production goroutine defers, then take locks in registry->subs order.
	s.TransLock.Lock()
	defer s.TransLock.Unlock()
	c.registry.mutex.Lock()
	defer c.registry.mutex.Unlock()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	owner := ""
	if s.Meid != nil {
		owner = s.Meid.RanName
	}
	id := s.ReqId.InstanceId
	recycled := 0
	for _, n := range c.registry.subIds {
		if n == id {
			recycled++
		}
	}
	return State{c.registry.register[id] == s, s.valid, c.remote[id], owner, s.EpList.Size(), s.OngoingDelCount, s.DeleteFromDb, c.db[id], c.routes[id], recycled}
}
func logEvidence(t *testing.T, c *Control, before, after map[uint32]State) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	b, _ := json.Marshal(map[string]interface{}{"scenario": t.Name(), "before": before, "after": after, "outputs": c.outputs, "route_calls": c.routeCalls, "db_remove_calls": c.dbCalls})
	t.Log("EVIDENCE " + string(b))
}
func runScenario(t *testing.T, p *xapp.RMRParams, target uint32, allowed bool) {
	t.Helper()
	c, ss := fixture()
	before := map[uint32]State{101: snapshot(c, ss[101]), 102: snapshot(c, ss[102])}
	c.handleE2TSubscriptionDeleteRequired(p)
	after := map[uint32]State{101: snapshot(c, ss[101]), 102: snapshot(c, ss[102])}
	logEvidence(t, c, before, after)
	if !allowed {
		if !reflect.DeepEqual(before, after) || len(c.outputs) != 0 || len(c.dbCalls) != 0 || len(c.routeCalls) != 0 {
			t.Errorf("rejected message changed state or emitted delete: before=%+v after=%+v outputs=%+v", before, after, c.outputs)
		}
		return
	}
	if after[target].Present || after[target].Valid || after[target].RemoteActive || after[target].DBOwner != "" || after[target].RouteEndpoints != 0 || after[target].Recycled != 1 {
		t.Errorf("legal delete did not clean target: %+v", after[target])
	}
	other := uint32(101)
	if target == 101 {
		other = 102
	}
	if !reflect.DeepEqual(before[other], after[other]) {
		t.Errorf("other node changed")
	}
	if len(c.outputs) != 1 || c.outputs[0] != (Output{before[target].Owner, target, 123, 2}) {
		t.Errorf("wrong delete output: %+v", c.outputs)
	}
}
func TestOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		id           uint32
		allow        bool
	}{
		{"A_own", "A", 101, true}, {"A_references_B", "A", 102, false}, {"B_own", "B", 102, true}, {"unknown_instance", "A", 65000, false}, {"B_references_A", "B", 101, false},
	} {
		t.Run(tc.name, func(t *testing.T) { runScenario(t, message(tc.source, item(tc.id)), tc.id, tc.allow) })
	}
}
func TestIdentityValidation(t *testing.T) {
	for _, name := range []string{"wrong_requestor", "wrong_function", "empty_source", "nil_source", "nil_owner", "nil_subscription_config"} {
		t.Run(name, func(t *testing.T) {
			c, ss := fixture()
			p := message("B", item(102))
			switch name {
			case "wrong_requestor":
				i := item(102)
				i.Id = 900
				p = message("B", i)
			case "wrong_function":
				i := item(102)
				i.FunctionId = 3
				p = message("B", i)
			case "empty_source":
				p.Meid.RanName = ""
			case "nil_source":
				p.Meid = nil
			case "nil_owner":
				ss[102].Meid = nil
			case "nil_subscription_config":
				ss[102].SubReqMsg = nil
			}
			before := snapshot(c, ss[102])
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("untrusted identity caused panic: %v", r)
				}
			}()
			c.handleE2TSubscriptionDeleteRequired(p)
			after := snapshot(c, ss[102])
			logEvidence(t, c, map[uint32]State{102: before}, map[uint32]State{102: after})
			if !reflect.DeepEqual(before, after) || len(c.outputs) != 0 || len(c.dbCalls) != 0 || len(c.routeCalls) != 0 {
				t.Errorf("identity mismatch affected subscription")
			}
		})
	}
}
func TestMixedBatch(t *testing.T) {
	c, ss := fixture()
	beforeB := snapshot(c, ss[102])
	before := map[uint32]State{101: snapshot(c, ss[101]), 102: beforeB}
	c.handleE2TSubscriptionDeleteRequired(message("A", item(102), item(101), item(65000)))
	logEvidence(t, c, before, map[uint32]State{101: snapshot(c, ss[101]), 102: snapshot(c, ss[102])})
	if !reflect.DeepEqual(beforeB, snapshot(c, ss[102])) {
		t.Error("foreign entry in mixed batch changed B")
	}
	if snapshot(c, ss[101]).Present || len(c.outputs) != 1 || c.outputs[0].Target != "A" {
		t.Errorf("own entry not isolated: %+v", c.outputs)
	}
}
func TestConcurrentDifferentSubscriptions(t *testing.T) {
	c, ss := fixture()
	before := map[uint32]State{101: snapshot(c, ss[101]), 102: snapshot(c, ss[102])}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, p := range []*xapp.RMRParams{message("A", item(101)), message("B", item(102))} {
		wg.Add(1)
		go func(p *xapp.RMRParams) { defer wg.Done(); <-start; c.handleE2TSubscriptionDeleteRequired(p) }(p)
	}
	close(start)
	wg.Wait()
	logEvidence(t, c, before, map[uint32]State{101: snapshot(c, ss[101]), 102: snapshot(c, ss[102])})
	if snapshot(c, ss[101]).Present || snapshot(c, ss[102]).Present || len(c.outputs) != 2 {
		t.Fatalf("concurrent cleanup failed: %+v", c.outputs)
	}
	for _, o := range c.outputs {
		if (o.Instance == 101 && o.Target != "A") || (o.Instance == 102 && o.Target != "B") {
			t.Errorf("cross association: %+v", o)
		}
	}
}
func TestConcurrentForeignAndOwn(t *testing.T) {
	c, ss := fixture()
	before := snapshot(c, ss[102])
	beforeAll := map[uint32]State{101: snapshot(c, ss[101]), 102: before}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, p := range []*xapp.RMRParams{message("A", item(101)), message("A", item(102))} {
		wg.Add(1)
		go func(p *xapp.RMRParams) { defer wg.Done(); <-start; c.handleE2TSubscriptionDeleteRequired(p) }(p)
	}
	close(start)
	wg.Wait()
	logEvidence(t, c, beforeAll, map[uint32]State{101: snapshot(c, ss[101]), 102: snapshot(c, ss[102])})
	if !reflect.DeepEqual(before, snapshot(c, ss[102])) || len(c.outputs) != 1 || c.outputs[0].Target != "A" {
		t.Errorf("concurrent foreign request affected B: %+v", c.outputs)
	}
}
func TestCleanupDoesNotRequireRemoteSuccess(t *testing.T) {
	for _, mode := range []string{"failure", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			c, ss := fixture()
			c.remoteMode = mode
			before := snapshot(c, ss[102])
			c.handleE2TSubscriptionDeleteRequired(message("B", item(102)))
			after := snapshot(c, ss[102])
			logEvidence(t, c, map[uint32]State{102: before}, map[uint32]State{102: after})
			if after.Present || after.DBOwner != "" || after.RouteEndpoints != 0 || after.Recycled != 1 || !after.RemoteActive {
				t.Errorf("unexpected characterization: %+v", after)
			}
		})
	}
}
func TestCleanupFailureCharacterization(t *testing.T) {
	c, ss := fixture()
	before := snapshot(c, ss[102])
	c.routeError = true
	c.dbError = true
	c.handleE2TSubscriptionDeleteRequired(message("B", item(102)))
	after := snapshot(c, ss[102])
	logEvidence(t, c, map[uint32]State{102: before}, map[uint32]State{102: after})
	if after.Present || after.Recycled != 1 || after.DBOwner != "B" || after.RouteEndpoints != 1 {
		t.Errorf("expected local release despite external cleanup errors: %+v", after)
	}
}

func TestRejectedMalformedOrOngoing(t *testing.T) {
	for _, name := range []string{"nil_params", "malformed_payload", "ongoing_delete"} {
		t.Run(name, func(t *testing.T) {
			c, ss := fixture()
			p := message("B", item(102))
			switch name {
			case "nil_params":
				p = nil
			case "malformed_payload":
				p.Payload = []byte("malformed synthetic codec input")
			case "ongoing_delete":
				ss[102].OngoingDelCount = 1
			}
			before := snapshot(c, ss[102])
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("rejected input panicked: %v", r)
				}
			}()
			c.handleE2TSubscriptionDeleteRequired(p)
			after := snapshot(c, ss[102])
			logEvidence(t, c, map[uint32]State{102: before}, map[uint32]State{102: after})
			if !reflect.DeepEqual(before, after) || len(c.outputs) != 0 || len(c.dbCalls) != 0 || len(c.routeCalls) != 0 {
				t.Error("rejected input had side effects")
			}
		})
	}
}
