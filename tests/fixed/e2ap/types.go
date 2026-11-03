// Test-only wire data shapes. JSON replaces ASN.1; no network encoder exists.
package e2ap

type RequestId struct{ Id, InstanceId uint32 }

func (r RequestId) String() string { return "synthetic-request" }

type FunctionId uint16
type Cause struct{ Content, Value uint8 }
type PackedData struct{ Buf []byte }
type E2APSubscriptionRequest struct {
	RequestId
	FunctionId
}
type E2APSubscriptionDeleteRequest struct {
	RequestId
	FunctionId
}
type E2APSubscriptionDeleteResponse struct {
	RequestId
	FunctionId
}
type E2APSubscriptionDeleteFailure struct {
	RequestId
	FunctionId
	Cause
}
type E2APSubscriptionDeleteRequired struct {
	RequestId
	FunctionId
	Cause
}
type SubscriptionDeleteRequiredList struct {
	E2APSubscriptionDeleteRequiredRequests []E2APSubscriptionDeleteRequired
}
