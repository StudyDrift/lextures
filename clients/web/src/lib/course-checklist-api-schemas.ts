import { z } from 'zod'

export const dismissReasonSchema = z.enum([
  'not_applicable',
  'done_elsewhere',
  'disagree',
  'later',
  'other',
])

export type DismissReason = z.infer<typeof dismissReasonSchema>

export const checklistStatusSchema = z.enum([
  'done',
  'todo',
  'in_progress',
  'unknown',
  'not_applicable',
])

export type ChecklistStatus = z.infer<typeof checklistStatusSchema>

const checklistNavTargetSchema = z
  .object({
    route: z.string(),
    anchor: z.string().nullish(),
  })
  .passthrough()

export type ChecklistNavTarget = z.infer<typeof checklistNavTargetSchema>

const checklistEvidenceRowSchema = z
  .object({
    label: z.string(),
    sublabel: z.string().nullish(),
    status: z.string(),
    target: checklistNavTargetSchema.nullish(),
  })
  .passthrough()

export type ChecklistEvidenceRow = z.infer<typeof checklistEvidenceRowSchema>

const checklistEvidenceSchema = z
  .object({
    columns: z.array(z.string()),
    rows: z.array(checklistEvidenceRowSchema),
    truncatedAt: z.number().int().nullish(),
  })
  .passthrough()

export type ChecklistEvidence = z.infer<typeof checklistEvidenceSchema>

const checklistDismissalSchema = z
  .object({
    dismissedAt: z.string(),
    byUserId: z.string(),
    byDisplayName: z.string(),
    reason: z.string(),
    note: z.string(),
  })
  .passthrough()

export type ChecklistDismissal = z.infer<typeof checklistDismissalSchema>

const checklistProgressSchema = z
  .object({
    done: z.number().int(),
    total: z.number().int(),
  })
  .passthrough()

export const checklistItemSchema = z
  .object({
    id: z.string(),
    titleKey: z.string(),
    title: z.string(),
    whyKey: z.string(),
    why: z.string(),
    tier: z.enum(['essential', 'recommended']),
    status: z.string(),
    detail: z.string().nullish(),
    progress: checklistProgressSchema.nullish(),
    sources: z.array(z.string()),
    helpRef: z.string().nullish(),
    target: checklistNavTargetSchema.nullish(),
    evidence: checklistEvidenceSchema.nullish(),
    dismissal: checklistDismissalSchema.nullish(),
  })
  .passthrough()

export type ChecklistItem = z.infer<typeof checklistItemSchema>

const checklistCategorySchema = z
  .object({
    id: z.string(),
    titleKey: z.string(),
    title: z.string(),
    items: z.array(checklistItemSchema),
  })
  .passthrough()

export type ChecklistCategory = z.infer<typeof checklistCategorySchema>

export const checklistSummarySchema = z
  .object({
    outstandingEssential: z.number().int(),
    outstandingTotal: z.number().int(),
    done: z.number().int(),
    total: z.number().int(),
    dismissed: z.number().int(),
    computedAt: z.string(),
    stale: z.boolean(),
  })
  .passthrough()

export type ChecklistSummary = z.infer<typeof checklistSummarySchema>

export const checklistResponseSchema = z
  .object({
    courseCode: z.string(),
    engineVersion: z.number().int(),
    catalogVersion: z.string(),
    computedAt: z.string(),
    stale: z.boolean(),
    evidenceTruncated: z.boolean(),
    summary: checklistSummarySchema,
    categories: z.array(checklistCategorySchema),
    dismissed: z.array(checklistItemSchema),
  })
  .passthrough()

export type ChecklistResponse = z.infer<typeof checklistResponseSchema>

export const dismissRequestSchema = z
  .object({
    reason: dismissReasonSchema,
    note: z.string().max(500).optional(),
  })
  .strict()

export type DismissRequest = z.infer<typeof dismissRequestSchema>

/** Normalize unknown status values to `unknown` for rendering (CC.7 §6). */
export function normalizeChecklistStatus(status: string): ChecklistStatus {
  const parsed = checklistStatusSchema.safeParse(status)
  return parsed.success ? parsed.data : 'unknown'
}

export function isOutstandingStatus(status: string): boolean {
  const s = normalizeChecklistStatus(status)
  return s === 'todo' || s === 'in_progress' || s === 'unknown'
}
