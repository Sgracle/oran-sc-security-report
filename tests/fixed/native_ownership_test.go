package control

import (
	"testing"

	"ownershiplab/e2ap"
	"ownershiplab/xapp"
)

// The stored Requestor ID is xApp-facing (900); E2 requests are sent as 123.
func TestSubscriptionMatchesDeleteRequired(t *testing.T) {
	for _, tc := range []struct {
		name                string
		source, owner       *xapp.RMRMeid
		requestor, instance uint32
		function            e2ap.FunctionId
		missingConfig, want bool
	}{
		{"own B", &xapp.RMRMeid{RanName: "B"}, &xapp.RMRMeid{RanName: "B"}, 123, 102, 2, false, true},
		{"foreign A", &xapp.RMRMeid{RanName: "A"}, &xapp.RMRMeid{RanName: "B"}, 123, 102, 2, false, false},
		{"missing source", nil, &xapp.RMRMeid{RanName: "B"}, 123, 102, 2, false, false},
		{"empty names", &xapp.RMRMeid{}, &xapp.RMRMeid{}, 123, 102, 2, false, false},
		{"missing owner", &xapp.RMRMeid{RanName: "B"}, nil, 123, 102, 2, false, false},
		{"xapp requestor is not wire requestor", &xapp.RMRMeid{RanName: "B"}, &xapp.RMRMeid{RanName: "B"}, 900, 102, 2, false, false},
		{"wrong instance", &xapp.RMRMeid{RanName: "B"}, &xapp.RMRMeid{RanName: "B"}, 123, 103, 2, false, false},
		{"wrong function", &xapp.RMRMeid{RanName: "B"}, &xapp.RMRMeid{RanName: "B"}, 123, 102, 3, false, false},
		{"missing config", &xapp.RMRMeid{RanName: "B"}, &xapp.RMRMeid{RanName: "B"}, 123, 102, 2, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Subscription{Meid: tc.owner, ReqId: RequestId{RequestId: e2ap.RequestId{Id: 900, InstanceId: 102}}, SubReqMsg: &e2ap.E2APSubscriptionRequest{FunctionId: 2}}
			if tc.missingConfig {
				s.SubReqMsg = nil
			}
			required := e2ap.E2APSubscriptionDeleteRequired{RequestId: e2ap.RequestId{Id: tc.requestor, InstanceId: tc.instance}, FunctionId: tc.function}
			if got := s.matchesDeleteRequired(tc.source, required); got != tc.want {
				t.Errorf("matchesDeleteRequired=%v want=%v", got, tc.want)
			}
		})
	}
}
