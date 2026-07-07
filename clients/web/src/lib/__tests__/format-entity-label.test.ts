import { describe, expect, it } from 'vitest'
import { formatEntityLabel } from '../format-entity-label'

describe('formatEntityLabel', () => {
  it('prefers display name', () => {
    expect(
      formatEntityLabel({ name: 'Alex Kim', pseudonym: 'Student 2', fallback: 'Unknown student' }),
    ).toBe('Alex Kim')
  })

  it('uses pseudonym when name is absent', () => {
    expect(formatEntityLabel({ name: null, pseudonym: 'Reviewer 3', fallback: 'Unknown reviewer' })).toBe(
      'Reviewer 3',
    )
  })

  it('falls back to neutral label instead of raw ids', () => {
    expect(formatEntityLabel({ name: '', pseudonym: '  ', fallback: 'Unknown staff' })).toBe('Unknown staff')
  })
})