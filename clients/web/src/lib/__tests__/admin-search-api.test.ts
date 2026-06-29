import { describe, expect, it } from 'vitest'
import { adminSearchResultsPath } from '../admin-search-api'

describe('adminSearchResultsPath', () => {
  it('builds path with query and type', () => {
    const path = adminSearchResultsPath('johnson', 'users')
    expect(path).toContain('q=johnson')
    expect(path).toContain('type=users')
    expect(path.startsWith('/org-admin/search?')).toBe(true)
  })

  it('includes orgId when provided', () => {
    const path = adminSearchResultsPath('bio', 'courses', 'org-123')
    expect(path).toContain('orgId=org-123')
  })
})
