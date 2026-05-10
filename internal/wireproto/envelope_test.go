package wireproto

import (
	"bytes"
	"strings"
	"testing"
)

// envelopeFixture builds a deterministic envelope frame for round-trip /
// rewrite tests. Each region is filled with a distinct byte so a
// misaligned slice or off-by-one is obvious in the failure output.
func envelopeFixture(t *testing.T, innerType byte, sealedLen int) (frame, peerKey, nonce, sealed []byte) {
	t.Helper()
	if sealedLen < EnvelopeMinSealedLen {
		t.Fatalf("envelopeFixture: sealedLen %d below minimum %d", sealedLen, EnvelopeMinSealedLen)
	}
	peerKey = bytes.Repeat([]byte{0xAA}, EnvelopePeerKeyLen)
	nonce = bytes.Repeat([]byte{0xBB}, EnvelopeNonceLen)
	sealed = bytes.Repeat([]byte{0xCC}, sealedLen)
	frame, err := MarshalEnvelope(innerType, peerKey, nonce, sealed)
	if err != nil {
		t.Fatalf("envelopeFixture: MarshalEnvelope: %v", err)
	}
	return frame, peerKey, nonce, sealed
}

func TestEnvelopeHeaderConstants(t *testing.T) {
	// Pin the wire layout against accidental refactor: 1+32+24 = 57 header,
	// +16 minimum sealed (Poly1305 tag) = 73 minimum frame.
	if EnvelopeHeaderLen != 57 {
		t.Errorf("EnvelopeHeaderLen = %d, want 57", EnvelopeHeaderLen)
	}
	if EnvelopeMinFrameLen != 73 {
		t.Errorf("EnvelopeMinFrameLen = %d, want 73", EnvelopeMinFrameLen)
	}
}

func TestMarshalParseRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		innerType byte
		sealedLen int
	}{
		{"json-control-min", InnerTypeJSONControl, EnvelopeMinSealedLen},
		{"json-control-typical", InnerTypeJSONControl, 256},
		{"file-chunk-typical", InnerTypeFileChunk, 64 * 1024},
		// Unknown inner type is allowed at the wire layer (forward-compat;
		// see ParseEnvelopeHeader docstring).
		{"unknown-inner-type", 0xFF, EnvelopeMinSealedLen},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame, peerKey, nonce, sealed := envelopeFixture(t, tc.innerType, tc.sealedLen)

			if len(frame) != EnvelopeHeaderLen+tc.sealedLen {
				t.Fatalf("frame length = %d, want %d", len(frame), EnvelopeHeaderLen+tc.sealedLen)
			}

			hdr, gotSealed, err := ParseEnvelopeHeader(frame)
			if err != nil {
				t.Fatalf("ParseEnvelopeHeader: %v", err)
			}
			if hdr.InnerType != tc.innerType {
				t.Errorf("InnerType = 0x%02x, want 0x%02x", hdr.InnerType, tc.innerType)
			}
			if !bytes.Equal(hdr.PeerKey, peerKey) {
				t.Errorf("PeerKey round-trip mismatch")
			}
			if !bytes.Equal(hdr.Nonce, nonce) {
				t.Errorf("Nonce round-trip mismatch")
			}
			if !bytes.Equal(gotSealed, sealed) {
				t.Errorf("sealed round-trip mismatch")
			}
		})
	}
}

func TestParseAliasesFrameBuffer(t *testing.T) {
	// The header docstring promises PeerKey and Nonce alias the source
	// frame (zero-copy). This test pins that contract — a refactor to
	// allocate fresh copies would silently regress callers that rely on
	// the alias (notably the relay forward path, which calls
	// RewriteDestinationToSource on the same frame).
	frame, _, _, _ := envelopeFixture(t, InnerTypeJSONControl, EnvelopeMinSealedLen)
	hdr, sealed, err := ParseEnvelopeHeader(frame)
	if err != nil {
		t.Fatalf("ParseEnvelopeHeader: %v", err)
	}
	frame[1] = 0x99   // first byte of peer key
	frame[33] = 0x88  // first byte of nonce
	frame[57] = 0x77  // first byte of sealed
	if hdr.PeerKey[0] != 0x99 {
		t.Errorf("PeerKey[0] does not alias frame[1]: got 0x%02x, want 0x99", hdr.PeerKey[0])
	}
	if hdr.Nonce[0] != 0x88 {
		t.Errorf("Nonce[0] does not alias frame[33]: got 0x%02x, want 0x88", hdr.Nonce[0])
	}
	if sealed[0] != 0x77 {
		t.Errorf("sealed[0] does not alias frame[57]: got 0x%02x, want 0x77", sealed[0])
	}
}

func TestParseRejectsShortFrames(t *testing.T) {
	// Anything below the minimum frame length must fail-fast with a
	// clear error message — no panic on out-of-bounds slice.
	for _, n := range []int{0, 1, EnvelopeHeaderLen, EnvelopeHeaderLen + EnvelopeMinSealedLen - 1} {
		short := bytes.Repeat([]byte{0x00}, n)
		_, _, err := ParseEnvelopeHeader(short)
		if err == nil {
			t.Errorf("ParseEnvelopeHeader(%d bytes): expected error, got nil", n)
			continue
		}
		if !strings.Contains(err.Error(), "too short") {
			t.Errorf("ParseEnvelopeHeader(%d bytes) error = %q, want substring \"too short\"", n, err)
		}
	}
}

func TestMarshalRejectsBadSizes(t *testing.T) {
	goodPeerKey := bytes.Repeat([]byte{0x01}, EnvelopePeerKeyLen)
	goodNonce := bytes.Repeat([]byte{0x02}, EnvelopeNonceLen)
	goodSealed := bytes.Repeat([]byte{0x03}, EnvelopeMinSealedLen)

	cases := []struct {
		name    string
		peerKey []byte
		nonce   []byte
		sealed  []byte
		errSubs string
	}{
		{"short-peer-key", bytes.Repeat([]byte{0x01}, 31), goodNonce, goodSealed, "peer key"},
		{"long-peer-key", bytes.Repeat([]byte{0x01}, 33), goodNonce, goodSealed, "peer key"},
		{"empty-peer-key", []byte{}, goodNonce, goodSealed, "peer key"},
		{"short-nonce", goodPeerKey, bytes.Repeat([]byte{0x02}, 23), goodSealed, "nonce"},
		{"long-nonce", goodPeerKey, bytes.Repeat([]byte{0x02}, 25), goodSealed, "nonce"},
		{"short-sealed", goodPeerKey, goodNonce, bytes.Repeat([]byte{0x03}, 15), "sealed payload"},
		{"empty-sealed", goodPeerKey, goodNonce, []byte{}, "sealed payload"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MarshalEnvelope(InnerTypeJSONControl, tc.peerKey, tc.nonce, tc.sealed)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errSubs) {
				t.Errorf("error = %q, want substring %q", err, tc.errSubs)
			}
		})
	}
}

func TestRewriteDestinationToSource(t *testing.T) {
	// Spec: rewrite the 32-byte peer field in place; do not touch any
	// other byte. Verified by checksumming the frame's untouched regions.
	frame, _, origNonce, origSealed := envelopeFixture(t, InnerTypeJSONControl, 128)
	origInnerType := frame[0]
	newKey := bytes.Repeat([]byte{0x55}, EnvelopePeerKeyLen)

	if err := RewriteDestinationToSource(frame, newKey); err != nil {
		t.Fatalf("RewriteDestinationToSource: %v", err)
	}

	if frame[0] != origInnerType {
		t.Errorf("inner-type byte changed: got 0x%02x, want 0x%02x", frame[0], origInnerType)
	}
	if !bytes.Equal(frame[1:33], newKey) {
		t.Errorf("peer-key region not rewritten correctly")
	}
	if !bytes.Equal(frame[33:57], origNonce) {
		t.Errorf("nonce region was clobbered by rewrite")
	}
	if !bytes.Equal(frame[57:], origSealed) {
		t.Errorf("sealed region was clobbered by rewrite")
	}
}

func TestRewriteRejectsBadInputs(t *testing.T) {
	frame, _, _, _ := envelopeFixture(t, InnerTypeJSONControl, EnvelopeMinSealedLen)

	// Frame too short for header.
	if err := RewriteDestinationToSource(bytes.Repeat([]byte{0x00}, EnvelopeHeaderLen-1), bytes.Repeat([]byte{0x55}, EnvelopePeerKeyLen)); err == nil {
		t.Errorf("expected error on short frame, got nil")
	}

	// Source key wrong length.
	if err := RewriteDestinationToSource(frame, bytes.Repeat([]byte{0x55}, 31)); err == nil {
		t.Errorf("expected error on short source key, got nil")
	}
	if err := RewriteDestinationToSource(frame, bytes.Repeat([]byte{0x55}, 33)); err == nil {
		t.Errorf("expected error on long source key, got nil")
	}
}
