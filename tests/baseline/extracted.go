/*
==================================================================================
  Copyright (c) 2019 AT&T Intellectual Property.
  Copyright (c) 2019 Nokia

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
==================================================================================
*/
// Code extracted from O-RAN SC (Apache-2.0). DO NOT EDIT.
package control

import (
	"fmt"
	"ownershiplab/e2ap"
	"ownershiplab/xapp"
	"strconv"
	"sync"
	"time"
)

func (c *Control) rmrSendToE2T(desc string, subs *Subscription, trans *TransactionSubs) (err error) {
	params := &xapp.RMRParams{}
	params.Mtype = trans.GetMtype()
	params.SubId = int(subs.GetReqId().InstanceId)
	params.Xid = ""
	params.Meid = subs.GetMeid()
	params.Src = ""
	params.PayloadLen = len(trans.Payload.Buf)
	params.Payload = trans.Payload.Buf
	params.Mbuf = nil
	xapp.Logger.Debug("MSG to E2T: %s %s %s", desc, trans.String(), params.String())
	err = c.SendWithRetry(params, false, 5)
	if err != nil {
		xapp.Logger.Error("rmrSendToE2T: Send failed: %+v", err)
	}
	return err
}
func (c *Control) handleSubscriptionDelete(subs *Subscription, parentTrans *TransactionXapp, waitRouteCleanupTime time.Duration) {

	trans := c.tracker.NewSubsTransaction(subs)
	subs.WaitTransactionTurn(trans)
	defer subs.ReleaseTransactionTurn(trans)
	defer trans.Release()

	xapp.Logger.Debug("SUBS-SubDelReq: Handling %s", idstring(nil, trans, subs, parentTrans))

	subs.mutex.Lock()

	if subs.valid && subs.EpList.HasEndpoint(parentTrans.GetEndpoint()) && subs.EpList.Size() == 1 {
		subs.valid = false
		subs.mutex.Unlock()
		c.sendE2TSubscriptionDeleteRequest(subs, trans, parentTrans)
	} else {
		subs.mutex.Unlock()
	}

	// Now RemoveFromSubscription in here to avoid race conditions (mostly concerns delete)
	c.registry.RemoveFromSubscription(subs, parentTrans, waitRouteCleanupTime, c)
	parentTrans.SendEvent(nil, 0)
}
func (c *Control) sendE2TSubscriptionDeleteRequest(subs *Subscription, trans *TransactionSubs, parentTrans *TransactionXapp) interface{} {
	var err error
	var event interface{}
	var timedOut bool
	const ricRequestorId = 123

	subDelReqMsg := &e2ap.E2APSubscriptionDeleteRequest{}
	subDelReqMsg.RequestId = subs.GetReqId().RequestId
	subDelReqMsg.RequestId.Id = ricRequestorId
	subDelReqMsg.FunctionId = subs.SubReqMsg.FunctionId
	trans.Mtype, trans.Payload, err = c.e2ap.PackSubscriptionDeleteRequest(subDelReqMsg)
	if err != nil {
		xapp.Logger.Error("SUBS-SubDelReq: %s", idstring(err, trans, subs, parentTrans))
		return event
	}

	for retries := uint64(0); retries < e2tMaxSubDelReqTryCount; retries++ {
		desc := fmt.Sprintf("(retry %d)", retries)
		if retries == 0 {
			c.UpdateCounter(cSubDelReqToE2)
		} else {
			c.UpdateCounter(cSubDelReReqToE2)
		}
		err := c.rmrSendToE2T(desc, subs, trans)
		if err != nil {
			xapp.Logger.Error("SUBS-SubDelReq: rmrSendToE2T failure: %s", idstring(err, trans, subs, parentTrans))
		}
		event, timedOut = trans.WaitEvent(e2tSubDelReqTime)
		if timedOut {
			c.UpdateCounter(cSubDelReqTimerExpiry)
			continue
		}
		break
	}
	xapp.Logger.Debug("SUBS-SubDelReq: Response handling event(%s) %s", typeofSubsMessage(event), idstring(nil, trans, subs, parentTrans))
	return event
}
func (c *Control) RemoveSubscriptionFromDb(subs *Subscription) {
	xapp.Logger.Debug("RemoveSubscriptionFromDb() subId = %v", subs.ReqId.InstanceId)
	err := c.RemoveSubscriptionFromSdl(subs.ReqId.InstanceId)
	if err != nil {
		xapp.Logger.Error("%v", err)
	}
}
func (c *Control) SendSubscriptionDeleteReq(subs *Subscription, e2SubsDelRequired bool) {

	if c.UTTesting == true {
		// Reqistry mutex is not locked after real restart but it can be when restart is simulated in unit tests
		c.registry.mutex = new(sync.Mutex)
	}

	const ricRequestorId = 123
	xapp.Logger.Debug("Sending subscription delete due to restart. subId = %v", subs.ReqId.InstanceId)

	// Send delete for every endpoint in the subscription
	if subs.PolicyUpdate == false {
		subDelReqMsg := &e2ap.E2APSubscriptionDeleteRequest{}
		subDelReqMsg.RequestId = subs.GetReqId().RequestId
		subDelReqMsg.RequestId.Id = ricRequestorId
		subDelReqMsg.FunctionId = subs.SubReqMsg.FunctionId
		mType, payload, err := c.e2ap.PackSubscriptionDeleteRequest(subDelReqMsg)
		if err != nil {
			xapp.Logger.Error("SendSubscriptionDeleteReq() %s", idstring(err))
			return
		}
		for _, endPoint := range subs.EpList.Endpoints {
			params := &xapp.RMRParams{}
			params.Mtype = mType
			params.SubId = int(subs.GetReqId().InstanceId)
			params.Xid = ""
			params.Meid = subs.Meid
			params.Src = endPoint.String()
			params.PayloadLen = len(payload.Buf)
			params.Payload = payload.Buf
			params.Mbuf = nil
			subs.DeleteFromDb = true
			if !e2SubsDelRequired {
				c.handleXAPPSubscriptionDeleteRequest(params)
			} else {
				c.SendSubscriptionDeleteReqToE2T(subs, params)
			}
		}
	}
}
func (c *Control) handleE2TSubscriptionDeleteRequired(params *xapp.RMRParams) {
	xapp.Logger.Info("MSG from E2T: %s", params.String())
	c.UpdateCounter(cSubDelRequFromE2)
	subsDelRequMsg, err := c.e2ap.UnpackSubscriptionDeleteRequired(params.Payload)
	if err != nil {
		xapp.Logger.Error("MSG-SubDelRequired: %s", idstring(err, params))
		//c.sendE2TErrorIndication(nil)
		return
	}
	var subscriptions = map[string][]e2ap.E2APSubscriptionDeleteRequired{}
	var subDB = []*Subscription{}
	for _, subsTobeRemove := range subsDelRequMsg.E2APSubscriptionDeleteRequiredRequests {
		subs, err := c.registry.GetSubscriptionFirstMatch([]uint32{subsTobeRemove.RequestId.InstanceId})
		if err != nil {
			xapp.Logger.Error("MSG-SubDelFail: %s", idstring(err, params))
			continue
		}
		// Check if Delete Subscription Already triggered
		if subs.OngoingDelCount > 0 {
			continue
		}
		subDB = append(subDB, subs)
		for _, endpoint := range subs.EpList.Endpoints {
			subscriptions[endpoint.Addr] = append(subscriptions[endpoint.Addr], subsTobeRemove)
		}
		// Sending Subscription Delete Request to E2T
		//	c.SendSubscriptionDeleteReq(subs, true)
	}
	for _, subsTobeRemove := range subDB {
		// Sending Subscription Delete Request to E2T
		c.SendSubscriptionDeleteReq(subsTobeRemove, true)
	}
}
func (c *Control) SendSubscriptionDeleteReqToE2T(subs *Subscription, params *xapp.RMRParams) {
	xapp.Logger.Debug("MSG TO E2T: %s", params.String())
	c.UpdateCounter(cSubDelReqToE2)

	if c.e2IfState.IsE2ConnectionUp(&params.Meid.RanName) == false {
		xapp.Logger.Error("No E2 connection for ranName %v", params.Meid.RanName)
		return
	}

	trans := c.tracker.NewXappTransaction(xapp.NewRmrEndpoint(params.Src), params.Xid, subs.ReqId.RequestId, params.Meid)
	if trans == nil {
		xapp.Logger.Error("XAPP-SubDelReq: %s", idstring(fmt.Errorf("transaction not created"), params))
		return
	}
	defer trans.Release()

	err := c.tracker.Track(trans)
	if err != nil {
		xapp.Logger.Error("XAPP-SubReq: %s", idstring(err, trans))
		return
	}

	//
	// Wake subs delete
	//
	subs.OngoingDelCount++
	go c.handleSubscriptionDelete(subs, trans, waitRouteCleanup_ms)
	trans.WaitEvent(0) //blocked wait as timeout is handled in subs side
	subs.OngoingDelCount--

	xapp.Logger.Debug("XAPP-SubDelReq: Handling event %s ", idstring(nil, trans, subs))

	if subs.NoRespToXapp == true {
		// Do no send delete responses to xapps due to submgr restart is deleting uncompleted subscriptions
		xapp.Logger.Debug("XAPP-SubDelReq: subs.NoRespToXapp == true")
		return
	}
}

type Registry struct {
	mutex             *sync.Mutex
	register          map[uint32]*Subscription
	subIds            []uint32
	rtmgrClient       *RtmgrClient
	restSubscriptions map[string]*RESTSubscription
}

func (r *Registry) RemoveFromSubscription(subs *Subscription, trans *TransactionXapp, waitRouteClean time.Duration, c *Control) {

	xapp.Logger.Debug("RemoveFromSubscription %s", idstring(nil, trans, subs, trans))
	r.mutex.Lock()
	defer r.mutex.Unlock()
	subs.mutex.Lock()
	defer subs.mutex.Unlock()

	delStatus := subs.EpList.DelEndpoint(trans.GetEndpoint())
	epamount := subs.EpList.Size()

	subId := subs.ReqId.InstanceId
	if delStatus == false {
		return
	}

	if waitRouteClean > 0 {
		// Wait here that response is delivered to xApp via RMR before route is cleaned
		xapp.Logger.Debug("Pending %v in order to wait route cleanup", waitRouteClean)
		r.mutex.Unlock()
		time.Sleep(waitRouteClean)
		r.mutex.Lock()
	}

	xapp.Logger.Debug("CLEAN %s", subs.String())

	if epamount == 0 {
		//
		// Subscription route delete
		//
		if subs.RMRRouteCreated == true {
			r.RouteDelete(subs, trans, c)
		}

		// Not merged subscription is being deleted
		xapp.Logger.Debug("Subscription route delete RemoveSubscriptionFromDb")
		c.RemoveSubscriptionFromDb(subs)

		//
		// Subscription release
		//

		if _, ok := r.register[subId]; ok {
			xapp.Logger.Debug("RELEASE %s", subs.String())
			delete(r.register, subId)
			xapp.Logger.Debug("Registry: substable=%v", r.register)
		}
		r.subIds = append(r.subIds, subId)
	} else if subs.EpList.Size() > 0 {
		//
		// Subscription route update
		//
		if subs.RMRRouteCreated == true {
			r.RouteDeleteUpdate(subs, c)
		}

		// Endpoint of merged subscription is being deleted
		xapp.Logger.Debug("Subscription route update WriteSubscriptionToDb")
		err := c.WriteSubscriptionToDb(subs)
		if err != nil {
			xapp.Logger.Error("tracker.UnTrackTransaction() failed:%s", err.Error())
		}
		c.UpdateCounter(cUnmergedSubscriptions)
	}
	return
}
func (r *Registry) RouteDelete(subs *Subscription, trans *TransactionXapp, c *Control) {
	tmpList := xapp.RmrEndpointList{}
	tmpList.AddEndpoint(trans.GetEndpoint())
	subRouteAction := SubRouteInfo{tmpList, uint16(subs.ReqId.InstanceId)}
	if err := r.rtmgrClient.SubscriptionRequestDelete(subRouteAction); err != nil {
		c.UpdateCounter(cRouteDeleteFail)
	}
}
func (r *Registry) RouteDeleteUpdate(subs *Subscription, c *Control) {
	subRouteAction := SubRouteInfo{subs.EpList, uint16(subs.ReqId.InstanceId)}
	if err := r.rtmgrClient.SubscriptionRequestUpdate(subRouteAction); err != nil {
		c.UpdateCounter(cRouteDeleteUpdateFail)
	}
}
func (r *Registry) GetSubscriptionFirstMatch(subIds []uint32) (*Subscription, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, subId := range subIds {
		if _, ok := r.register[subId]; ok {
			return r.register[subId], nil
		}
	}
	return nil, fmt.Errorf("No valid subscription found with subIds %v", subIds)
}

type Subscription struct {
	mutex            sync.Mutex                    // Lock
	valid            bool                          // valid
	registry         *Registry                     // Registry
	ReqId            RequestId                     // ReqId (Requestor Id + Seq Nro a.k.a subsid)
	Meid             *xapp.RMRMeid                 // Meid/RanName
	EpList           xapp.RmrEndpointList          // Endpoints
	RMRRouteCreated  bool                          // Does subscription have RMR route
	TransLock        sync.Mutex                    // Lock transactions, only one executed per time for subs
	TheTrans         TransactionIf                 // Ongoing transaction
	SubReqMsg        *e2ap.E2APSubscriptionRequest // Subscription information
	SubRFMsg         interface{}                   // Subscription information
	OngoingReqCount  int                           // Subscription create process is ongoing. In merge case it can ongoing for more than one endpoint
	OngoingDelCount  int                           // Subscription delete process is ongoing. In merge case it can ongoing for more than one endpoint
	PolicyUpdate     bool                          // This is true when policy subscrition is being updated. Used not to send delete for update after timeout or restart
	RetryFromXapp    bool                          // Retry form xApp for subscription that already exist
	SubRespRcvd      bool                          // Subscription response received
	DeleteFromDb     bool                          // Delete subscription from db
	NoRespToXapp     bool                          // Send no response for subscription delete to xApp after restart
	DoNotWaitSubResp bool                          // Test flag. Response is not waited for Subscription Request
}

func (s *Subscription) String() string {
	meidstr := "N/A"
	if s.Meid != nil {
		meidstr = s.Meid.String()
	}
	return "subs(" + s.ReqId.String() + "/" + meidstr + "/" + s.EpList.String() + ")"
}
func (s *Subscription) GetReqId() *RequestId {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return &s.ReqId
}
func (s *Subscription) GetMeid() *xapp.RMRMeid {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.Meid
}
func (s *Subscription) GetTransaction() TransactionIf {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.TheTrans
}
func (s *Subscription) WaitTransactionTurn(trans TransactionIf) {
	s.TransLock.Lock()
	s.mutex.Lock()
	s.TheTrans = trans
	s.mutex.Unlock()
}
func (s *Subscription) ReleaseTransactionTurn(trans TransactionIf) {
	s.mutex.Lock()
	if trans != nil && trans == s.TheTrans {
		s.TheTrans = nil
	}
	s.mutex.Unlock()
	s.TransLock.Unlock()
}

type RequestId struct {
	e2ap.RequestId
}
type TransactionIf interface {
	String() string
	Release()
	SendEvent(interface{}, time.Duration) (bool, bool)
	WaitEvent(time.Duration) (interface{}, bool)
}
type Transaction struct {
	mutex     sync.Mutex       //
	Seq       uint64           //transaction sequence
	tracker   *Tracker         //tracker instance
	Meid      *xapp.RMRMeid    //meid transaction related
	Mtype     int              //Encoded message type to be send
	Payload   *e2ap.PackedData //Encoded message to be send
	EventChan chan interface{}
}

func (t *Transaction) String() string {
	meidstr := "N/A"
	if t.Meid != nil {
		meidstr = t.Meid.String()
	}
	return "trans(" + strconv.FormatUint(uint64(t.Seq), 10) + "/" + meidstr + ")"
}
func (t *Transaction) SendEvent(event interface{}, waittime time.Duration) (bool, bool) {
	if waittime > 0 {
		select {
		case t.EventChan <- event:
			return true, false
		case <-time.After(waittime):
			return false, true
		}
		return false, false
	}
	t.EventChan <- event
	return true, false
}
func (t *Transaction) WaitEvent(waittime time.Duration) (interface{}, bool) {
	if waittime > 0 {
		select {
		case event := <-t.EventChan:
			return event, false
		case <-time.After(waittime):
			return nil, true
		}
	}
	event := <-t.EventChan
	return event, false
}
func (t *Transaction) GetMtype() int {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.Mtype
}
func (t *Transaction) GetMeid() *xapp.RMRMeid {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.Meid != nil {
		return t.Meid
	}
	return nil
}

type TransactionSubs struct {
	Transaction //
}

func (t *TransactionSubs) String() string {
	return "transsubs(" + t.Transaction.String() + ")"
}
func (t *TransactionSubs) Release() {
	t.mutex.Lock()
	xapp.Logger.Debug("RELEASE %s", t.String())
	t.tracker = nil
	t.mutex.Unlock()
}

type TransactionXappKey struct {
	InstanceID uint32
	xapp.RmrEndpoint
	Xid string // xapp xid in req
}

func (key *TransactionXappKey) String() string {
	return "transkey(" + key.RmrEndpoint.String() + "/" + key.Xid + ")"
}

type TransactionXapp struct {
	Transaction
	XappKey   *TransactionXappKey
	RequestId e2ap.RequestId
}

func (t *TransactionXapp) String() string {
	var transkey string = "transkey(N/A)"
	if t.XappKey != nil {
		transkey = t.XappKey.String()
	}
	return "transxapp(" + t.Transaction.String() + "/" + transkey + "/" + strconv.FormatUint(uint64(t.RequestId.InstanceId), 10) + ")"
}
func (t *TransactionXapp) GetEndpoint() *xapp.RmrEndpoint {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.XappKey != nil {
		return &t.XappKey.RmrEndpoint
	}
	return nil
}
func (t *TransactionXapp) GetXid() string {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.XappKey != nil {
		return t.XappKey.Xid
	}
	return ""
}
func (t *TransactionXapp) GetSubId() uint32 {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.RequestId.InstanceId
}
func (t *TransactionXapp) Release() {
	t.mutex.Lock()
	xapp.Logger.Debug("RELEASE %s", t.String())
	tracker := t.tracker
	xappkey := t.XappKey
	t.tracker = nil
	t.mutex.Unlock()

	if tracker != nil && xappkey != nil {
		_, err := tracker.UnTrackTransaction(*xappkey)
		if err != nil {
			xapp.Logger.Error("tracker.UnTrackTransaction() failed:%s", err.Error())
		}
	}
}

type Tracker struct {
	mutex                sync.Mutex
	transactionXappTable map[TransactionXappKey]*TransactionXapp
	transSeq             uint64
}

func (t *Tracker) Init() {
	t.transactionXappTable = make(map[TransactionXappKey]*TransactionXapp)
}
func (t *Tracker) initTransaction(transBase *Transaction) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	transBase.EventChan = make(chan interface{})
	transBase.tracker = t
	transBase.Seq = t.transSeq
	t.transSeq++
}
func (t *Tracker) NewSubsTransaction(subs *Subscription) *TransactionSubs {
	trans := &TransactionSubs{}
	trans.Meid = subs.GetMeid()
	t.initTransaction(&trans.Transaction)
	xapp.Logger.Debug("CREATE %s", trans.String())
	return trans
}
func (t *Tracker) NewXappTransaction(
	endpoint *xapp.RmrEndpoint,
	xid string,
	requestId e2ap.RequestId,
	meid *xapp.RMRMeid) *TransactionXapp {

	trans := &TransactionXapp{}
	trans.XappKey = &TransactionXappKey{requestId.Id, *endpoint, xid}
	trans.Meid = meid
	trans.RequestId = requestId
	t.initTransaction(&trans.Transaction)
	xapp.Logger.Debug("CREATE %s", trans.String())
	return trans
}
func (t *Tracker) Track(trans *TransactionXapp) error {

	if trans.GetEndpoint() == nil {
		err := fmt.Errorf("Tracker: No valid endpoint given in %s", trans.String())
		return err
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	theKey := *trans.XappKey

	if othtrans, ok := t.transactionXappTable[theKey]; ok {
		err := fmt.Errorf("Tracker: %s is ongoing, not tracking %s", othtrans, trans)
		return err
	}

	trans.tracker = t
	t.transactionXappTable[theKey] = trans
	xapp.Logger.Debug("Tracker: Append %s", trans.String())
	//xapp.Logger.Debug("Tracker: transtable=%v", t.transactionXappTable)
	return nil
}
func (t *Tracker) UnTrackTransaction(xappKey TransactionXappKey) (*TransactionXapp, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if trans, ok2 := t.transactionXappTable[xappKey]; ok2 {
		xapp.Logger.Debug("Tracker: Remove %s", trans.String())
		delete(t.transactionXappTable, xappKey)
		//xapp.Logger.Debug("Tracker: transtable=%v", t.transactionXappTable)
		return trans, nil
	}
	return nil, fmt.Errorf("Tracker: No record %v", xappKey)
}
