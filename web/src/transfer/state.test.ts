import { describe, expect, it } from 'vitest'

import { sha256 } from '@noble/hashes/sha2.js'

import { type InboundTransfer, finalizeBlobUrl, formatBytes, tryAssemble } from './state'

function makeInbound(
  overrides: Partial<InboundTransfer> = {},
): InboundTransfer {
  return {
    transferIdHex: 'aa'.repeat(16),
    peerHex: 'bb'.repeat(32),
    filename: 'test.bin',
    totalBytes: 0,
    expectedSha256: sha256(new Uint8Array(0)),
    receivedBytes: 0,
    chunks: new Map(),
    lastSeq: null,
    status: 'streaming',
    ...overrides,
  }
}

// inboundFromChunks builds an InboundTransfer whose totalBytes and
// expectedSha256 are derived from the chunks, so the hash/size checks
// in tryAssemble are exercised on data the test actually controls.
function inboundFromChunks(
  ordered: Uint8Array[],
  overrides: Partial<InboundTransfer> = {},
): InboundTransfer {
  const totalLen = ordered.reduce((s, c) => s + c.length, 0)
  const flat = new Uint8Array(totalLen)
  let off = 0
  for (const c of ordered) {
    flat.set(c, off)
    off += c.length
  }
  return makeInbound({
    totalBytes: totalLen,
    expectedSha256: sha256(flat),
    receivedBytes: totalLen,
    ...overrides,
  })
}

describe('tryAssemble', () => {
  it('returns input unchanged when lastSeq is null', () => {
    const t = makeInbound({
      chunks: new Map([
        [0, new Uint8Array([0x10])],
        [1, new Uint8Array([0x20])],
      ]),
      receivedBytes: 2,
    })
    const out = tryAssemble(t)
    expect(out).toBe(t)
    expect(out.status).toBe('streaming')
  })

  it('returns input unchanged when there is a gap below lastSeq', () => {
    const t = makeInbound({
      chunks: new Map([
        [0, new Uint8Array([0x10])],
        // missing seq 1
        [2, new Uint8Array([0x30])],
      ]),
      lastSeq: 2,
      receivedBytes: 2,
    })
    const out = tryAssemble(t)
    expect(out).toBe(t)
    expect(out.status).toBe('streaming')
  })

  it('assembles when all chunks 0..lastSeq are present', () => {
    const c0 = new Uint8Array([0x01, 0x02])
    const c1 = new Uint8Array([0x03, 0x04, 0x05])
    const c2 = new Uint8Array([0x06])
    const t = inboundFromChunks([c0, c1, c2], {
      chunks: new Map([[0, c0], [1, c1], [2, c2]]),
      lastSeq: 2,
    })
    const out = tryAssemble(t)
    expect(out.status).toBe('done')
    expect(out.blobUrl).toBeUndefined() // tryAssemble is pure; URL minted post-commit
    expect(out.assembledBytes).toBeDefined()
    expect(out.chunks.size).toBe(0) // memory freed
    expect(Array.from(out.assembledBytes!)).toEqual([0x01, 0x02, 0x03, 0x04, 0x05, 0x06])
  })

  it('respects sequence order during assembly (out-of-order arrivals)', () => {
    // Insert chunks in arbitrary order; tryAssemble must walk 0..lastSeq
    // to concat them in the correct order.
    const c0 = new Uint8Array([0x01, 0x02])
    const c1 = new Uint8Array([0x03, 0x04, 0x05])
    const c2 = new Uint8Array([0x06])
    const t = inboundFromChunks([c0, c1, c2], {
      chunks: new Map([[2, c2], [0, c0], [1, c1]]),
      lastSeq: 2,
    })
    const out = tryAssemble(t)
    expect(out.status).toBe('done')
    expect(Array.from(out.assembledBytes!)).toEqual([0x01, 0x02, 0x03, 0x04, 0x05, 0x06])
  })

  it('handles a single-chunk transfer (lastSeq=0)', () => {
    const c0 = new Uint8Array([0x99])
    const t = inboundFromChunks([c0], {
      chunks: new Map([[0, c0]]),
      lastSeq: 0,
    })
    const out = tryAssemble(t)
    expect(out.status).toBe('done')
    expect(Array.from(out.assembledBytes!)).toEqual([0x99])
  })

  it('handles a zero-data last-chunk-only transfer', () => {
    // Edge case: a sender shipping a 0-byte file with a single empty
    // chunk that just sets the last-flag.
    const t = inboundFromChunks([new Uint8Array(0)], {
      chunks: new Map([[0, new Uint8Array(0)]]),
      lastSeq: 0,
    })
    const out = tryAssemble(t)
    expect(out.status).toBe('done')
    expect(out.assembledBytes!.length).toBe(0)
  })

  it('aborts when assembled length differs from advertised totalBytes', () => {
    const c0 = new Uint8Array([0x10, 0x11, 0x12])
    const t = inboundFromChunks([c0], {
      chunks: new Map([[0, c0]]),
      lastSeq: 0,
      totalBytes: 5, // sender advertised more than actually arrived
    })
    const out = tryAssemble(t)
    expect(out.status).toBe('aborted')
    expect(out.assembledBytes).toBeUndefined()
    expect(out.chunks.size).toBe(0)
  })

  it('aborts when assembled bytes hash to a different SHA-256 than advertised', () => {
    const c0 = new Uint8Array([0xAA, 0xBB])
    const t = inboundFromChunks([c0], {
      chunks: new Map([[0, c0]]),
      lastSeq: 0,
      // Override expectedSha256 with a wrong digest.
      expectedSha256: new Uint8Array(32).fill(0xFF),
    })
    const out = tryAssemble(t)
    expect(out.status).toBe('aborted')
    expect(out.assembledBytes).toBeUndefined()
  })
})

describe('finalizeBlobUrl', () => {
  it('materializes a blob URL from assembledBytes and clears the bytes', async () => {
    const bytes = new Uint8Array([0x01, 0x02, 0x03, 0x04, 0x05, 0x06])
    const t: InboundTransfer = {
      transferIdHex: 'aa'.repeat(16),
      peerHex: 'bb'.repeat(32),
      filename: 'test.bin',
      totalBytes: 6,
      expectedSha256: sha256(bytes),
      receivedBytes: 6,
      chunks: new Map(),
      lastSeq: 5,
      status: 'done',
      assembledBytes: bytes,
    }
    const out = finalizeBlobUrl(t)
    expect(out.blobUrl).toMatch(/^blob:/)
    expect(out.assembledBytes).toBeUndefined()
    // Read the Blob back to verify byte content.
    const resp = await fetch(out.blobUrl!)
    const buf = new Uint8Array(await resp.arrayBuffer())
    expect(Array.from(buf)).toEqual([0x01, 0x02, 0x03, 0x04, 0x05, 0x06])
    URL.revokeObjectURL(out.blobUrl!)
  })

  it('returns input unchanged when assembledBytes is missing', () => {
    const t: InboundTransfer = {
      transferIdHex: 'aa'.repeat(16),
      peerHex: 'bb'.repeat(32),
      filename: 'test.bin',
      totalBytes: 0,
      expectedSha256: sha256(new Uint8Array(0)),
      receivedBytes: 0,
      chunks: new Map(),
      lastSeq: null,
      status: 'streaming',
    }
    const out = finalizeBlobUrl(t)
    expect(out).toBe(t)
  })
})

describe('formatBytes', () => {
  it.each([
    [0, '0 B'],
    [1, '1 B'],
    [1023, '1023 B'],
    [1024, '1.0 KB'],
    [1500, '1.5 KB'],
    [1024 * 1024, '1.0 MB'],
    [1024 * 1024 * 1024, '1.0 GB'],
    [1024 * 1024 * 1024 * 1024, '1.0 TB'],
    [1.5 * 1024 * 1024 * 1024 * 1024, '1.5 TB'],
  ])('%i -> %s', (n, want) => {
    expect(formatBytes(n)).toBe(want)
  })
})
