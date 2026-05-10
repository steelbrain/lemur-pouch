package wireproto

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalPeerListSetsType(t *testing.T) {
	in := PeerListMsg{
		Peers: []PeerRecord{
			{
				Ed25519Pub: bytes.Repeat([]byte{0x11}, 32),
				X25519Pub:  bytes.Repeat([]byte{0x22}, 32),
				SigBinding: bytes.Repeat([]byte{0x33}, 64),
				IP:         "192.168.1.42",
				Port:       54321,
			},
		},
	}
	data, err := MarshalPeerList(in)
	if err != nil {
		t.Fatalf("MarshalPeerList: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypePeerList {
		t.Errorf("type = %q, want %q", typ, TypePeerList)
	}

	var out PeerListMsg
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Peers) != 1 {
		t.Fatalf("peers length = %d, want 1", len(out.Peers))
	}
	if !bytes.Equal(out.Peers[0].Ed25519Pub, in.Peers[0].Ed25519Pub) ||
		!bytes.Equal(out.Peers[0].X25519Pub, in.Peers[0].X25519Pub) ||
		!bytes.Equal(out.Peers[0].SigBinding, in.Peers[0].SigBinding) {
		t.Error("peer-list peer round-trip lost bytes")
	}
	if out.Peers[0].IP != "192.168.1.42" || out.Peers[0].Port != 54321 {
		t.Errorf("peer ip/port: got %s:%d", out.Peers[0].IP, out.Peers[0].Port)
	}
}

func TestMarshalPeerListEmpty(t *testing.T) {
	// A peer-list with no other peers should still produce valid JSON
	// with an empty (or null) array — the receiver tolerates either.
	data, err := MarshalPeerList(PeerListMsg{})
	if err != nil {
		t.Fatalf("MarshalPeerList: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypePeerList {
		t.Errorf("type = %q, want %q", typ, TypePeerList)
	}
	// Round-trip an explicitly empty []PeerRecord{} too — should yield "[]"
	// (encoding/json marshals nil slice as null but empty slice as []).
	data, err = MarshalPeerList(PeerListMsg{Peers: []PeerRecord{}})
	if err != nil {
		t.Fatalf("MarshalPeerList (empty slice): %v", err)
	}
	if !strings.Contains(string(data), `"peers":[]`) {
		t.Errorf("expected peers:[] for empty slice, got: %s", data)
	}
}

func TestMarshalPeerJoinedRoundTrip(t *testing.T) {
	in := PeerJoinedMsg{
		Peer: PeerRecord{
			Ed25519Pub: bytes.Repeat([]byte{0xAA}, 32),
			X25519Pub:  bytes.Repeat([]byte{0xBB}, 32),
			SigBinding: bytes.Repeat([]byte{0xCC}, 64),
			IP:         "10.0.0.5",
			Port:       12345,
		},
	}
	data, err := MarshalPeerJoined(in)
	if err != nil {
		t.Fatalf("MarshalPeerJoined: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypePeerJoined {
		t.Errorf("type = %q, want %q", typ, TypePeerJoined)
	}
	var out PeerJoinedMsg
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(out.Peer.Ed25519Pub, in.Peer.Ed25519Pub) ||
		!bytes.Equal(out.Peer.X25519Pub, in.Peer.X25519Pub) ||
		!bytes.Equal(out.Peer.SigBinding, in.Peer.SigBinding) {
		t.Error("peer-joined peer round-trip lost bytes")
	}
}

func TestMarshalPeerLeftRoundTrip(t *testing.T) {
	in := PeerLeftMsg{Ed25519Pub: bytes.Repeat([]byte{0xDD}, 32)}
	data, err := MarshalPeerLeft(in)
	if err != nil {
		t.Fatalf("MarshalPeerLeft: %v", err)
	}
	if typ, _ := PeekType(data); typ != TypePeerLeft {
		t.Errorf("type = %q, want %q", typ, TypePeerLeft)
	}
	var out PeerLeftMsg
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(out.Ed25519Pub, in.Ed25519Pub) {
		t.Errorf("ed25519_pub round-trip mismatch")
	}
}

// TestDiscoveryJSONFieldNames pins the JSON field names for the discovery
// message types — same role as TestSpecConformantJSONFieldNames does for
// the handshake messages. Renaming `peers`, `peer`, or `ed25519_pub` here
// silently breaks Go-TS interop on the discovery layer.
func TestDiscoveryJSONFieldNames(t *testing.T) {
	plData, err := MarshalPeerList(PeerListMsg{Peers: []PeerRecord{
		{Ed25519Pub: []byte{1}, X25519Pub: []byte{2}, SigBinding: []byte{3}, IP: "x", Port: 1},
	}})
	if err != nil {
		t.Fatalf("MarshalPeerList: %v", err)
	}
	for _, key := range []string{
		`"type":"peer-list"`,
		`"peers":`,
		`"ed25519_pub":`,
		`"x25519_pub":`,
		`"sig_binding":`,
	} {
		if !strings.Contains(string(plData), key) {
			t.Errorf("peer-list JSON missing %q\n got: %s", key, plData)
		}
	}

	pjData, err := MarshalPeerJoined(PeerJoinedMsg{Peer: PeerRecord{IP: "x", Port: 1}})
	if err != nil {
		t.Fatalf("MarshalPeerJoined: %v", err)
	}
	for _, key := range []string{`"type":"peer-joined"`, `"peer":`} {
		if !strings.Contains(string(pjData), key) {
			t.Errorf("peer-joined JSON missing %q\n got: %s", key, pjData)
		}
	}

	plLeftData, err := MarshalPeerLeft(PeerLeftMsg{Ed25519Pub: []byte{1}})
	if err != nil {
		t.Fatalf("MarshalPeerLeft: %v", err)
	}
	for _, key := range []string{`"type":"peer-left"`, `"ed25519_pub":`} {
		if !strings.Contains(string(plLeftData), key) {
			t.Errorf("peer-left JSON missing %q\n got: %s", key, plLeftData)
		}
	}
}
