import { describe, expect, it } from 'vitest'
import {
  applyCoursePickerSelection,
  courseMatchesScope,
  parseCoursePickerState,
  parseSearchQuery,
} from '../search-query-parse'

describe('parseSearchQuery', () => {
  it('returns empty text for blank input', () => {
    expect(parseSearchQuery('')).toMatchObject({ text: '', scopeCourseCode: null, types: null })
  })

  it('extracts in: scope', () => {
    expect(parseSearchQuery('in:BIOL-101 midterm')).toMatchObject({
      scopeCourseCode: 'biol-101',
      text: 'midterm',
    })
  })

  it('supports multiple type: prefixes', () => {
    const parsed = parseSearchQuery('type:course type:content essay')
    expect(parsed.types).toEqual(new Set(['course', 'content']))
    expect(parsed.text).toBe('essay')
  })
})

describe('courseMatchesScope', () => {
  it('matches case-insensitively', () => {
    expect(courseMatchesScope('BIOL-101', 'biol-101')).toBe(true)
    expect(courseMatchesScope('BIOL-101', 'chem-200')).toBe(false)
  })
})

describe('parseCoursePickerState', () => {
  it('activates on bare @', () => {
    expect(parseCoursePickerState('@')).toMatchObject({ active: true, filter: '', atIndex: 0 })
  })

  it('activates while typing a partial course code', () => {
    expect(parseCoursePickerState('@bio')).toMatchObject({ active: true, filter: 'bio', atIndex: 0 })
  })

  it('deactivates after scope is completed with a trailing space', () => {
    expect(parseCoursePickerState('@bio ')).toMatchObject({ active: false })
    expect(parseCoursePickerState('@bio gradebook')).toMatchObject({ active: false })
  })

  it('supports @ picker suffix after other text', () => {
    expect(parseCoursePickerState('type:page @chem')).toMatchObject({
      active: true,
      filter: 'chem',
    })
  })
})

describe('applyCoursePickerSelection', () => {
  it('inserts the course code and trailing space', () => {
    expect(applyCoursePickerSelection('@bio', 0, 'BIOL-101')).toBe('@BIOL-101 ')
    expect(applyCoursePickerSelection('hello @b', 6, 'CHEM-1')).toBe('hello @CHEM-1 ')
  })
})
