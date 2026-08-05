import { describe, expect, it } from 'vitest'
import {
  CHECKLIST_ACCOMMODATION_ITEM_IDS,
  emitChecklistTelemetry,
  onChecklistTelemetry,
  validateChecklistTelemetryEvent,
} from '../checklist-telemetry'

describe('checklist telemetry (CC.10)', () => {
  it('accepts allowed events and strips unknown keys', () => {
    const v = validateChecklistTelemetryEvent('checklist_item_dismissed', {
      itemId: 'syllabus.late-policy',
      reason: 'disagree',
      unknown: 'x',
    })
    expect(v).not.toBeNull()
    expect(v!.event).toBe('checklist_item_dismissed')
    expect(v!.props).toEqual({ itemId: 'syllabus.late-policy', reason: 'disagree' })
  })

  it('rejects accommodation item ids (AC-8)', () => {
    for (const id of CHECKLIST_ACCOMMODATION_ITEM_IDS) {
      const v = validateChecklistTelemetryEvent('checklist_item_expanded', { itemId: id })
      expect(v).toBeNull()
    }
  })

  it('rejects PII / evidence smuggling keys', () => {
    expect(
      validateChecklistTelemetryEvent('checklist_viewed', { courseCode: 'BIO101' }),
    ).toBeNull()
    expect(
      validateChecklistTelemetryEvent('checklist_item_dismissed', {
        itemId: 'x',
        note: 'secret',
      }),
    ).toBeNull()
  })

  it('emits to listeners when valid', () => {
    const seen: string[] = []
    const unsub = onChecklistTelemetry((e) => seen.push(e.event))
    emitChecklistTelemetry('checklist_viewed')
    emitChecklistTelemetry('checklist_item_dismissed', {
      itemId: 'accommodations.honored',
      reason: 'later',
    })
    unsub()
    expect(seen).toEqual(['checklist_viewed'])
  })
})
