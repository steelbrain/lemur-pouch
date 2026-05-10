import { describe, expect, it } from 'vitest'

import { WireProtocolError, base64ToBytes, bytesToBase64 } from '../relay/wire'

import {
  TRANSFER_ID_LEN,
  TYPE_TRANSFER_ACCEPT,
  TYPE_TRANSFER_END,
  TYPE_TRANSFER_OFFER,
  TYPE_TRANSFER_REJECT,
  buildTransferAccept,
  buildTransferEnd,
  buildTransferOffer,
  buildTransferReject,
  parseTransferAccept,
  parseTransferControl,
  parseTransferEnd,
  parseTransferOffer,
  parseTransferReject,
} from './control'

const sampleTransferId = new Uint8Array(TRANSFER_ID_LEN).fill(0x77)
const sampleSha256 = new Uint8Array(32).fill(0x88)

describe('build + parse round-trips', () => {
  it('transfer-offer', () => {
    const json = buildTransferOffer(sampleTransferId, 'photo.jpg', 1_048_576, sampleSha256)
    const msg = parseTransferOffer(json)
    expect(msg.type).toBe(TYPE_TRANSFER_OFFER)
    expect(Array.from(msg.transferId)).toEqual(Array.from(sampleTransferId))
    expect(msg.filename).toBe('photo.jpg')
    expect(msg.size).toBe(1_048_576)
    expect(Array.from(msg.sha256)).toEqual(Array.from(sampleSha256))
  })

  it('transfer-accept', () => {
    const json = buildTransferAccept(sampleTransferId)
    const msg = parseTransferAccept(json)
    expect(msg.type).toBe(TYPE_TRANSFER_ACCEPT)
    expect(Array.from(msg.transferId)).toEqual(Array.from(sampleTransferId))
  })

  it('transfer-reject (no reason)', () => {
    const json = buildTransferReject(sampleTransferId)
    const obj = JSON.parse(json) as Record<string, unknown>
    expect(obj).not.toHaveProperty('reason') // omit field when undefined
    const msg = parseTransferReject(json)
    expect(msg.type).toBe(TYPE_TRANSFER_REJECT)
    expect(msg.reason).toBeUndefined()
  })

  it('transfer-reject (with reason)', () => {
    const json = buildTransferReject(sampleTransferId, 'out of disk space')
    const msg = parseTransferReject(json)
    expect(msg.reason).toBe('out of disk space')
  })

  it('transfer-end', () => {
    const json = buildTransferEnd(sampleTransferId)
    const msg = parseTransferEnd(json)
    expect(msg.type).toBe(TYPE_TRANSFER_END)
    expect(Array.from(msg.transferId)).toEqual(Array.from(sampleTransferId))
  })
})

describe('JSON field-name pinning', () => {
  // Pin the snake_case field names. Drift here silently breaks Go-TS
  // interop — same role as the FriendshipJSONFieldNames pinning test
  // in wire.test.ts.
  it('transfer-offer uses transfer_id, filename, size, sha256', () => {
    const json = buildTransferOffer(sampleTransferId, 'a', 1, sampleSha256)
    const obj = JSON.parse(json) as Record<string, unknown>
    expect(obj.type).toBe(TYPE_TRANSFER_OFFER)
    expect(typeof obj.transfer_id).toBe('string')
    expect(typeof obj.filename).toBe('string')
    expect(typeof obj.size).toBe('number')
    expect(typeof obj.sha256).toBe('string')
    expect(Object.keys(obj).sort()).toEqual([
      'filename',
      'sha256',
      'size',
      'transfer_id',
      'type',
    ])
  })

  it('transfer-accept uses transfer_id only', () => {
    const obj = JSON.parse(buildTransferAccept(sampleTransferId)) as Record<string, unknown>
    expect(Object.keys(obj).sort()).toEqual(['transfer_id', 'type'])
  })

  it('transfer-reject (with reason) uses transfer_id + reason', () => {
    const obj = JSON.parse(buildTransferReject(sampleTransferId, 'no')) as Record<string, unknown>
    expect(Object.keys(obj).sort()).toEqual(['reason', 'transfer_id', 'type'])
  })
})

describe('build helpers reject bad inputs', () => {
  it.each([
    ['short transfer_id', new Uint8Array(15), sampleSha256, /transfer_id must be 16 bytes/],
    ['long transfer_id', new Uint8Array(17), sampleSha256, /transfer_id must be 16 bytes/],
    ['short sha256', sampleTransferId, new Uint8Array(31), /sha256 must be 32 bytes/],
    ['long sha256', sampleTransferId, new Uint8Array(33), /sha256 must be 32 bytes/],
  ])('transfer-offer: %s', (_name, tid, sha, msg) => {
    expect(() => buildTransferOffer(tid, 'f', 1, sha)).toThrow(msg)
  })

  it.each<[string, number]>([
    ['negative', -1],
    ['fractional', 1.5],
    ['NaN', Number.NaN],
    ['+Infinity', Number.POSITIVE_INFINITY],
  ])('transfer-offer: rejects size = %s', (_name, size) => {
    expect(() => buildTransferOffer(sampleTransferId, 'f', size, sampleSha256)).toThrow(
      /size must be a non-negative integer/,
    )
  })

  it('transfer-accept rejects wrong-size transfer_id', () => {
    expect(() => buildTransferAccept(new Uint8Array(15))).toThrow(WireProtocolError)
  })

  it('transfer-reject rejects wrong-size transfer_id', () => {
    expect(() => buildTransferReject(new Uint8Array(15), 'no')).toThrow(WireProtocolError)
    expect(() => buildTransferReject(new Uint8Array(15))).toThrow(WireProtocolError)
  })

  it('transfer-end rejects wrong-size transfer_id', () => {
    expect(() => buildTransferEnd(new Uint8Array(15))).toThrow(WireProtocolError)
  })
})

describe('parsers reject bad inputs', () => {
  function offerJson(overrides: Record<string, unknown> = {}): string {
    return JSON.stringify({
      type: TYPE_TRANSFER_OFFER,
      transfer_id: bytesToBase64(sampleTransferId),
      filename: 'photo.jpg',
      size: 1024,
      sha256: bytesToBase64(sampleSha256),
      ...overrides,
    })
  }

  it('rejects malformed JSON', () => {
    expect(() => parseTransferOffer('{not json')).toThrow(WireProtocolError)
  })

  it('rejects wrong type discriminator', () => {
    expect(() => parseTransferOffer(offerJson({ type: TYPE_TRANSFER_END }))).toThrow(
      WireProtocolError,
    )
  })

  it('rejects truncated transfer_id (would-be 12 bytes)', () => {
    const shortId = new Uint8Array(12).fill(0x99)
    expect(() => parseTransferOffer(offerJson({ transfer_id: bytesToBase64(shortId) }))).toThrow(
      /transfer_id.*must decode to 16 bytes/,
    )
  })

  it('rejects truncated sha256', () => {
    const shortHash = new Uint8Array(20).fill(0xab)
    expect(() => parseTransferOffer(offerJson({ sha256: bytesToBase64(shortHash) }))).toThrow(
      /sha256.*must decode to 32 bytes/,
    )
  })

  it('rejects missing filename', () => {
    const obj = {
      type: TYPE_TRANSFER_OFFER,
      transfer_id: bytesToBase64(sampleTransferId),
      size: 1,
      sha256: bytesToBase64(sampleSha256),
    }
    expect(() => parseTransferOffer(JSON.stringify(obj))).toThrow(/filename/)
  })

  it('rejects negative size', () => {
    expect(() => parseTransferOffer(offerJson({ size: -1 }))).toThrow(/size/)
  })

  it('rejects non-integer size', () => {
    expect(() => parseTransferOffer(offerJson({ size: 1.5 }))).toThrow(/size/)
  })

  it('rejects null transfer_id (Go nil-slice JSON)', () => {
    expect(() => parseTransferOffer(offerJson({ transfer_id: null }))).toThrow(WireProtocolError)
  })

  it('parseTransferReject accepts missing reason as undefined', () => {
    const json = JSON.stringify({
      type: TYPE_TRANSFER_REJECT,
      transfer_id: bytesToBase64(sampleTransferId),
    })
    const msg = parseTransferReject(json)
    expect(msg.reason).toBeUndefined()
  })

  it('parseTransferReject treats reason: null as missing', () => {
    // Forward compat with a hypothetical Go optional pointer-to-string
    // field that marshals null when unset.
    const json = JSON.stringify({
      type: TYPE_TRANSFER_REJECT,
      transfer_id: bytesToBase64(sampleTransferId),
      reason: null,
    })
    const msg = parseTransferReject(json)
    expect(msg.reason).toBeUndefined()
  })
})

describe('parseTransferControl dispatcher', () => {
  it.each([TYPE_TRANSFER_OFFER, TYPE_TRANSFER_ACCEPT, TYPE_TRANSFER_REJECT, TYPE_TRANSFER_END])(
    'dispatches %s',
    (typ) => {
      // Build the right shape for each type so the per-type parser
      // succeeds inside the dispatcher.
      let json: string
      if (typ === TYPE_TRANSFER_OFFER) {
        json = buildTransferOffer(sampleTransferId, 'f', 1, sampleSha256)
      } else if (typ === TYPE_TRANSFER_ACCEPT) {
        json = buildTransferAccept(sampleTransferId)
      } else if (typ === TYPE_TRANSFER_REJECT) {
        json = buildTransferReject(sampleTransferId)
      } else {
        json = buildTransferEnd(sampleTransferId)
      }
      const msg = parseTransferControl(json)
      expect(msg?.type).toBe(typ)
    },
  )

  it('returns null for non-transfer types', () => {
    // Discovery / friendship messages, unknown types, malformed JSON
    // — all should return null (caller treats them as "not for me").
    expect(parseTransferControl(JSON.stringify({ type: 'peer-list', peers: [] }))).toBeNull()
    expect(parseTransferControl(JSON.stringify({ type: 'invite-from', from: 'AA==' }))).toBeNull()
    expect(parseTransferControl('{not json')).toBeNull()
    expect(parseTransferControl('')).toBeNull()
    expect(parseTransferControl('{}')).toBeNull()
  })

  it('throws when the type matches but the payload is malformed', () => {
    // Differentiates "not for me" (null) from "broken transfer message"
    // (throw) — same contract as parseDiscovery / parseFriendshipNotification.
    const broken = JSON.stringify({
      type: TYPE_TRANSFER_OFFER,
      transfer_id: bytesToBase64(sampleTransferId),
      // missing filename + size + sha256
    })
    expect(() => parseTransferControl(broken)).toThrow(WireProtocolError)
  })
})

describe('base64 encoding compatibility with Go side', () => {
  it('transfer-offer round-trips through base64ToBytes', () => {
    const json = buildTransferOffer(sampleTransferId, 'f', 1, sampleSha256)
    const obj = JSON.parse(json) as Record<string, unknown>
    expect(base64ToBytes(obj.transfer_id as string).length).toBe(TRANSFER_ID_LEN)
    expect(base64ToBytes(obj.sha256 as string).length).toBe(32)
  })
})
