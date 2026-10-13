import { describe, expect, it } from 'vitest'
import { settingsPageHeader } from '../settings-page-header'

describe('settingsPageHeader (#791)', () => {
  it('uses admin copy on system settings pages', () => {
    for (const path of ['/settings/platform', '/settings/lti-tools', '/settings/ai/models', '/settings/people']) {
      const h = settingsPageHeader(path)
      expect(h.title).toBe('System settings')
      expect(h.description).not.toMatch(/learning preferences/i)
    }
  })

  it('keeps the personal header on account pages', () => {
    expect(settingsPageHeader('/settings/account')).toEqual({
      title: 'Settings',
      description: 'Account and learning preferences.',
    })
  })
})
