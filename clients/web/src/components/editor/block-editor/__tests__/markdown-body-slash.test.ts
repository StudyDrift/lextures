import { describe, expect, it } from 'vitest'
import {
  filterSlashCommands,
  getBlockSlashRange,
  getSlashState,
  slashCommandsForEditor,
} from '../markdown-body-slash'

describe('getSlashState', () => {
  it('detects slash at block start', () => {
    expect(getSlashState('/', 1)).toEqual({ start: 0, query: '' })
  })

  it('detects slash after whitespace', () => {
    expect(getSlashState('hello /he', 9)).toEqual({ start: 6, query: 'he' })
  })

  it('rejects slash mid-word', () => {
    expect(getSlashState('foo/bar', 7)).toBeNull()
  })

  it('rejects slash query with spaces', () => {
    expect(getSlashState('/hello world', 13)).toBeNull()
  })
})

describe('filterSlashCommands', () => {
  const commands = slashCommandsForEditor({ equation: true })

  it('returns all commands for empty query', () => {
    expect(filterSlashCommands(commands, '')).toHaveLength(commands.length)
  })

  it('filters by label', () => {
    const filtered = filterSlashCommands(commands, 'head')
    expect(filtered.map((c) => c.id)).toEqual(['heading1', 'heading2', 'heading3'])
  })

  it('filters by keyword', () => {
    const filtered = filterSlashCommands(commands, 'latex')
    expect(filtered.some((c) => c.id === 'equation')).toBe(true)
  })

  it('filters image by id and photo keyword', () => {
    const commands = slashCommandsForEditor({ image: true })
    expect(filterSlashCommands(commands, 'image').map((c) => c.id)).toEqual(['image'])
    expect(filterSlashCommands(commands, 'photo').map((c) => c.id)).toEqual(['image'])
  })

  it('does not match unrelated blocks for photo query', () => {
    const commands = slashCommandsForEditor({ image: false })
    expect(filterSlashCommands(commands, 'photo').some((c) => c.id === 'paragraph')).toBe(false)
    expect(filterSlashCommands(commands, 'photo')).toEqual([])
  })

  it('omits image command when disabled', () => {
    const commands = slashCommandsForEditor({ image: false })
    expect(commands.some((c) => c.id === 'image')).toBe(false)
  })
})

describe('getBlockSlashRange', () => {
  it('returns null for non-text blocks', () => {
    expect(getBlockSlashRange({ selection: { empty: false } } as never)).toBeNull()
  })
})
