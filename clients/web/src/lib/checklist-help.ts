/**
 * Per-item checklist help content (CC.10 FR-1–FR-3).
 * Source of truth: docs/help/course-checklist/items.json (copied to packages).
 */

import helpCatalog from './checklist-help-data.json'

export type ChecklistHelpEntry = {
  itemId: string
  helpRef: string
  title: string
  what: string
  why: string
  how: string
  whenToDismiss: string
  sources: string[]
}

type HelpCatalog = {
  version: number
  items: Record<string, ChecklistHelpEntry>
}

const catalog = helpCatalog as HelpCatalog

/** Stable support URL for a HelpRef like `course-checklist#outcomes-assessment-mapping`. */
export function helpSupportUrl(helpRef: string | null | undefined): string | null {
  if (!helpRef) return null
  const slug = helpRef.includes('#') ? helpRef.split('#')[1] : helpRef
  if (!slug) return null
  return `/help/course-checklist#${slug}`
}

/** Resolve four-part help content for a HelpRef (dangling refs return null). */
export function resolveChecklistHelp(helpRef: string | null | undefined): ChecklistHelpEntry | null {
  if (!helpRef) return null
  const entry = catalog.items[helpRef]
  return entry ?? null
}

/**
 * Stable in-app URL for source chips → rule-to-standard research (CC.10 FR-4).
 * Lives under /help (not /docs/plan) so plan-folder moves do not break product links.
 */
export const COURSE_DESIGN_RESEARCH_HREF = '/help/course-checklist/research'

export function listChecklistHelpRefs(): string[] {
  return Object.keys(catalog.items)
}
