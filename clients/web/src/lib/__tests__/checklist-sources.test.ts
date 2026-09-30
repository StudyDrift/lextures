import { beforeEach, describe, expect, it } from 'vitest'
import { setCachedAccountTypeFromUser } from '../auth'
import {
  filterChecklistSourcesForAudience,
  isCampusRubricSource,
} from '../checklist-sources'

describe('checklist-sources', () => {
  beforeEach(() => {
    setCachedAccountTypeFromUser(undefined)
  })

  it('detects QM/OSCQR campus rubric codes', () => {
    expect(isCampusRubricSource('QM 1.1')).toBe(true)
    expect(isCampusRubricSource('OSCQR 2')).toBe(true)
    expect(isCampusRubricSource('WCAG 1.4.3')).toBe(false)
    expect(isCampusRubricSource('NSQ A')).toBe(false)
  })

  it('hides campus codes for parent accounts', () => {
    setCachedAccountTypeFromUser({ accountType: 'parent' })
    expect(
      filterChecklistSourcesForAudience(['QM 1.1', 'OSCQR 2', 'WCAG 1.4.3', 'NSQ A']),
    ).toEqual(['WCAG 1.4.3', 'NSQ A'])
  })

  it('keeps campus codes for standard accounts', () => {
    setCachedAccountTypeFromUser({ accountType: 'standard' })
    expect(
      filterChecklistSourcesForAudience(['QM 1.1', 'OSCQR 2', 'WCAG 1.4.3']),
    ).toEqual(['QM 1.1', 'OSCQR 2', 'WCAG 1.4.3'])
  })
})
