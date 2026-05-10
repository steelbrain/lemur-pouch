package wireproto

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// roundTripDirective marshals via the helper, peeks the type, and
// re-unmarshals into a FriendshipDirective for byte-equality checks.
func roundTripDirective(t *testing.T, marshal func([]byte) ([]byte, error), wantType string, to []byte) FriendshipDirective {
	t.Helper()
	data, err := marshal(to)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, _ := PeekType(data); got != wantType {
		t.Errorf("type field = %q, want %q", got, wantType)
	}
	var out FriendshipDirective
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != wantType {
		t.Errorf("unmarshalled Type = %q, want %q", out.Type, wantType)
	}
	if !bytes.Equal(out.To, to) {
		t.Errorf("To round-trip lost bytes")
	}
	return out
}

// roundTripNotification mirrors roundTripDirective for the s2c shape.
func roundTripNotification(t *testing.T, marshal func([]byte) ([]byte, error), wantType string, from []byte) FriendshipNotification {
	t.Helper()
	data, err := marshal(from)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, _ := PeekType(data); got != wantType {
		t.Errorf("type field = %q, want %q", got, wantType)
	}
	var out FriendshipNotification
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != wantType {
		t.Errorf("unmarshalled Type = %q, want %q", out.Type, wantType)
	}
	if !bytes.Equal(out.From, from) {
		t.Errorf("From round-trip lost bytes")
	}
	return out
}

func TestMarshalInviteRoundTrip(t *testing.T) {
	to := bytes.Repeat([]byte{0xAA}, 32)
	roundTripDirective(t, MarshalInvite, TypeInvite, to)
}

func TestMarshalAcceptRoundTrip(t *testing.T) {
	to := bytes.Repeat([]byte{0xBB}, 32)
	roundTripDirective(t, MarshalAccept, TypeAccept, to)
}

func TestMarshalRejectRoundTrip(t *testing.T) {
	to := bytes.Repeat([]byte{0xCC}, 32)
	roundTripDirective(t, MarshalReject, TypeReject, to)
}

func TestMarshalInviteFromRoundTrip(t *testing.T) {
	from := bytes.Repeat([]byte{0x11}, 32)
	roundTripNotification(t, MarshalInviteFrom, TypeInviteFrom, from)
}

func TestMarshalAcceptFromRoundTrip(t *testing.T) {
	from := bytes.Repeat([]byte{0x22}, 32)
	roundTripNotification(t, MarshalAcceptFrom, TypeAcceptFrom, from)
}

func TestMarshalRejectFromRoundTrip(t *testing.T) {
	from := bytes.Repeat([]byte{0x33}, 32)
	roundTripNotification(t, MarshalRejectFrom, TypeRejectFrom, from)
}

func TestMarshalInviteDeferredRoundTrip(t *testing.T) {
	from := bytes.Repeat([]byte{0x44}, 32)
	roundTripNotification(t, MarshalInviteDeferred, TypeInviteDeferred, from)
}

func TestMarshalInviteAutoRejectedRoundTrip(t *testing.T) {
	from := bytes.Repeat([]byte{0x55}, 32)
	roundTripNotification(t, MarshalInviteAutoRejected, TypeInviteAutoRejected, from)
}

// TestFriendshipJSONFieldNames pins the JSON field names for the friendship
// messages — same role as TestSpecConformantJSONFieldNames does for the
// handshake messages and TestDiscoveryJSONFieldNames does for discovery.
// Renaming `to` or `from` here silently breaks Go-TS interop on the
// friendship layer.
func TestFriendshipJSONFieldNames(t *testing.T) {
	to := []byte{0x01}
	from := []byte{0x02}

	directiveCases := []struct {
		name    string
		marshal func([]byte) ([]byte, error)
		typ     string
	}{
		{"invite", MarshalInvite, TypeInvite},
		{"accept", MarshalAccept, TypeAccept},
		{"reject", MarshalReject, TypeReject},
	}
	for _, tc := range directiveCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.marshal(to)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			s := string(data)
			for _, key := range []string{
				`"type":"` + tc.typ + `"`,
				`"to":`,
			} {
				if !strings.Contains(s, key) {
					t.Errorf("%s JSON missing %q\n got: %s", tc.name, key, s)
				}
			}
		})
	}

	notificationCases := []struct {
		name    string
		marshal func([]byte) ([]byte, error)
		typ     string
	}{
		{"invite-from", MarshalInviteFrom, TypeInviteFrom},
		{"accept-from", MarshalAcceptFrom, TypeAcceptFrom},
		{"reject-from", MarshalRejectFrom, TypeRejectFrom},
		{"invite-deferred", MarshalInviteDeferred, TypeInviteDeferred},
		{"invite-auto-rejected", MarshalInviteAutoRejected, TypeInviteAutoRejected},
	}
	for _, tc := range notificationCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.marshal(from)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			s := string(data)
			for _, key := range []string{
				`"type":"` + tc.typ + `"`,
				`"from":`,
			} {
				if !strings.Contains(s, key) {
					t.Errorf("%s JSON missing %q\n got: %s", tc.name, key, s)
				}
			}
		})
	}
}
