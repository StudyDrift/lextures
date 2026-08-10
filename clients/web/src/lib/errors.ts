import { parseValidationErrorResponse } from './form/parse-validation-error'
import type { FieldViolation } from './form/types'

/** Parses JSON error bodies returned by the StudyDrift API. */
export function readApiErrorMessage(raw: unknown): string {
  // UX.6 §9 envelope — prefer human summary, never render as HTML.
  const envelope = parseValidationErrorResponse(raw)
  if (envelope?.message) return envelope.message

  if (raw && typeof raw === 'object' && 'type' in raw) {
    const t = (raw as { type?: unknown }).type
    if (t === 'password_policy_violation') {
      const d = (raw as { detail?: unknown }).detail
      if (typeof d === 'string' && d.trim()) return d
    }
  }
  if (raw && typeof raw === 'object' && 'error' in raw) {
    const err = (raw as { error?: unknown }).error
    // Legacy nested shape: { error: { message } }
    if (err && typeof err === 'object' && err !== null && 'message' in err) {
      const m = (err as { message?: unknown }).message
      if (typeof m === 'string' && m.trim()) return m
    }
    // UX.6 top-level error code string without full envelope
    if (typeof err === 'string' && err.trim()) return err
  }
  if (raw && typeof raw === 'object' && 'message' in raw) {
    const m = (raw as { message?: unknown }).message
    if (typeof m === 'string') return m
  }
  return 'Request failed'
}

/**
 * UX.6 FR-9 — extract field-addressable 422 violations when present.
 * Returns empty array for legacy banner-only errors.
 */
export function readApiFieldErrors(raw: unknown): FieldViolation[] {
  const envelope = parseValidationErrorResponse(raw)
  return envelope?.fields ?? []
}

export type { FieldViolation }
