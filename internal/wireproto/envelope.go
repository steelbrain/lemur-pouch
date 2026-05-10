package wireproto

import "fmt"

// Binary envelope wire format — AGENTS.md "Wire Protocol > Encrypted
// Envelopes (binary frames)".
//
//	[ 1 byte  ] inner type        (0x01 = JSON control, 0x02 = file chunk)
//	[ 32 bytes] peer ed25519_pub  (destination on c2s; rewritten to source on s2c)
//	[ 24 bytes] XChaCha20-Poly1305 nonce  (random per frame)
//	[ N bytes ] ciphertext + 16-byte Poly1305 tag  (N ≥ 16; the AEAD always emits at least the tag)
//
// The relay never decrypts. Its only legal mutation is rewriting the
// 32-byte peer field from "destination" (what the sender wrote) to
// "source" (the sender's authenticated identity from the connection)
// before forwarding to the recipient — see RewriteDestinationToSource.
//
// The 1-byte inner-type prefix is the AEAD's additional authenticated
// data, so any tamper there fails decrypt at the recipient. The 32-byte
// peer field is NOT in the AAD because the relay legitimately rewrites
// it; tampering by a malicious relay is detected indirectly via per-pair
// session keys (a wrongly-routed envelope decrypts under a key the
// recipient doesn't share with the supposed sender, so the AEAD fails).

// Sizes for the binary envelope's fixed header + minimum payload.
const (
	EnvelopePeerKeyLen   = 32
	EnvelopeNonceLen     = 24
	EnvelopeHeaderLen    = 1 + EnvelopePeerKeyLen + EnvelopeNonceLen // = 57
	EnvelopeMinSealedLen = 16                                        // Poly1305 tag — XChaCha20-Poly1305 always emits at least this
	EnvelopeMinFrameLen  = EnvelopeHeaderLen + EnvelopeMinSealedLen  // = 73
)

// Inner-type discriminator values per AGENTS.md. Defined so callers don't
// hardcode magic numbers; the relay does NOT enforce the set (it forwards
// any inner type for forward compat — recipients drop unknown values).
const (
	InnerTypeJSONControl byte = 0x01
	InnerTypeFileChunk   byte = 0x02
)

// EnvelopeHeader is the parsed fixed prefix of a binary envelope frame.
//
// PeerKey and Nonce alias slices of the source frame buffer — they are
// NOT freshly-allocated copies. Callers reading these fields after the
// frame buffer is reused (or after RewriteDestinationToSource is called
// on it) will observe the mutation. Use bytes.Clone if you need a stable
// snapshot.
type EnvelopeHeader struct {
	InnerType byte
	PeerKey   []byte // length 32; aliases frame[1:33]
	Nonce     []byte // length 24; aliases frame[33:57]
}

// ParseEnvelopeHeader extracts the fixed prefix of a binary envelope
// frame and returns the (header, sealed) tuple. sealed aliases
// frame[57:].
//
// Length-only validation: this function does NOT validate the inner-type
// byte. Unknown inner types are forwarded by the relay so a future
// inner-type addition (e.g. an inner type for a v1.x extension) doesn't
// require a relay redeploy; recipients that don't understand a value
// drop the frame.
//
// Returns an error if frame is shorter than EnvelopeMinFrameLen.
func ParseEnvelopeHeader(frame []byte) (EnvelopeHeader, []byte, error) {
	if len(frame) < EnvelopeMinFrameLen {
		return EnvelopeHeader{}, nil, fmt.Errorf(
			"wireproto: envelope frame too short: %d bytes (min %d)",
			len(frame), EnvelopeMinFrameLen,
		)
	}
	return EnvelopeHeader{
		InnerType: frame[0],
		PeerKey:   frame[1:33],
		Nonce:     frame[33:57],
	}, frame[EnvelopeHeaderLen:], nil
}

// RewriteDestinationToSource overwrites the 32-byte peer-identity field
// (bytes 1..33) of an envelope frame in place. This is the relay's
// forward-path mutation — replace what the sender wrote (the destination
// identity) with the authenticated source identity from the sender's
// connection.
//
// Errors:
//   - frame shorter than EnvelopeHeaderLen
//   - sourceKey not exactly EnvelopePeerKeyLen bytes
func RewriteDestinationToSource(frame, sourceKey []byte) error {
	if len(frame) < EnvelopeHeaderLen {
		return fmt.Errorf(
			"wireproto: envelope frame too short for peer-key rewrite: %d bytes (need %d)",
			len(frame), EnvelopeHeaderLen,
		)
	}
	if len(sourceKey) != EnvelopePeerKeyLen {
		return fmt.Errorf(
			"wireproto: source key must be %d bytes, got %d",
			EnvelopePeerKeyLen, len(sourceKey),
		)
	}
	copy(frame[1:33], sourceKey)
	return nil
}

// MarshalEnvelope concatenates the header parts and sealed payload into
// a single frame ready to send. Used by tests and by future Go-side
// clients (if any). The relay does NOT call this — it forwards
// already-formed frames in place via RewriteDestinationToSource.
//
// Validates that peerKey is EnvelopePeerKeyLen bytes, nonce is
// EnvelopeNonceLen bytes, and sealed is at least EnvelopeMinSealedLen
// bytes (the Poly1305 tag — every well-formed AEAD output has it).
func MarshalEnvelope(innerType byte, peerKey, nonce, sealed []byte) ([]byte, error) {
	if len(peerKey) != EnvelopePeerKeyLen {
		return nil, fmt.Errorf(
			"wireproto: peer key must be %d bytes, got %d",
			EnvelopePeerKeyLen, len(peerKey),
		)
	}
	if len(nonce) != EnvelopeNonceLen {
		return nil, fmt.Errorf(
			"wireproto: nonce must be %d bytes, got %d",
			EnvelopeNonceLen, len(nonce),
		)
	}
	if len(sealed) < EnvelopeMinSealedLen {
		return nil, fmt.Errorf(
			"wireproto: sealed payload must be at least %d bytes (Poly1305 tag), got %d",
			EnvelopeMinSealedLen, len(sealed),
		)
	}
	out := make([]byte, EnvelopeHeaderLen+len(sealed))
	out[0] = innerType
	copy(out[1:33], peerKey)
	copy(out[33:57], nonce)
	copy(out[EnvelopeHeaderLen:], sealed)
	return out, nil
}
