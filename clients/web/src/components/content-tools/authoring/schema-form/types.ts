/** Minimal JSON Schema subset used by the generic Content Tools config form (CT.2). */

export type JsonSchema = {
  $schema?: string
  type?: string | string[]
  title?: string
  description?: string
  default?: unknown
  enum?: unknown[]
  properties?: Record<string, JsonSchema>
  required?: string[]
  items?: JsonSchema
  minimum?: number
  maximum?: number
  minLength?: number
  maxLength?: number
  additionalProperties?: boolean | JsonSchema
  format?: string
  'x-lex-sensitive'?: boolean
  'x-lex-multiline'?: boolean
}

export type SchemaFieldError = { path: string; message: string }

export function schemaType(schema: JsonSchema): string | undefined {
  if (Array.isArray(schema.type)) {
    return schema.type.find((t) => t !== 'null')
  }
  return schema.type
}

export function isMultilineString(schema: JsonSchema): boolean {
  if (schema['x-lex-multiline'] === true) return true
  if (schema.format === 'textarea' || schema.format === 'markdown') return true
  const hint = `${schema.title ?? ''} ${schema.description ?? ''}`.toLowerCase()
  if (hint.includes('markdown') || hint.includes('multiline')) return true
  if (typeof schema.maxLength === 'number' && schema.maxLength <= 64) return false
  return true
}

export function labelForSchema(schema: JsonSchema, fallbackKey: string): string {
  if (schema.title?.trim()) return schema.title.trim()
  if (schema.description?.trim()) return schema.description.trim()
  return fallbackKey
}

export function errorsForPath(
  errors: SchemaFieldError[],
  path: string,
): SchemaFieldError[] {
  return errors.filter((e) => e.path === path || e.path === `/${path}` || e.path.endsWith(`.${path}`))
}
