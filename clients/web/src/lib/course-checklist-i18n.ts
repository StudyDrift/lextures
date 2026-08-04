import type { DismissReason } from './course-checklist-api-schemas'

/** UI copy for the course checklist (CC.7). Server title/why used when no client key. */
export const courseChecklistI18n = {
  navLabel: 'Checklist',
  pageTitle: 'Course checklist',
  commandPaletteTitle: 'Course checklist',
  recheck: 'Re-check',
  rechecking: 'Re-checking…',
  progressDoneOfTotal: (done: number, total: number) => `${done} of ${total} done`,
  needAttention: (n: number) => (n === 1 ? '1 needs attention' : `${n} need attention`),
  checkedAgo: (relative: string) => `checked ${relative}`,
  outstandingCount: (n: number) => (n === 1 ? '1 outstanding' : `${n} outstanding`),
  completedLabel: 'Completed',
  essentialTier: 'Essential',
  showEvidence: (n: number) => `Show the ${n}`,
  hideEvidence: 'Hide evidence',
  evidenceTruncated: (shown: number, total: number) => `Showing first ${shown} of ${total}`,
  unknownDetail: "Couldn't check this right now",
  dismiss: 'Dismiss',
  restore: 'Restore',
  restoring: 'Restoring…',
  dismissedSection: (n: number) => `Dismissed (${n})`,
  dismissedBy: (name: string, when: string) => `Dismissed by ${name} · ${when}`,
  dismissDialogTitle: 'Dismiss checklist item',
  dismissDialogHelp: 'Explain why this item does not apply so your co-teachers can see it.',
  dismissReasonLabel: 'Reason',
  dismissNoteLabel: 'Note (optional)',
  dismissNotePlaceholder: 'Optional context for your co-teachers (max 500 characters)',
  dismissConfirm: 'Dismiss item',
  dismissCancel: 'Cancel',
  dismissReasons: {
    not_applicable: 'Not applicable',
    done_elsewhere: 'Done elsewhere',
    disagree: 'Disagree with this check',
    later: 'Later',
    other: 'Other',
  } satisfies Record<DismissReason, string>,
  allDoneTitle: 'Everything on the checklist is done',
  allDoneBody: 'Your course meets every active checklist item.',
  showCompleted: 'Show completed',
  hideCompleted: 'Hide completed',
  catalogEmpty: 'Nothing to check right now.',
  noAccess: "You don't have access to this page.",
  loadError: 'Could not load the course checklist.',
  retry: 'Retry',
  itemDismissedLive: 'Item dismissed',
  itemRestoredLive: 'Item restored',
  itemRecheckedLive: 'Re-checked',
  badgeAria: (n: number) =>
    n === 1 ? '1 checklist item needs attention' : `${n} checklist items need attention`,
  dashboardTitle: 'Course checklist',
  dashboardComplete: 'Your course checklist is complete',
  dashboardViewAll: 'Open checklist',
  dashboardProgress: (done: number, total: number) => `${done} of ${total} done`,
  whyExpand: 'Why this matters',
  overflowMenu: 'Item actions',
  aiActionSlot: '', // reserved for CC.10
} as const

export function dismissReasonLabel(reason: string): string {
  const key = reason as DismissReason
  return courseChecklistI18n.dismissReasons[key] ?? reason
}
