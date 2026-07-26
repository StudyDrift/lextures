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
