import { describe, expect, it } from 'vitest'
import { SITE_BUILD_STALE_MS, siteBuildSummary } from '../site-build-summary'

const now = Date.parse('2026-10-13T18:00:00Z')

describe('siteBuildSummary', () => {
  it('flags a pending update requested long ago as stuck instead of "pending"', () => {
    const s = siteBuildSummary(
      { id: 'b1', status: 'pending', createdAt: '2026-08-13T12:00:00Z' },
      now,
    )
    expect(s.tone).toBe('warning')
    expect(s.title).toBe('Site update is stuck')
    expect(s.title.toLowerCase()).not.toContain('pending')
  })

  it('shows an in-progress update for a recent request', () => {
    const s = siteBuildSummary(
      { id: 'b2', status: 'running', createdAt: new Date(now - SITE_BUILD_STALE_MS / 4).toISOString() },
      now,
    )
    expect(s.tone).toBe('info')
    expect(s.title).toBe('Updating the public site')
  })

  it('describes success and failure in plain language', () => {
    expect(
      siteBuildSummary({ id: 'b3', status: 'succeeded', createdAt: '2026-10-13T17:00:00Z', completedAt: '2026-10-13T17:05:00Z' }, now).title,
    ).toBe('Public site is up to date')
    const failed = siteBuildSummary({ id: 'b4', status: 'timed_out', createdAt: '2026-10-13T17:00:00Z' }, now)
    expect(failed.tone).toBe('danger')
    expect(failed.title).not.toContain('timed_out')
  })
})
