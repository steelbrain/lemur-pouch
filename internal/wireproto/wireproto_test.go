package wireproto

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPeekType(t *testing.T) {
	cases := []struct {
		name    string
		data    string
		want    string
		wantErr bool
	}{
		{"challenge", `{"type":"challenge","nonce":"AAAA"}`, TypeChallenge, false},
		{"identify", `{"type":"identify"}`, TypeIdentify, false},
		{"welcome", `{"type":"welcome","you":{}}`, TypeWelcome, false},
		{"error", `{"type":"error","code":"x","message":"y"}`, TypeError, false},
		{"unknown type still returns it", `{"type":"future-message-type"}`, "future-message-type", false},
		{"empty type field", `{"type":""}`, "", false},
		{"missing type field", `{"foo":"bar"}`, "", false},
		{"malformed json", `{not json`, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PeekType([]byte(tc.data))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("PeekType = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMarshalChallengeSetsType(t *testing.T) {
	data, err := MarshalChallenge(ChallengeMsg{Nonce: []byte{1, 2, 3, 4}})
	if err != nil {
		t.Fatalf("MarshalChallenge: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypeChallenge {
		t.Errorf("type field = %q, want %q", typ, TypeChallenge)
	}
	// Confirm Nonce round-trips through standard base64 (Go's default for []byte).
	var rt ChallengeMsg
	if err := json.Unmarshal(data, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(rt.Nonce, []byte{1, 2, 3, 4}) {
		t.Errorf("nonce round-trip: got %x, want 01020304", rt.Nonce)
	}
}

func TestMarshalIdentifyRoundTrip(t *testing.T) {
	in := IdentifyMsg{
		Ed25519Pub:  bytes.Repeat([]byte{0xAA}, 32),
		X25519Pub:   bytes.Repeat([]byte{0xBB}, 32),
		SigLiveness: bytes.Repeat([]byte{0xCC}, 64),
		SigBinding:  bytes.Repeat([]byte{0xDD}, 64),
	}
	data, err := MarshalIdentify(in)
	if err != nil {
		t.Fatalf("MarshalIdentify: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypeIdentify {
		t.Errorf("type field = %q, want %q", typ, TypeIdentify)
	}
	var out IdentifyMsg
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(out.Ed25519Pub, in.Ed25519Pub) ||
		!bytes.Equal(out.X25519Pub, in.X25519Pub) ||
		!bytes.Equal(out.SigLiveness, in.SigLiveness) ||
		!bytes.Equal(out.SigBinding, in.SigBinding) {
		t.Fatal("identify round-trip lost or mangled bytes")
	}
}

func TestMarshalWelcomeRoundTrip(t *testing.T) {
	in := WelcomeMsg{
		You: PeerRecord{
			Ed25519Pub: bytes.Repeat([]byte{0x11}, 32),
			X25519Pub:  bytes.Repeat([]byte{0x22}, 32),
			SigBinding: bytes.Repeat([]byte{0x33}, 64),
			IP:         "192.168.1.42",
			Port:       54321,
		},
	}
	data, err := MarshalWelcome(in)
	if err != nil {
		t.Fatalf("MarshalWelcome: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypeWelcome {
		t.Errorf("type field = %q, want %q", typ, TypeWelcome)
	}
	var out WelcomeMsg
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.You.IP != in.You.IP {
		t.Errorf("IP: got %q, want %q", out.You.IP, in.You.IP)
	}
	if out.You.Port != in.You.Port {
		t.Errorf("Port: got %d, want %d", out.You.Port, in.You.Port)
	}
	if !bytes.Equal(out.You.Ed25519Pub, in.You.Ed25519Pub) {
		t.Errorf("Ed25519Pub round-trip mismatch")
	}
}

func TestMarshalErrorRoundTrip(t *testing.T) {
	in := ErrorMsg{Code: ErrCodeInvalidSignature, Message: "sig_liveness verification failed"}
	data, err := MarshalError(in)
	if err != nil {
		t.Fatalf("MarshalError: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypeError {
		t.Errorf("type field = %q, want %q", typ, TypeError)
	}
	var out ErrorMsg
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != in.Code || out.Message != in.Message {
		t.Errorf("error round-trip: got %+v, want %+v", out, in)
	}
}

// TestNilByteFieldsMarshalAsNull pins encoding/json's behavior for nil
// []byte fields on the wire structs: nil marshals to the literal `null`
// (not `""`, not base64 of the empty string). Callers MUST populate every
// []byte field before marshaling — a refactor that quietly changes this
// (e.g. by switching to a custom MarshalJSON or adding `omitempty`) will
// fail this test loudly so the TS side isn't blindsided.
func TestNilByteFieldsMarshalAsNull(t *testing.T) {
	data, err := MarshalChallenge(ChallengeMsg{})
	if err != nil {
		t.Fatalf("MarshalChallenge: %v", err)
	}
	if !strings.Contains(string(data), `"nonce":null`) {
		t.Fatalf("expected nil Nonce to marshal as null, got: %s", data)
	}
}

// TestSpecConformantJSONFieldNames pins the JSON field names to the spec
// (AGENTS.md "Wire Protocol > Cleartext Control"). Renaming a struct field's
// json tag here will break the wire compat with the TS side.
func TestSpecConformantJSONFieldNames(t *testing.T) {
	data, err := MarshalIdentify(IdentifyMsg{
		Ed25519Pub:  []byte{1},
		X25519Pub:   []byte{2},
		SigLiveness: []byte{3},
		SigBinding:  []byte{4},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	for _, key := range []string{
		`"type":"identify"`,
		`"ed25519_pub":`,
		`"x25519_pub":`,
		`"sig_liveness":`,
		`"sig_binding":`,
	} {
		if !strings.Contains(s, key) {
			t.Errorf("identify JSON missing %q\n got: %s", key, s)
		}
	}

	data, err = MarshalWelcome(WelcomeMsg{You: PeerRecord{
		Ed25519Pub: []byte{1},
		X25519Pub:  []byte{2},
		SigBinding: []byte{3},
		IP:         "x",
		Port:       1,
	}})
	if err != nil {
		t.Fatalf("marshal welcome: %v", err)
	}
	s = string(data)
	for _, key := range []string{
		`"type":"welcome"`,
		`"you":`,
		`"ed25519_pub":`,
		`"x25519_pub":`,
		`"sig_binding":`,
		`"ip":`,
		`"port":`,
	} {
		if !strings.Contains(s, key) {
			t.Errorf("welcome JSON missing %q\n got: %s", key, s)
		}
	}
}

// TestSpecConformantTypeStrings pins the literal string values of every
// type-discriminator constant in this package. The forward-reference
// constants (peer-list, peer-joined, …) aren't yet used by the relay, so
// without this test a typo would only surface when the layer that
// consumes the constant lands. Pinning them here catches any drift the
// moment it's introduced — well before the consumer is wired up.
func TestSpecConformantTypeStrings(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"TypeChallenge", TypeChallenge, "challenge"},
		{"TypeIdentify", TypeIdentify, "identify"},
		{"TypeWelcome", TypeWelcome, "welcome"},
		{"TypeError", TypeError, "error"},
		{"TypePeerList", TypePeerList, "peer-list"},
		{"TypePeerJoined", TypePeerJoined, "peer-joined"},
		{"TypePeerLeft", TypePeerLeft, "peer-left"},
		{"TypeInvite", TypeInvite, "invite"},
		{"TypeAccept", TypeAccept, "accept"},
		{"TypeReject", TypeReject, "reject"},
		{"TypeInviteFrom", TypeInviteFrom, "invite-from"},
		{"TypeAcceptFrom", TypeAcceptFrom, "accept-from"},
		{"TypeRejectFrom", TypeRejectFrom, "reject-from"},
		{"TypeInviteDeferred", TypeInviteDeferred, "invite-deferred"},
		{"TypeInviteAutoRejected", TypeInviteAutoRejected, "invite-auto-rejected"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
