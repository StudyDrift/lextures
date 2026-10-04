import type { ChecklistItem, ChecklistResponse } from './course-checklist-api-schemas'

/** Checklist item that crawls outbound links in the background. */
export const EXTERNAL_LINK_HEALTH_ITEM_ID = 'links.external-health'

/** Server detail while a link-health job is still in flight. */
export const EXTERNAL_LINK_CHECKING_DETAIL = 'Checking links…'

export function isExternalLinkCheckPending(item: { id: string; detail?: string | null }): boolean {
  return item.id === EXTERNAL_LINK_HEALTH_ITEM_ID && (item.detail ?? '').startsWith('Checking links')
}

export function checklistHasPendingLinkCheck(
  data: { categories: { items: { id: string; detail?: string | null }[] }[] } | null,
): boolean {
  if (!data) return false
  return data.categories.some((category) => category.items.some(isExternalLinkCheckPending))
}

export function replaceChecklistItem(data: ChecklistResponse, item: ChecklistItem): ChecklistResponse {
  return {
    ...data,
    categories: data.categories.map((category) => ({
      ...category,
      items: category.items.map((current) => (current.id === item.id ? item : current)),
    })),
  }
}
