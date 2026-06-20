import { authorizedFetch } from './api'

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
    throw new Error(text || `Request failed: ${res.status}`)
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

export async function putItemCompletionRule(
  courseCode: string,
  itemId: string,
  body: { ruleType: ItemCompletionRuleType; threshold?: number | null },
): Promise<ItemCompletionRule> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/completion-rule`,
    { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) },
  )
  return parseJson(res)
}

export async function deleteItemCompletionRule(courseCode: string, itemId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/completion-rule`,
    { method: 'DELETE' },
  )
  await parseJson(res)
}

export async function fetchRequirementsReport(
  courseCode: string,
): Promise<{ rows: RequirementsReportRow[] }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/requirements/report`,
  )
  return parseJson(res)
}

export async function postModuleUnlockOverride(
  courseCode: string,
  moduleId: string,
  enrollmentId: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/unlock-override`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enrollmentId }),
    },
  )
  await parseJson(res)
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
