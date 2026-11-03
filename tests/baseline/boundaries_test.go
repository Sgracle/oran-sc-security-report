package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"ownershiplab/e2ap"
	"ownershiplab/xapp"
	"sync"
	"time"
)

// Only external dependencies are simulated. Control flow, tracker, transaction
// synchronization, subscription locking and registry cleanup are extracted.
const (
	cSubDelRequFromE2      = "required"
	cSubDelReqToE2         = "delete"
	cSubDelReReqToE2       = "retry"
	cSubDelReqTimerExpiry  = "timeout"
	cUnmergedSubscriptions = "unmerge"
	cRouteDeleteFail       = "route-error"
	cRouteDeleteUpdateFail = "route-update-error"
)

var waitRouteCleanup_ms time.Duration = 0
var e2tSubDelReqTime = 5 * time.Millisecond
var e2tMaxSubDelReqTryCount uint64 = 1

type RESTSubscription struct{}
type SubRouteInfo struct {
	Endpoints xapp.RmrEndpointList
	SubID     uint16
}
type RtmgrClient struct{ control *Control }

func (r *RtmgrClient) SubscriptionRequestDelete(p SubRouteInfo) error {
	c := r.control
	c.mu.Lock()
	defer c.mu.Unlock()
	c.routeCalls = append(c.routeCalls, fmt.Sprintf("delete:%d", p.SubID))
	if c.routeError {
		return errors.New("synthetic route failure")
	}
	delete(c.routes, uint32(p.SubID))
	return nil
}
func (r *RtmgrClient) SubscriptionRequestUpdate(p SubRouteInfo) error {
	c := r.control
	c.mu.Lock()
	defer c.mu.Unlock()
	c.routeCalls = append(c.routeCalls, fmt.Sprintf("update:%d", p.SubID))
	c.routes[uint32(p.SubID)] = p.Endpoints.Size()
	return nil
}

type E2IfState struct{ connected map[string]bool }

func (s *E2IfState) IsE2ConnectionUp(name *string) bool { return s.connected[*name] }

type E2ap struct{}

func (*E2ap) UnpackSubscriptionDeleteRequired(b []byte) (*e2ap.SubscriptionDeleteRequiredList, error) {
	var p e2ap.SubscriptionDeleteRequiredList
	err := json.Unmarshal(b, &p)
	return &p, err
}
func (*E2ap) PackSubscriptionDeleteRequest(p *e2ap.E2APSubscriptionDeleteRequest) (int, *e2ap.PackedData, error) {
	b, err := json.Marshal(p)
	return 12020, &e2ap.PackedData{Buf: b}, err
}

type Output struct {
	Target              string
	Instance, Requestor uint32
	Function            e2ap.FunctionId
}
type Control struct {
	registry            *Registry
	tracker             *Tracker
	e2ap                *E2ap
	e2IfState           *E2IfState
	UTTesting           bool // Deliberately false: production mutexes are not replaced.
	mu                  sync.Mutex
	outputs             []Output
	db                  map[uint32]string
	routes              map[uint32]int
	routeCalls          []string
	dbCalls             []uint32
	remote              map[uint32]bool
	counters            map[string]int
	remoteMode          string
	routeError, dbError bool
}

func (c *Control) UpdateCounter(n string)  { c.mu.Lock(); defer c.mu.Unlock(); c.counters[n]++ }
func idstring(...interface{}) string       { return "synthetic-context" }
func typeofSubsMessage(interface{}) string { return "synthetic-event" }
func (c *Control) handleXAPPSubscriptionDeleteRequest(*xapp.RMRParams) {
	panic("restart path outside fixture")
}
func (c *Control) WriteSubscriptionToDb(s *Subscription) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.db[s.ReqId.InstanceId] = s.Meid.RanName
	return nil
}
func (c *Control) RemoveSubscriptionFromSdl(id uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dbCalls = append(c.dbCalls, id)
	if c.dbError {
		return errors.New("synthetic SDL failure")
	}
	delete(c.db, id)
	return nil
}
func (c *Control) SendWithRetry(p *xapp.RMRParams, _ bool, _ int) error {
	var req e2ap.E2APSubscriptionDeleteRequest
	if err := json.Unmarshal(p.Payload, &req); err != nil {
		return err
	}
	c.mu.Lock()
	c.outputs = append(c.outputs, Output{p.Meid.RanName, req.InstanceId, req.Id, req.FunctionId})
	mode := c.remoteMode
	if mode == "success" {
		c.remote[req.InstanceId] = false
	}
	c.mu.Unlock()
	if mode != "timeout" {
		s := c.registry.GetSubscriptionForFixture(req.InstanceId)
		trans := s.GetTransaction()
		var event interface{} = &e2ap.E2APSubscriptionDeleteResponse{RequestId: req.RequestId, FunctionId: req.FunctionId}
		if mode == "failure" {
			event = &e2ap.E2APSubscriptionDeleteFailure{RequestId: req.RequestId, FunctionId: req.FunctionId}
		}
		go trans.SendEvent(event, time.Second)
	}
	return nil
}
func (r *Registry) GetSubscriptionForFixture(id uint32) *Subscription {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.register[id]
}
