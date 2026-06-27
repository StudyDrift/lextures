import type { CourseStructureItem } from '../../../lib/courses-api'

export type CourseQuizOption = {
  id: string
  title: string
  moduleTitle: string
}

function resolveModuleTitle(item: CourseStructureItem, byId: Map<string, CourseStructureItem>): string {
  let parent: CourseStructureItem | undefined = item.parentId ? byId.get(item.parentId) : undefined
  const guard = new Set<string>()
  while (parent && !guard.has(parent.id)) {
    guard.add(parent.id)
    if (parent.kind === 'module') return parent.title.trim() || 'Untitled module'
    parent = parent.parentId ? byId.get(parent.parentId) : undefined
  }
  return ''
}

export function quizOptionsFromStructure(items: CourseStructureItem[]): CourseQuizOption[] {
  const byId = new Map(items.map((item) => [item.id, item]))
  const quizzes = items
    .filter((item) => item.kind === 'quiz')
    .map((item) => ({
      id: item.id,
      title: item.title.trim() || 'Untitled quiz',
      moduleTitle: resolveModuleTitle(item, byId),
    }))

  return [...quizzes].sort((a, b) => {
    const moduleCmp = a.moduleTitle.localeCompare(b.moduleTitle, undefined, {
      sensitivity: 'base',
      numeric: true,
    })
    if (moduleCmp !== 0) return moduleCmp
    return a.title.localeCompare(b.title, undefined, { sensitivity: 'base', numeric: true })
  })
}

export function filterQuizOptions(options: CourseQuizOption[], query: string): CourseQuizOption[] {
  const needle = query.trim().toLowerCase()
  if (!needle) return options
  return options.filter(
    (option) =>
      option.title.toLowerCase().includes(needle) ||
      option.moduleTitle.toLowerCase().includes(needle),
  )
}