import { describe, expect, it } from 'vitest'
import { isHomeschoolOrK12Audience } from '../family-audience'

describe('isHomeschoolOrK12Audience', () => {
  it('treats parent accounts as a family audience even on a higher-ed course', () => {
    expect(
      isHomeschoolOrK12Audience({
        accountType: 'parent',
        orgKnown: true,
        orgId: 'org-1',
        orgType: 'higher-ed',
      }),
    ).toBe(true)
  })

  it('treats a loaded course with no organization as homeschool', () => {
    expect(
      isHomeschoolOrK12Audience({
        accountType: 'standard',
        orgKnown: true,
        orgId: null,
      }),
    ).toBe(true)
  })

  it('treats grade levels and k-12 orgs as K–12', () => {
    expect(
      isHomeschoolOrK12Audience({
        accountType: 'standard',
        orgKnown: true,
        orgId: 'org-1',
        gradeLevels: ['6-8'],
      }),
    ).toBe(true)
    expect(
      isHomeschoolOrK12Audience({
        accountType: 'standard',
        orgKnown: true,
        orgId: 'org-1',
        orgType: 'k-12',
      }),
    ).toBe(true)
  })

  it('keeps campus copy for higher-ed orgs and until the course is known', () => {
    expect(
      isHomeschoolOrK12Audience({
        accountType: 'standard',
        orgKnown: true,
        orgId: 'org-1',
        orgType: 'higher-ed',
      }),
    ).toBe(false)
    expect(
      isHomeschoolOrK12Audience({
        accountType: 'standard',
        orgKnown: false,
        orgId: null,
      }),
    ).toBe(false)
  })
})
