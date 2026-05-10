package wireproto

import "encoding/json"

// Friendship message types — AGENTS.md "Wire Protocol > Cleartext Control"
// ("Friendship handshake" + "Queue/log signals") and "Anti-Abuse: Per-IP
// Rate Limiting".
//
// The friendship handshake establishes mutual consent before any encrypted
// envelope (transfer offer, file chunks) flows between two peers. The
// messages split into two shapes:
//
//   - Client-to-server directives: pure consent + a target identity.
//     {type, to} — invite, accept, reject.
//
//   - Server-to-client notifications: forwards of the above plus queue/log
//     signals. {type, from} — invite-from, accept-from, reject-from,
//     invite-deferred, invite-auto-rejected.
//
// Per the spec the relay enforces a per-(sender_IP, recipient) invite gate:
// at most one active pending invite per pair; further invites queue. If
// the first invite is rejected, subsequent invites are auto-rejected and
// dropped into a per-recipient log. Identity churn doesn't help an
// attacker because the budget is per-IP, not per-identity.

// FriendshipDirective is the single struct for all three c2s friendship
// messages (invite, accept, reject) — they share the same shape, just
// different Type discriminators. To is the target peer's ed25519_pub.
type FriendshipDirective struct {
	Type string `json:"type"`
	To   []byte `json:"to"`
}

// FriendshipNotification is the single struct for all five s2c friendship
// messages (invite-from / accept-from / reject-from / invite-deferred /
// invite-auto-rejected). From is the originating peer's ed25519_pub:
//
//   - invite-from / accept-from / reject-from: the peer the directive
//     came from (or, in the case of accept-from / reject-from, the peer
//     who responded to your invite).
//   - invite-deferred: the sender of the queued invite that just became
//     active because the previous active one was accepted.
//   - invite-auto-rejected: the sender whose invite was logged because
//     the recipient had previously rejected an invite from that IP.
type FriendshipNotification struct {
	Type string `json:"type"`
	From []byte `json:"from"`
}

// MarshalInvite / MarshalAccept / MarshalReject set the Type field to the
// spec-mandated discriminator and JSON-encode. The Type-from-constant
// pattern matches MarshalChallenge / MarshalIdentify / MarshalWelcome /
// MarshalError in wireproto.go: callers can't accidentally produce a
// directive with the wrong type.

func MarshalInvite(to []byte) ([]byte, error) {
	return json.Marshal(FriendshipDirective{Type: TypeInvite, To: to})
}

func MarshalAccept(to []byte) ([]byte, error) {
	return json.Marshal(FriendshipDirective{Type: TypeAccept, To: to})
}

func MarshalReject(to []byte) ([]byte, error) {
	return json.Marshal(FriendshipDirective{Type: TypeReject, To: to})
}

func MarshalInviteFrom(from []byte) ([]byte, error) {
	return json.Marshal(FriendshipNotification{Type: TypeInviteFrom, From: from})
}

func MarshalAcceptFrom(from []byte) ([]byte, error) {
	return json.Marshal(FriendshipNotification{Type: TypeAcceptFrom, From: from})
}

func MarshalRejectFrom(from []byte) ([]byte, error) {
	return json.Marshal(FriendshipNotification{Type: TypeRejectFrom, From: from})
}

func MarshalInviteDeferred(from []byte) ([]byte, error) {
	return json.Marshal(FriendshipNotification{Type: TypeInviteDeferred, From: from})
}

func MarshalInviteAutoRejected(from []byte) ([]byte, error) {
	return json.Marshal(FriendshipNotification{Type: TypeInviteAutoRejected, From: from})
}
