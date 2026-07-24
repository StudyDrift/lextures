import { describe, expect, it } from 'vitest'
import { dedupeOrgTermsForPicker, type OrgTerm } from '../courses-api'

function term(partial: Partial<OrgTerm> & { id: string; name: string }): OrgTerm {
  return {
    orgId: 'org-1',
    termType: 'semester',
    startDate: '2025-01-01',
    endDate: '2025-05-01',
    status: 'completed',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...partial,
  }
}

describe('dedupeOrgTermsForPicker', () => {
  it('keeps the first occurrence of each case-insensitive name', () => {
    const rows = [
      term({ id: 'a', name: 'Summer 2025', startDate: '2025-05-01' }),
      term({ id: 'b', name: 'summer 2025', startDate: '2025-05-01' }),
      term({ id: 'c', name: 'Spring 2025', startDate: '2025-01-01' }),
      term({ id: 'd', name: 'Spring 2025', startDate: '2025-01-01' }),
    ]
    expect(dedupeOrgTermsForPicker(rows).map((t) => t.id)).toEqual(['a', 'c'])
  })

  it('skips blank names', () => {
    expect(dedupeOrgTermsForPicker([term({ id: 'x', name: '   ' })])).toEqual([])
  })
})
