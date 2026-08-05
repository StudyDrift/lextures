import { describe, expect, it } from 'vitest'
import {
  checklistResponseSchema,
  checklistSummarySchema,
  isDoneStatus,
  isOutstandingStatus,
  normalizeChecklistStatus,
  visibleChecklistItems,
} from '../course-checklist-api-schemas'

describe('course-checklist-api-schemas', () => {
  it('normalizes unknown status values to unknown', () => {
    expect(normalizeChecklistStatus('done')).toBe('done')
    expect(normalizeChecklistStatus('weird')).toBe('unknown')
  })

  it('detects outstanding statuses', () => {
    expect(isOutstandingStatus('todo')).toBe(true)
    expect(isOutstandingStatus('in_progress')).toBe(true)
    expect(isOutstandingStatus('unknown')).toBe(true)
    expect(isOutstandingStatus('done')).toBe(false)
  })

  it('detects done status', () => {
    expect(isDoneStatus('done')).toBe(true)
    expect(isDoneStatus('todo')).toBe(false)
    expect(isDoneStatus('not_applicable')).toBe(false)
  })

  it('filters visible items when completed are hidden', () => {
    const items = [
      { id: 'a', status: 'done' },
      { id: 'b', status: 'todo' },
      { id: 'c', status: 'in_progress' },
      { id: 'd', status: 'not_applicable' },
    ]
    expect(visibleChecklistItems(items, false).map((i) => i.id)).toEqual(['b', 'c', 'd'])
    expect(visibleChecklistItems(items, true).map((i) => i.id)).toEqual(['a', 'b', 'c', 'd'])
  })

  it('parses a summary payload', () => {
    const parsed = checklistSummarySchema.parse({
      outstandingEssential: 2,
      outstandingTotal: 5,
      done: 10,
      total: 15,
      dismissed: 1,
      computedAt: '2026-08-04T12:00:00Z',
      stale: false,
    })
    expect(parsed.outstandingEssential).toBe(2)
  })

  it('parses a checklist response with evidence and dismissal', () => {
    const parsed = checklistResponseSchema.parse({
      courseCode: 'C1',
      engineVersion: 1,
      catalogVersion: 'v1',
      computedAt: '2026-08-04T12:00:00Z',
      stale: false,
      evidenceTruncated: false,
      summary: {
        outstandingEssential: 1,
        outstandingTotal: 1,
        done: 0,
        total: 1,
        dismissed: 0,
        computedAt: '2026-08-04T12:00:00Z',
        stale: false,
      },
      categories: [
        {
          id: 'foundations',
          titleKey: 'cat.foundations',
          title: 'Foundations',
          items: [
            {
              id: 'orientation.welcome-message',
              titleKey: 'item.welcome',
              title: 'Post a welcome',
              whyKey: 'why.welcome',
              why: 'Students need orientation',
              tier: 'essential',
              status: 'todo',
              detail: 'Missing announcement',
              progress: null,
              sources: ['QM 1.2'],
              helpRef: null,
              target: { route: '/courses/C1/feed' },
              evidence: {
                columns: ['Item', 'Type'],
                rows: [
                  {
                    label: 'Essay 1',
                    sublabel: 'Assign.',
                    status: 'unmapped',
                    target: { route: '/courses/C1/modules/assignment/a1' },
                  },
                ],
                truncatedAt: null,
              },
              dismissal: null,
            },
          ],
        },
      ],
      dismissed: [],
    })
    expect(parsed.categories[0]?.items[0]?.evidence?.rows).toHaveLength(1)
  })
})
