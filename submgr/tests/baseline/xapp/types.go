// Test-only external framework boundary. No sockets, HTTP, RMR or SDL client.
package xapp

type RMRMeid struct{ PlmnID, EnbID, RanName string }

func (m *RMRMeid) String() string {
	if m == nil {
		return "nil"
	}
	return m.RanName
}

type RMRParams struct {
	Mtype, SubId, PayloadLen int
	Meid                     *RMRMeid
	Src, Xid                 string
	Payload                  []byte
	Mbuf                     interface{}
}

func (p *RMRParams) String() string { return "synthetic-internal-message" }

type RmrEndpoint struct{ Addr string }

func NewRmrEndpoint(s string) *RmrEndpoint { return &RmrEndpoint{Addr: s} }
func (e *RmrEndpoint) String() string      { return e.Addr }

type RmrEndpointList struct{ Endpoints []RmrEndpoint }

func (l *RmrEndpointList) AddEndpoint(e *RmrEndpoint) bool {
	if l.HasEndpoint(e) {
		return false
	}
	l.Endpoints = append(l.Endpoints, *e)
	return true
}
func (l *RmrEndpointList) HasEndpoint(e *RmrEndpoint) bool {
	for _, p := range l.Endpoints {
		if p == *e {
			return true
		}
	}
	return false
}
func (l *RmrEndpointList) DelEndpoint(e *RmrEndpoint) bool {
	for i, p := range l.Endpoints {
		if p == *e {
			l.Endpoints = append(l.Endpoints[:i], l.Endpoints[i+1:]...)
			return true
		}
	}
	return false
}
func (l *RmrEndpointList) Size() int      { return len(l.Endpoints) }
func (l *RmrEndpointList) String() string { return "synthetic-endpoints" }

type logger struct{}

var Logger logger

func (logger) Debug(string, ...interface{}) {}
func (logger) Info(string, ...interface{})  {}
func (logger) Error(string, ...interface{}) {}
