import { describe, expect, it } from 'vitest'
import { IcuFormatPlugin, normalizeI18nextPlaceholders } from '../icu-format-plugin'

describe('normalizeI18nextPlaceholders', () => {
  it('converts double-brace placeholders to ICU single braces', () => {
    expect(normalizeI18nextPlaceholders('{{count}} attempts left')).toBe('{count} attempts left')
    expect(normalizeI18nextPlaceholders('About {{min}}–{{max}} words')).toBe(
      'About {min}–{max} words',
    )
  })

  it('leaves single-brace ICU messages unchanged', () => {
    expect(normalizeI18nextPlaceholders('{count} attempts left')).toBe('{count} attempts left')
    expect(
      normalizeI18nextPlaceholders('{count, plural, one {# item} other {# items}}'),
    ).toBe('{count, plural, one {# item} other {# items}}')
  })
})

describe('IcuFormatPlugin', () => {
  const plugin = new IcuFormatPlugin()

  function format(res: string, options: Record<string, unknown>): string {
    return plugin.parse(res, options, 'en', 'contentTools', 'test.key', {
      resolved: { res },
    })
  }

  it('interpolates legacy double-brace content-tool strings', () => {
    expect(format('About {{min}}–{{max}} words (a few sentences).', { min: 25, max: 150 })).toBe(
      'About 25–150 words (a few sentences).',
    )
    expect(
      format('{{count}} words (aim for {{min}}–{{max}})', { count: 0, min: 25, max: 150 }),
    ).toBe('0 words (aim for 25–150)')
    expect(format('{{count}} attempts left', { count: 3 })).toBe('3 attempts left')
  })

  it('interpolates ICU single-brace strings', () => {
    expect(format('{count} words (aim for {min}–{max})', { count: 12, min: 25, max: 150 })).toBe(
      '12 words (aim for 25–150)',
    )
  })

  it('interpolates ICU plurals', () => {
    expect(
      format('{count, plural, one {# assignment} other {# assignments}}', { count: 1 }),
    ).toBe('1 assignment')
    expect(
      format('{count, plural, one {# assignment} other {# assignments}}', { count: 2 }),
    ).toBe('2 assignments')
  })
})
