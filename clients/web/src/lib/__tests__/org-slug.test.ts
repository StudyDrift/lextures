import { describe, expect, it } from 'vitest'
import { orgLoginPath, suggestOrgSlugFromName, validateOrgSlug } from '../org-slug'

describe('org-slug', () => {
  it('suggests a slug from a display name', () => {
    expect(suggestOrgSlugFromName("Chase's Org")).toBe('chase-s-org')
    expect(suggestOrgSlugFromName('Riverdale USD')).toBe('riverdale-usd')
  })

  it('validates acceptable slugs', () => {
    expect(validateOrgSlug('chase')).toBeNull()
    expect(validateOrgSlug('riverdale-usd')).toBeNull()
  })

  it('rejects invalid slugs', () => {
    expect(validateOrgSlug('')).not.toBeNull()
    expect(validateOrgSlug('bad slug')).not.toBeNull()
    expect(validateOrgSlug('default')).not.toBeNull()
  })

  it('builds org login paths', () => {
    expect(orgLoginPath('chase')).toBe('/login/chase')
  })
})