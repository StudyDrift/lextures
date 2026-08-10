/**
 * Pure helpers for modules-page structure drag-and-drop.
 *
 * Live `onDragOver` reordering often leaves `activeId === overId` on a successful
 * drop; that must still persist, or a refresh snaps the outline back.
 */
import { arrayMove } from '@dnd-kit/sortable'
import type { CourseStructureItem } from '../../lib/courses-api'

export type StructureReorderDropAction =
  | 'noop'
  | 'persist-current'
  | 'revert'
  | 'apply-over'

export function structureReorderDropAction(args: {
  hasCourseCode: boolean
  overId: string | null | undefined
  activeId: string
  committedDuringDrag: boolean
}): StructureReorderDropAction {
  if (!args.hasCourseCode) return 'noop'
  if (!args.overId) return args.committedDuringDrag ? 'revert' : 'noop'
  if (args.activeId === args.overId) {
    return args.committedDuringDrag ? 'persist-current' : 'noop'
  }
  return 'apply-over'
}

export const STRUCTURE_CHILD_KINDS = new Set<CourseStructureItem['kind']>([
  'heading',
  'content_page',
  'assignment',
  'quiz',
  'external_link',
  'survey',
  'lti_link',
  'h5p',
  'scorm',
  'vibe_activity',
  'library_resource',
  'textbook_resource',
  'attendance',
])

export function buildModuleChildrenMap(
  items: CourseStructureItem[],
): Map<string, CourseStructureItem[]> {
  const m = new Map<string, CourseStructureItem[]>()
  for (const i of items) {
    if (STRUCTURE_CHILD_KINDS.has(i.kind) && i.parentId) {
      const list = m.get(i.parentId) ?? []
      list.push(i)
      m.set(i.parentId, list)
    }
  }
  for (const [, list] of m) {
    list.sort((a, b) => a.sortOrder - b.sortOrder)
  }
  return m
}

export function findModuleIdForChildItem(
  childId: string,
  moduleChildrenById: Map<string, CourseStructureItem[]>,
): string | undefined {
  for (const [mid, list] of moduleChildrenById) {
    if (list.some((c) => c.id === childId)) return mid
  }
  return undefined
}

export function flattenOrderedStructure(
  topLevelOrdered: CourseStructureItem[],
  childrenByModule: Map<string, CourseStructureItem[]>,
): CourseStructureItem[] {
  let sortOrder = 0
  const out: CourseStructureItem[] = []
  for (const top of topLevelOrdered) {
    out.push({ ...top, sortOrder: sortOrder++ })
    if (top.kind === 'module') {
      for (const child of childrenByModule.get(top.id) ?? []) {
        out.push({ ...child, sortOrder: sortOrder++ })
      }
    }
  }
  return out
}

export function reorderModulesInStructure(
  items: CourseStructureItem[],
  activeModuleId: string,
  overModuleId: string,
): CourseStructureItem[] | null {
  const childrenByModule = buildModuleChildrenMap(items)
  const topLevel = items
    .filter((i) => !i.parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  const modules = topLevel.filter((i) => i.kind === 'module')
  const oldIndex = modules.findIndex((m) => m.id === activeModuleId)
  const newIndex = modules.findIndex((m) => m.id === overModuleId)
  if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) return null

  const nonModules = topLevel.filter((i) => i.kind !== 'module')
  const nextModules = arrayMove(modules, oldIndex, newIndex)
  return flattenOrderedStructure([...nonModules, ...nextModules], childrenByModule)
}

/**
 * UX.5 — absolute-index module move (click-to-move / "Move to…" menu).
 * `toIndex` is 0-based among top-level modules only.
 */
export function moveModuleToIndex(
  items: CourseStructureItem[],
  moduleId: string,
  toIndex: number,
): CourseStructureItem[] | null {
  const childrenByModule = buildModuleChildrenMap(items)
  const topLevel = items
    .filter((i) => !i.parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  const modules = topLevel.filter((i) => i.kind === 'module')
  const fromIndex = modules.findIndex((m) => m.id === moduleId)
  if (fromIndex < 0 || toIndex < 0 || toIndex >= modules.length || fromIndex === toIndex) {
    return null
  }
  const nonModules = topLevel.filter((i) => i.kind !== 'module')
  const nextModules = arrayMove(modules, fromIndex, toIndex)
  return flattenOrderedStructure([...nonModules, ...nextModules], childrenByModule)
}

/**
 * UX.5 — absolute-index child move within a module.
 */
export function moveChildToIndex(
  items: CourseStructureItem[],
  moduleId: string,
  childId: string,
  toIndex: number,
): CourseStructureItem[] | null {
  const childrenByModule = buildModuleChildrenMap(items)
  const children = [...(childrenByModule.get(moduleId) ?? [])]
  const fromIndex = children.findIndex((c) => c.id === childId)
  if (fromIndex < 0 || toIndex < 0 || toIndex >= children.length || fromIndex === toIndex) {
    return null
  }
  childrenByModule.set(moduleId, arrayMove(children, fromIndex, toIndex))
  const topLevel = items
    .filter((i) => !i.parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  return flattenOrderedStructure(topLevel, childrenByModule)
}

export function reorderChildrenInStructure(
  items: CourseStructureItem[],
  moduleId: string,
  activeChildId: string,
  overChildId: string,
): CourseStructureItem[] | null {
  const childrenByModule = buildModuleChildrenMap(items)
  const children = [...(childrenByModule.get(moduleId) ?? [])]
  const oldIndex = children.findIndex((c) => c.id === activeChildId)
  const newIndex = children.findIndex((c) => c.id === overChildId)
  if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) return null

  childrenByModule.set(moduleId, arrayMove(children, oldIndex, newIndex))
  const topLevel = items
    .filter((i) => !i.parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  return flattenOrderedStructure(topLevel, childrenByModule)
}

/**
 * Move a child item within its module or into another module.
 * - `overType === 'module'`: append to the end of that module (or no-op if already last there).
 * - `overType === 'child'` (or unknown but resolvable as a child): insert at that child's index.
 */
export function moveChildInStructure(
  items: CourseStructureItem[],
  activeChildId: string,
  overId: string,
  overType: 'child' | 'module' | undefined,
): CourseStructureItem[] | null {
  if (activeChildId === overId) return null

  const childrenByModule = buildModuleChildrenMap(items)
  const sourceModuleId = findModuleIdForChildItem(activeChildId, childrenByModule)
  if (!sourceModuleId) return null

  let targetModuleId: string | undefined
  let overChildId: string | null = null

  if (overType === 'module') {
    targetModuleId = overId
  } else if (overType === 'child') {
    targetModuleId = findModuleIdForChildItem(overId, childrenByModule)
    overChildId = overId
  } else {
    // Resolve from structure when type data is missing (keyboard / edge cases).
    targetModuleId = findModuleIdForChildItem(overId, childrenByModule)
    if (targetModuleId) {
      overChildId = overId
    } else if (items.some((i) => i.id === overId && i.kind === 'module' && !i.parentId)) {
      targetModuleId = overId
    }
  }

  if (!targetModuleId) return null

  // Same module: reorder among siblings (or no-op when dropping on own module card).
  if (sourceModuleId === targetModuleId) {
    if (!overChildId) return null
    return reorderChildrenInStructure(items, sourceModuleId, activeChildId, overChildId)
  }

  const sourceChildren = [...(childrenByModule.get(sourceModuleId) ?? [])]
  const targetChildren = [...(childrenByModule.get(targetModuleId) ?? [])]
  const fromIndex = sourceChildren.findIndex((c) => c.id === activeChildId)
  if (fromIndex < 0) return null

  const [moved] = sourceChildren.splice(fromIndex, 1)
  const movedWithParent: CourseStructureItem = { ...moved, parentId: targetModuleId }

  let toIndex = targetChildren.length
  if (overChildId) {
    const idx = targetChildren.findIndex((c) => c.id === overChildId)
    if (idx >= 0) toIndex = idx
  }
  targetChildren.splice(toIndex, 0, movedWithParent)

  childrenByModule.set(sourceModuleId, sourceChildren)
  childrenByModule.set(targetModuleId, targetChildren)

  const topLevel = items
    .filter((i) => !i.parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  return flattenOrderedStructure(topLevel, childrenByModule)
}

export function buildReorderPayloadFromItems(items: CourseStructureItem[]): {
  moduleOrder: string[]
  childOrderByModule: Record<string, string[]>
} {
  const modules = items
    .filter((i) => i.kind === 'module' && !i.parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  const moduleOrder = modules.map((m) => m.id)
  const childOrderByModule: Record<string, string[]> = {}
  for (const m of modules) {
    childOrderByModule[m.id] = items
      .filter((i) => i.parentId === m.id && STRUCTURE_CHILD_KINDS.has(i.kind))
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .map((c) => c.id)
  }
  return { moduleOrder, childOrderByModule }
}
