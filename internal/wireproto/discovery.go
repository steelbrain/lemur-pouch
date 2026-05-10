package wireproto

import "encoding/json"

// Discovery message types — sent by the relay to keep clients informed of
// the live peer set. See AGENTS.md "Wire Protocol > Cleartext Control" and
// "Discovery". The relay pushes:
//
//   - PeerListMsg immediately after WelcomeMsg (a snapshot of the current
//     peer set, excluding the recipient itself).
//   - PeerJoinedMsg to all other peers when a new peer completes its
//     connection handshake.
//   - PeerLeftMsg to all remaining peers when a peer's WebSocket closes.
//
// All three are cleartext JSON: discovery happens before any friendship is
// established and there's no per-pair session key yet.

// PeerListMsg is the relay's snapshot of currently-connected peers, sent
// to a newly-connected client immediately after WelcomeMsg. The recipient's
// own record is excluded so a peer never has to filter itself out of the
// list it just received.
type PeerListMsg struct {
	Type  string       `json:"type"`
	Peers []PeerRecord `json:"peers"`
}

// PeerJoinedMsg is the relay's broadcast that a new peer has just completed
// the handshake. Sent to every peer except the one being announced.
type PeerJoinedMsg struct {
	Type string     `json:"type"`
	Peer PeerRecord `json:"peer"`
}

// PeerLeftMsg is the relay's broadcast that a peer has disconnected. Sent
// to every remaining peer once the leaving peer's WebSocket has closed and
// the hub has dropped its record.
//
// Only the Ed25519Pub is carried — that's enough for clients to remove the
// matching row from their local discovery list. The full PeerRecord isn't
// needed because the receiver already has it from the prior PeerListMsg or
// PeerJoinedMsg that introduced this peer.
type PeerLeftMsg struct {
	Type       string `json:"type"`
	Ed25519Pub []byte `json:"ed25519_pub"`
}

// MarshalPeerList / MarshalPeerJoined / MarshalPeerLeft mirror the
// MarshalChallenge / MarshalIdentify / MarshalWelcome / MarshalError
// pattern in wireproto.go: each sets the Type field to the spec-mandated
// constant before JSON-encoding so callers can't construct a struct with
// the wrong discriminator.

func MarshalPeerList(m PeerListMsg) ([]byte, error) {
	m.Type = TypePeerList
	return json.Marshal(m)
}

func MarshalPeerJoined(m PeerJoinedMsg) ([]byte, error) {
	m.Type = TypePeerJoined
	return json.Marshal(m)
}

func MarshalPeerLeft(m PeerLeftMsg) ([]byte, error) {
	m.Type = TypePeerLeft
	return json.Marshal(m)
}
