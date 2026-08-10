import type { FieldViolation, ValidationErrorResponse } from './types'

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null
}

/**
 * Parse UX.6 §9 `validation_failed` envelope.
 * Returns null when the body is not that shape (caller falls back to banner).
 */
export function parseValidationErrorResponse(raw: unknown): ValidationErrorResponse | null {
  if (!isRecord(raw)) return null
  if (raw.error !== 'validation_failed') return null
  if (typeof raw.message !== 'string') return null
  if (!Array.isArray(raw.fields)) return null

  const fields: FieldViolation[] = []
  for (const item of raw.fields) {
    if (!isRecord(item)) continue
    if (typeof item.path !== 'string' || !item.path) continue
    const code = typeof item.code === 'string' && item.code ? item.code : 'custom'
    const message =
      typeof item.message === 'string' && item.message.trim()
        ? item.message
        : 'Enter a valid value for this field.'
    const params =
      item.params && isRecord(item.params) ? (item.params as Record<string, unknown>) : undefined
    fields.push({ path: item.path, code, message, params })
  }
  return { error: 'validation_failed', message: raw.message, fields }
}

/**
 * Best-effort extraction of field violations from legacy server shapes
 * (content-tools `error.errors[]`, custom-field validation, etc.).
 */
export function parseLegacyFieldErrors(raw: unknown): FieldViolation[] {
  if (!isRecord(raw)) return []

  // { error: { errors: [{ path, message }] } }
  if (isRecord(raw.error) && Array.isArray(raw.error.errors)) {
    return raw.error.errors
      .filter(isRecord)
      .map((e) => ({
        path: typeof e.path === 'string' ? e.path : String(e.field ?? ''),
        code: typeof e.code === 'string' ? e.code : 'custom',
        message: typeof e.message === 'string' ? e.message : 'Enter a valid value for this field.',
      }))
      .filter((e) => e.path)
  }

  // { fields: { name: "msg" } } map form
  if (isRecord(raw.fields) && !Array.isArray(raw.fields)) {
    return Object.entries(raw.fields)
      .filter(([, v]) => typeof v === 'string')
      .map(([path, message]) => ({
        path,
        code: 'custom',
        message: message as string,
      }))
  }

  return []
}

/**
 * Unified parser: prefer the UX.6 envelope, then legacy shapes.
 * Returns field violations and an optional page-level message.
 */
export function readFieldAddressableErrors(raw: unknown): {
  formMessage: string | null
  fields: FieldViolation[]
  isEnvelope: boolean
} {
  const envelope = parseValidationErrorResponse(raw)
  if (envelope) {
    return {
      formMessage: envelope.message || null,
      fields: envelope.fields,
      isEnvelope: true,
    }
  }
  const legacy = parseLegacyFieldErrors(raw)
  return { formMessage: null, fields: legacy, isEnvelope: false }
}
