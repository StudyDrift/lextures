import { describe, expect, it } from 'vitest'
import type { ChecklistItem, ChecklistResponse } from '../course-checklist-api-schemas'
import {
  checklistHasPendingLinkCheck,
  isExternalLinkCheckPending,
  replaceChecklistItem,
} from '../checklist-link-health'

function item(partial: Partial<ChecklistItem> & Pick<ChecklistItem, 'id'>): ChecklistItem {
  return {
    titleKey: 't',
    title: 'Check external links',
    whyKey: 'w',
    why: 'why',
    tier: 'recommended',
    status: 'unknown',
    sources: [],
    ...partial,
  }
}

describe('checklist link health pending', () => {
  it('treats the checking detail as in flight', () => {
    expect(isExternalLinkCheckPending(item({ id: 'links.external-health', detail: 'Checking links…' }))).toBe(
      true,
    )
    expect(
      isExternalLinkCheckPending(
        item({ id: 'links.external-health', detail: 'Outbound link checking is turned off.' }),
      ),
    ).toBe(false)
    expect(isExternalLinkCheckPending(item({ id: 'other', detail: 'Checking links…' }))).toBe(false)
  })

  it('finds a pending row in the checklist', () => {
    const pending = {
      categories: [{ items: [item({ id: 'links.external-health', detail: 'Checking links…' })] }],
    }
    expect(checklistHasPendingLinkCheck(pending)).toBe(true)
    expect(checklistHasPendingLinkCheck({ categories: [] })).toBe(false)
    expect(checklistHasPendingLinkCheck(null)).toBe(false)
  })

  it('replaces only the matching item', () => {
    const data = {
      categories: [
        {
          id: 'launch',
          titleKey: 'k',
          title: 'Launch',
          items: [
            item({ id: 'links.external-health', detail: 'Checking links…' }),
            item({ id: 'other', status: 'todo', detail: 'Fix this' }),
          ],
        },
      ],
    } as ChecklistResponse
    const next = replaceChecklistItem(
      data,
      item({ id: 'links.external-health', status: 'done', detail: 'External links resolved on the last check.' }),
    )
    expect(next.categories[0]?.items[0]?.status).toBe('done')
    expect(next.categories[0]?.items[1]?.id).toBe('other')
  })
})
