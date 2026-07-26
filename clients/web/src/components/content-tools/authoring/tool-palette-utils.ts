import type { ContentToolsCatalogTool } from '../../../lib/courses-api'

export type ToolPaletteItem = Pick<
  ContentToolsCatalogTool,
  'id' | 'name' | 'category' | 'ui'
> & {
  description?: string
  keywords?: string[]
}

export function filterToolPalette(
  tools: ToolPaletteItem[],
  query: string,
): ToolPaletteItem[] {
  const q = query.trim().toLowerCase()
  if (!q) return tools
  return tools.filter((tool) => {
    if (tool.id.toLowerCase().includes(q)) return true
    if (tool.name.toLowerCase().includes(q)) return true
    if ((tool.description ?? '').toLowerCase().includes(q)) return true
    if (tool.category.toLowerCase().includes(q)) return true
    return (tool.keywords ?? []).some((kw) => kw.toLowerCase().includes(q))
  })
}

export function groupToolsByCategory(
  tools: ToolPaletteItem[],
): Array<{ category: string; tools: ToolPaletteItem[] }> {
  const map = new Map<string, ToolPaletteItem[]>()
  for (const tool of tools) {
    const cat = tool.category || tool.ui.group || 'other'
    const list = map.get(cat) ?? []
    list.push(tool)
    map.set(cat, list)
  }
  return [...map.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([category, groupTools]) => ({ category, tools: groupTools }))
}
