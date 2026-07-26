import { describe, expect, it } from 'vitest'
import {
  parseFencePayload,
  serializeFence,
  serializeLexToolFenceBlock,
} from '../lex-tool-fence'

describe('lex-tool-fence', () => {
  const sample = {
    instanceId: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    toolId: 'noop_probe',
  }

  it('serializes with stable key order', () => {
    expect(serializeFence(sample)).toBe(
      '{"instanceId":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","toolId":"noop_probe","v":1}',
    )
  })

  it('round-trips parse → serialize', () => {
    const json = serializeFence(sample)
    const parsed = parseFencePayload(json)
    expect(parsed).toEqual({ ...sample, v: 1 })
    expect(serializeFence(parsed!)).toBe(json)
  })

  it('builds a full fence block', () => {
    expect(serializeLexToolFenceBlock(sample)).toBe(
      '```lex-tool\n{"instanceId":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","toolId":"noop_probe","v":1}\n```',
    )
  })

  it('rejects invalid payloads', () => {
    expect(parseFencePayload('')).toBeNull()
    expect(parseFencePayload('not-json')).toBeNull()
    expect(parseFencePayload('{"toolId":"noop_probe","v":1}')).toBeNull()
    expect(parseFencePayload('{"instanceId":"x","toolId":"noop_probe","v":2}')).toBeNull()
  })

  it('accepts v as string "1"', () => {
    expect(
      parseFencePayload(
        '{"instanceId":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","toolId":"noop_probe","v":"1"}',
      ),
    ).toEqual({ ...sample, v: 1 })
  })
})
