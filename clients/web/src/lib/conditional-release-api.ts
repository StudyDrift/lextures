import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ModuleCompletionMode = 'all_items' | 'one_item' | 'sequential_order'

export type ItemCompletionRuleType =
  | 'must_view'
  | 'must_mark_done'
  | 'must_submit'
  | 'must_score_at_least'
  | 'must_contribute'

export type ModuleRequirement = {
  moduleId: string
  completionMode: ModuleCompletionMode
  unlockAt?: string | null
  /** API field name from Go model. */
  prerequisiteModuleIds?: string[]
  /** Legacy alias kept for older clients. */
  prerequisiteIds?: string[]
}

export type ItemCompletionRule = {
  itemId: string
  ruleType: ItemCompletionRuleType
  threshold?: number | null
}

export type LockReason = {
  code: string
  message: string
  itemId?: string
  title?: string
}

export type ItemLockState = {
  itemId: string
  locked: boolean
  complete: boolean
  reason?: LockReason | null
}

export type ModuleLockState = {
  moduleId: string
  title: string
  sortOrder: number
  locked: boolean
  complete: boolean
  reason?: LockReason | null
  items?: ItemLockState[]
}

export type ModulesProgressSnapshot = {
  enrollmentId: string
  modules: ModuleLockState[]
}

export type RequirementsReportRow = {
  enrollmentId: string
  userId: string
  displayName: string
  email: string
  itemId: string
  itemTitle: string
  moduleTitle: string
  ruleType?: string
  status: string
  metAt?: string
}

async function parseJson<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    if (text) {
      try {
        throw new Error(readApiErrorMessage(JSON.parse(text) as unknown))
      } catch (e) {
        if (e instanceof SyntaxError) {
          throw new Error(text)
        }
        throw e
      }
    }
    throw new Error(`Request failed: ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export async function fetchModulesProgress(courseCode: string): Promise<ModulesProgressSnapshot> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/modules/progress`,
  )
  return parseJson(res)
}

/** Loads module requirements. Returns null when none are configured yet. */
export async function fetchModuleRequirements(
  courseCode: string,
  moduleId: string,
): Promise<ModuleRequirement | null> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/requirements`,
  )
  if (res.status === 404) return null
  return parseJson(res)
}

export async function putModuleRequirements(
  courseCode: string,
  moduleId: string,
  body: {
    completionMode: ModuleCompletionMode
    prerequisiteModuleIds?: string[]
    unlockAt?: string | null
  },
): Promise<ModuleRequirement> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/requirements`,
    { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) },
  )
  return parseJson(res)
}

export function itemLockState(
  progress: ModulesProgressSnapshot | null,
  itemId: string,
): ItemLockState | null {
  if (!progress) return null
  for (const mod of progress.modules) {
    for (const item of mod.items ?? []) {
      if (item.itemId === itemId) return item
    }
  }
  return null
}

export function moduleLockState(
  progress: ModulesProgressSnapshot | null,
  moduleId: string,
): ModuleLockState | null {
  if (!progress) return null
  return progress.modules.find((m) => m.moduleId === moduleId) ?? null
}
