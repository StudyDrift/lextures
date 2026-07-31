/**
 * CT.M4 AC-16 — shared bridge fixture suite. Same messages.json drives iOS/Android/web.
 */
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  BRIDGE_MAX_MESSAGE_BYTES,
  BRIDGE_MAX_MESSAGES_PER_SEC,
  BRIDGE_VERSION,
  BridgeRateLimiter,
  isBridgeFromTool,
  isBridgeToTool,
  measureMessageBytes,
} from '@lextures/tool-sdk'
import { opaqueParticipantId } from '../bridge'

const here = dirname(fileURLToPath(import.meta.url))
const fixturePath = resolve(
  here,
  '../../../../../../../mobile/fixtures/content-tools/bridge/messages.json',
)

type Fixture = {
  constants: {
    bridgeVersion: number
    maxMessageBytes: number
    maxMessagesPerSec: number
    minHeight: number
    maxHeight: number
  }
  validation: {
    cases: Array<{ name: string; msg: unknown; direction: string; accept: boolean }>
  }
  rateLimit: {
    maxPerSec: number
    cases: Array<{ name: string; timestampsMs: number[]; expectedAllow: boolean[] }>
  }
  sizeGuard: {
    cases: Array<{ name: string; approxBytes: number; reject: boolean }>
  }
  heightClamp: {
    cases: Array<{ input: number; expected: number }>
  }
  opaqueParticipantId: {
    cases: Array<{ instanceId: string; enrollmentHint: string | null; expected: string }>
  }
}

function loadFixture(): Fixture {
  return JSON.parse(readFileSync(fixturePath, 'utf8')) as Fixture
}

function clampHeight(h: number): number {
  if (!Number.isFinite(h)) return 80
  return Math.min(2000, Math.max(80, h))
}

describe('shared CT.M4 bridge fixtures (AC-16)', () => {
  const fixture = loadFixture()

  it('constants match protocol', () => {
    expect(BRIDGE_VERSION).toBe(fixture.constants.bridgeVersion)
    expect(BRIDGE_MAX_MESSAGE_BYTES).toBe(fixture.constants.maxMessageBytes)
    expect(BRIDGE_MAX_MESSAGES_PER_SEC).toBe(fixture.constants.maxMessagesPerSec)
  })

  it('validation accept/reject matches fixture', () => {
    for (const c of fixture.validation.cases) {
      const actual =
        c.direction === 'fromTool' ? isBridgeFromTool(c.msg) : isBridgeToTool(c.msg)
      expect(actual, c.name).toBe(c.accept)
    }
  })

  it('rate limiter matches fixture', () => {
    for (const c of fixture.rateLimit.cases) {
      const limiter = new BridgeRateLimiter(fixture.rateLimit.maxPerSec)
      const actual = c.timestampsMs.map((t) => limiter.allow(t))
      expect(actual, c.name).toEqual(c.expectedAllow)
    }
  })

  it('size guard rejects oversized messages', () => {
    for (const c of fixture.sizeGuard.cases) {
      const payload = 'x'.repeat(c.approxBytes)
      const msg = { t: 'announce', v: 1, message: payload }
      const oversized = measureMessageBytes(msg) > BRIDGE_MAX_MESSAGE_BYTES
      expect(oversized, c.name).toBe(c.reject)
    }
  })

  it('height clamp matches fixture', () => {
    for (const c of fixture.heightClamp.cases) {
      expect(clampHeight(c.input)).toBe(c.expected)
    }
  })

  it('opaqueParticipantId matches fixture', () => {
    for (const c of fixture.opaqueParticipantId.cases) {
      expect(opaqueParticipantId(c.instanceId, c.enrollmentHint ?? undefined)).toBe(c.expected)
    }
  })
})
