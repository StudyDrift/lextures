import { describe, expect, it } from 'vitest'
import {
  clampSeekTime,
  findDueCheckpoint,
  mergeLocalSegments,
} from '../checkpoint-engine'
import { parseTranscriptMarkdown, prefersReducedMotion } from '../types'
import type { Checkpoint } from '../types'

const cps: Checkpoint[] = [
  {
    id: 'c1',
    atSec: 15,
    required: true,
    question: { type: 'single', prompt: 'Q1', options: [{ id: 'a', text: 'A' }] },
  },
  {
    id: 'c2',
    atSec: 45,
    required: true,
    question: { type: 'single', prompt: 'Q2', options: [{ id: 'a', text: 'A' }] },
  },
]

describe('media_checkpoints checkpoint-engine', () => {
  it('fires due checkpoint near timestamp', () => {
    const due = findDueCheckpoint(cps, {}, 15.1, new Set())
    expect(due?.id).toBe('c1')
  })

  it('skips already answered checkpoints', () => {
    const due = findDueCheckpoint(
      cps,
      { c1: { done: true, attempts: [{ value: 'a', correct: true, at: 't' }] } },
      45,
      new Set(),
    )
    expect(due?.id).toBe('c2')
  })

  it('clamps seek past unanswered required checkpoint', () => {
    const { time, clamped } = clampSeekTime(true, cps, {}, 100)
    expect(clamped).toBe(true)
    expect(time).toBe(15)
  })

  it('merges coarse watch segments', () => {
    const segs = mergeLocalSegments([], 0.2, 3.1)
    expect(segs).toEqual([[0, 5]])
    const more = mergeLocalSegments(segs, 4, 12)
    expect(more).toEqual([[0, 15]])
  })

  it('parses transcript markdown timestamps', () => {
    const lines = parseTranscriptMarkdown('0:00 Intro\n1:02 Concept\nplain')
    expect(lines[0]).toMatchObject({ atSec: 0, text: 'Intro' })
    expect(lines[1]).toMatchObject({ atSec: 62, text: 'Concept' })
    expect(lines[2]?.text).toBe('plain')
  })

  it('exposes prefersReducedMotion helper', () => {
    expect(typeof prefersReducedMotion()).toBe('boolean')
  })
})
