import type { JsonSchema, SchemaFieldError } from './types'

/** Client-side required-field check before PATCH. */
export function validateRequiredFields(
  schema: JsonSchema,
  value: Record<string, unknown>,
): SchemaFieldError[] {
  const required = schema.required ?? []
  const errors: SchemaFieldError[] = []
  for (const key of required) {
    const v = value[key]
    if (v === undefined || v === null || v === '') {
      errors.push({ path: key, message: 'This field is required.' })
    }
  }
  return errors
}

export type ParseConfigJSONResult =
  | { ok: true; config: Record<string, unknown> }
  | { ok: false; errors: SchemaFieldError[] }

/**
 * Parse pasted tool config JSON and run client-side schema checks (required fields).
 * Full structural validation is enforced by the server on save.
 */
export function parseAndValidateConfigJSON(
  raw: string,
  schema: JsonSchema,
): ParseConfigJSONResult {
  const trimmed = raw.trim()
  if (!trimmed) {
    return { ok: false, errors: [{ path: '', message: 'JSON is empty.' }] }
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed) as unknown
  } catch {
    return { ok: false, errors: [{ path: '', message: 'Invalid JSON.' }] }
  }

  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return {
      ok: false,
      errors: [{ path: '', message: 'Config must be a JSON object.' }],
    }
  }

  const config = parsed as Record<string, unknown>
  const fieldErrors = validateRequiredFields(schema, config)
  if (fieldErrors.length > 0) {
    return { ok: false, errors: fieldErrors }
  }
  return { ok: true, config }
}
