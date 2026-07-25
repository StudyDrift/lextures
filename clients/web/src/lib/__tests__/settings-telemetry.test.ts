import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  __resetSettingsTelemetryForTests,
  emitSettingsTelemetry,
  hashSearchQuerySync,
  normaliseSearchQuery,
  onSettingsTelemetry,
  scheduleSettingsControlChanged,
  scheduleSettingsSearchTelemetry,
  SETTINGS_CONTROL_TELEMETRY_DEBOUNCE_MS,
  SETTINGS_SEARCH_TELEMETRY_DEBOUNCE_MS,
  validateSettingsTelemetryEvent,
} from '../settings-telemetry'

describe('settings-telemetry', () => {
  beforeEach(() => {
    __resetSettingsTelemetryForTests()
    vi.useFakeTimers()
  })

  afterEach(() => {
    __resetSettingsTelemetryForTests()
    vi.useRealTimers()
  })

  it('validateSettingsTelemetryEvent rejects unknown fields and raw query', () => {
    expect(
      validateSettingsTelemetryEvent('settings_pin_added', {
        surface: 'quiz',
        role: 'instructor',
        query: 'Smith essay',
      }),
    ).toBeNull()

    expect(
      validateSettingsTelemetryEvent('settings_pin_added', {
        surface: 'quiz',
        role: 'instructor',
        setting_id: 'quiz.scheduling.due-date',
        course_id: 'c1',
      }),
    ).toMatchObject({
      event: 'settings_pin_added',
      props: {
        surface: 'quiz',
        role: 'instructor',
        setting_id: 'quiz.scheduling.due-date',
      },
    })
    // course_id must not appear on the cleaned props
    const ok = validateSettingsTelemetryEvent('settings_pin_added', {
      surface: 'quiz',
      role: 'instructor',
      course_id: 'c1',
    })
    expect(ok?.props).not.toHaveProperty('course_id')
  })

  it('validate rejects unknown event names', () => {
    expect(
      validateSettingsTelemetryEvent('settings_pin_hacked', {
        surface: 'quiz',
        role: 'other',
      }),
    ).toBeNull()
  })

  it('normaliseSearchQuery is Unicode-aware (NFKC + lowercase)', () => {
    expect(normaliseSearchQuery('  Café  LOCKDOWN  ')).toBe('café lockdown')
    expect(normaliseSearchQuery('ＡＢＣ')).toBe('abc') // fullwidth → ascii via NFKC
  })

  it('hashSearchQuerySync is stable and does not embed the raw string', () => {
    const a = hashSearchQuerySync('Smith essay')
    const b = hashSearchQuerySync('Smith essay')
    const c = hashSearchQuerySync('smith essay')
    expect(a).toBe(b)
    expect(a).toBe(c) // normalisation
    expect(a).not.toContain('Smith')
    expect(a).not.toContain('essay')
    expect(a.length).toBeGreaterThanOrEqual(16)
  })

  it('emitSettingsTelemetry delivers only validated events to listeners', () => {
    const seen: string[] = []
    onSettingsTelemetry((e) => {
      seen.push(e.event)
      expect(e.props).not.toHaveProperty('query')
    })
    emitSettingsTelemetry('settings_pin_added', {
      surface: 'quiz',
      role: 'instructor',
      setting_id: 'quiz.scheduling.due-date',
    })
    expect(seen).toEqual(['settings_pin_added'])
  })

  it('search debounce emits at most one event per 1s idle window (FR-11 / AC-8)', async () => {
    const events: string[] = []
    onSettingsTelemetry((e) => events.push(e.event))

    for (let i = 0; i < 12; i++) {
      scheduleSettingsSearchTelemetry({
        surface: 'quiz',
        role: 'instructor',
        query: 'Smith essay' + 'x'.repeat(i % 3),
        resultCount: 0,
        enabled: true,
      })
      await vi.advanceTimersByTimeAsync(100)
    }
    // Still within typing burst — no flush yet if last keystroke < 1s ago.
    // Advance remaining debounce.
    await vi.advanceTimersByTimeAsync(SETTINGS_SEARCH_TELEMETRY_DEBOUNCE_MS)
    // performed + zero_results for the last query
    expect(events.filter((e) => e === 'settings_search_performed')).toHaveLength(1)
    expect(events.filter((e) => e === 'settings_search_zero_results')).toHaveLength(1)
    // Payload must not include raw query
    const listenerPayloads: unknown[] = []
    __resetSettingsTelemetryForTests()
    onSettingsTelemetry((e) => listenerPayloads.push(e))
    scheduleSettingsSearchTelemetry({
      surface: 'quiz',
      role: 'other',
      query: 'Smith essay',
      resultCount: 2,
      enabled: true,
    })
    await vi.advanceTimersByTimeAsync(SETTINGS_SEARCH_TELEMETRY_DEBOUNCE_MS)
    expect(JSON.stringify(listenerPayloads)).not.toContain('Smith')
    expect(listenerPayloads[0]).toMatchObject({
      event: 'settings_search_performed',
      props: { surface: 'quiz', result_count: 2 },
    })
    expect((listenerPayloads[0] as { props: { query_hash?: string } }).props.query_hash).toBeTruthy()
  })

  it('control-changed debounce is per control per 2s', async () => {
    const events: Array<{ event: string; id?: string }> = []
    onSettingsTelemetry((e) =>
      events.push({ event: e.event, id: e.props.setting_id }),
    )
    for (let i = 0; i < 5; i++) {
      scheduleSettingsControlChanged({
        surface: 'quiz',
        role: 'instructor',
        settingId: 'quiz.scheduling.due-date',
        enabled: true,
      })
      await vi.advanceTimersByTimeAsync(200)
    }
    await vi.advanceTimersByTimeAsync(SETTINGS_CONTROL_TELEMETRY_DEBOUNCE_MS)
    expect(events.filter((e) => e.event === 'settings_control_changed')).toHaveLength(1)
  })

  it('emits nothing when enabled is false (AC-9)', async () => {
    const events: string[] = []
    onSettingsTelemetry((e) => events.push(e.event))
    scheduleSettingsSearchTelemetry({
      surface: 'quiz',
      role: 'other',
      query: 'lockdown',
      resultCount: 1,
      enabled: false,
    })
    await vi.advanceTimersByTimeAsync(SETTINGS_SEARCH_TELEMETRY_DEBOUNCE_MS)
    scheduleSettingsControlChanged({
      surface: 'quiz',
      role: 'other',
      settingId: 'quiz.presentation.lockdown-mode',
      enabled: false,
    })
    await vi.advanceTimersByTimeAsync(SETTINGS_CONTROL_TELEMETRY_DEBOUNCE_MS)
    expect(events).toEqual([])
  })
})
