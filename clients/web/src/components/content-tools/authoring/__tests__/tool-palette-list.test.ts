import { describe, expect, it } from 'vitest'
import {
  filterToolPalette,
  groupToolsByCategory,
  type ToolPaletteItem,
} from '../tool-palette-utils'

const tools: ToolPaletteItem[] = [
  {
    id: 'noop_probe',
    name: 'No-op probe',
    category: 'assess',
    description: 'Built-in test tool',
    keywords: ['noop', 'probe'],
    ui: { renderer: 'noop_probe', icon: 'flask', group: 'assess' },
  },
  {
    id: 'ask_questions',
    name: 'Ask Questions',
    category: 'discuss',
    description: 'Open-ended prompts',
    keywords: ['ask', 'questions'],
    ui: { renderer: 'ask', icon: 'message', group: 'discuss' },
  },
]

describe('filterToolPalette', () => {
  it('returns all tools for empty query', () => {
    expect(filterToolPalette(tools, '')).toHaveLength(2)
  })

  it('filters by id / keyword', () => {
    expect(filterToolPalette(tools, 'noop').map((t) => t.id)).toEqual(['noop_probe'])
    expect(filterToolPalette(tools, 'probe').map((t) => t.id)).toEqual(['noop_probe'])
  })

  it('filters by category', () => {
    expect(filterToolPalette(tools, 'discuss').map((t) => t.id)).toEqual(['ask_questions'])
  })
})

describe('groupToolsByCategory', () => {
  it('groups and sorts categories', () => {
    const groups = groupToolsByCategory(tools)
    expect(groups.map((g) => g.category)).toEqual(['assess', 'discuss'])
  })
})
