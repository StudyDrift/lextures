import { ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import {
  errorsForPath,
  isMultilineString,
  labelForSchema,
  schemaType,
  type JsonSchema,
  type SchemaFieldError,
} from './types'

export type SchemaFormProps = {
  schema: JsonSchema
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  errors?: SchemaFieldError[]
  disabled?: boolean
  idPrefix?: string
  /** Path prefix for nested error mapping (e.g. "" or "items.0"). */
  pathPrefix?: string
}

function fieldPath(prefix: string, key: string): string {
  return prefix ? `${prefix}.${key}` : key
}

function FieldErrorText({ id, message }: { id: string; message: string }) {
  return (
    <p id={id} className="mt-1 text-xs text-rose-600 dark:text-rose-400" role="alert">
      {message}
    </p>
  )
}

function FieldShell({
  label,
  htmlFor,
  description,
  errorId,
  errorMessage,
  children,
}: {
  label: string
  htmlFor: string
  description?: string
  errorId?: string
  errorMessage?: string
  children: ReactNode
}) {
  return (
    <div className="space-y-1">
      <label htmlFor={htmlFor} className="block text-xs font-medium text-slate-600 dark:text-neutral-300">
        {label}
      </label>
      {description ? (
        <p className="text-[11px] leading-snug text-slate-500 dark:text-neutral-400">{description}</p>
      ) : null}
      {children}
      {errorMessage && errorId ? <FieldErrorText id={errorId} message={errorMessage} /> : null}
    </div>
  )
}

const inputClass =
  'w-full rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-neutral-500 dark:focus:ring-neutral-500'

function setAtPath(
  root: Record<string, unknown>,
  key: string,
  nextValue: unknown,
): Record<string, unknown> {
  return { ...root, [key]: nextValue }
}

function renderPrimitive(
  schema: JsonSchema,
  value: unknown,
  onChange: (v: unknown) => void,
  opts: {
    id: string
    disabled?: boolean
    errorId?: string
    describedBy?: string
  },
): ReactNode {
  const t = schemaType(schema)
  if (schema.enum && schema.enum.length > 0) {
    const options = schema.enum
    if (options.length <= 5) {
      return (
        <div role="radiogroup" aria-labelledby={opts.id} className="flex flex-col gap-1.5">
          {options.map((opt, i) => {
            const optId = `${opts.id}-opt-${i}`
            const str = String(opt)
            return (
              <label key={optId} htmlFor={optId} className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
                <input
                  id={optId}
                  type="radio"
                  name={opts.id}
                  checked={value === opt}
                  disabled={opts.disabled}
                  onChange={() => onChange(opt)}
                  className="border-slate-300 text-slate-700 focus:ring-slate-400 dark:border-neutral-600"
                />
                {str}
              </label>
            )
          })}
        </div>
      )
    }
    return (
      <select
        id={opts.id}
        value={value == null ? '' : String(value)}
        disabled={opts.disabled}
        aria-describedby={opts.describedBy}
        onChange={(e) => {
          const raw = e.target.value
          const match = options.find((o) => String(o) === raw)
          onChange(match ?? raw)
        }}
        className={inputClass}
      >
        <option value="">Select…</option>
        {options.map((opt, i) => (
          <option key={`${opts.id}-${i}`} value={String(opt)}>
            {String(opt)}
          </option>
        ))}
      </select>
    )
  }

  if (t === 'boolean') {
    return (
      <input
        id={opts.id}
        type="checkbox"
        checked={Boolean(value)}
        disabled={opts.disabled}
        aria-describedby={opts.describedBy}
        onChange={(e) => onChange(e.target.checked)}
        className="rounded border-slate-300 text-slate-700 focus:ring-slate-400 dark:border-neutral-600"
      />
    )
  }

  if (t === 'integer' || t === 'number') {
    return (
      <input
        id={opts.id}
        type="number"
        step={t === 'integer' ? 1 : 'any'}
        min={schema.minimum}
        max={schema.maximum}
        value={typeof value === 'number' ? value : value == null ? '' : String(value)}
        disabled={opts.disabled}
        aria-describedby={opts.describedBy}
        onChange={(e) => {
          const raw = e.target.value
          if (raw === '') {
            onChange(undefined)
            return
          }
          const n = t === 'integer' ? Number.parseInt(raw, 10) : Number.parseFloat(raw)
          onChange(Number.isFinite(n) ? n : undefined)
        }}
        className={inputClass}
      />
    )
  }

  if (t === 'string' || t == null) {
    if (isMultilineString(schema)) {
      return (
        <textarea
          id={opts.id}
          rows={3}
          value={typeof value === 'string' ? value : ''}
          disabled={opts.disabled}
          aria-describedby={opts.describedBy}
          onChange={(e) => onChange(e.target.value)}
          className={`${inputClass} min-h-[4.5rem] resize-y`}
        />
      )
    }
    return (
      <input
        id={opts.id}
        type="text"
        value={typeof value === 'string' ? value : ''}
        disabled={opts.disabled}
        aria-describedby={opts.describedBy}
        onChange={(e) => onChange(e.target.value)}
        className={inputClass}
      />
    )
  }

  return (
    <p className="text-xs text-slate-500 dark:text-neutral-400">Unsupported field type: {String(t)}</p>
  )
}

function ArrayOfObjectsField({
  schema,
  value,
  onChange,
  errors,
  disabled,
  idPrefix,
  pathPrefix,
  fieldKey,
}: {
  schema: JsonSchema
  value: unknown[]
  onChange: (next: unknown[]) => void
  errors: SchemaFieldError[]
  disabled?: boolean
  idPrefix: string
  pathPrefix: string
  fieldKey: string
}) {
  const itemSchema = schema.items ?? { type: 'object', properties: {} }
  const rows = Array.isArray(value) ? value : []

  function move(index: number, delta: number) {
    const nextIndex = index + delta
    if (nextIndex < 0 || nextIndex >= rows.length) return
    const next = [...rows]
    const tmp = next[index]
    next[index] = next[nextIndex]!
    next[nextIndex] = tmp!
    onChange(next)
  }

  function remove(index: number) {
    onChange(rows.filter((_, i) => i !== index))
  }

  function add() {
    const defaults: Record<string, unknown> = {}
    const props = itemSchema.properties ?? {}
    for (const [k, s] of Object.entries(props)) {
      if (s.default !== undefined) defaults[k] = s.default
    }
    onChange([...rows, defaults])
  }

  return (
    <fieldset className="space-y-2 rounded-md border border-slate-200 p-2 dark:border-neutral-700">
      <legend className="px-1 text-xs font-medium text-slate-600 dark:text-neutral-300">
        {labelForSchema(schema, fieldKey)}
      </legend>
      {rows.map((row, index) => {
        const rowObj =
          row && typeof row === 'object' && !Array.isArray(row)
            ? (row as Record<string, unknown>)
            : {}
        return (
          <div
            key={`${idPrefix}-${fieldKey}-${index}`}
            className="rounded border border-slate-100 bg-slate-50/60 p-2 dark:border-neutral-700 dark:bg-neutral-900/40"
          >
            <div className="mb-2 flex items-center justify-end gap-1">
              <button
                type="button"
                disabled={disabled || index === 0}
                aria-label="Move up"
                onClick={() => move(index, -1)}
                className="rounded p-1 text-slate-600 hover:bg-slate-200 disabled:opacity-40 dark:text-neutral-300 dark:hover:bg-neutral-700"
              >
                <ChevronUp className="h-3.5 w-3.5" aria-hidden />
              </button>
              <button
                type="button"
                disabled={disabled || index === rows.length - 1}
                aria-label="Move down"
                onClick={() => move(index, 1)}
                className="rounded p-1 text-slate-600 hover:bg-slate-200 disabled:opacity-40 dark:text-neutral-300 dark:hover:bg-neutral-700"
              >
                <ChevronDown className="h-3.5 w-3.5" aria-hidden />
              </button>
              <button
                type="button"
                disabled={disabled}
                aria-label="Remove row"
                onClick={() => remove(index)}
                className="rounded p-1 text-rose-600 hover:bg-rose-50 disabled:opacity-40 dark:text-rose-400 dark:hover:bg-rose-950/40"
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden />
              </button>
            </div>
            <SchemaForm
              schema={itemSchema}
              value={rowObj}
              onChange={(next) => {
                const copy = [...rows]
                copy[index] = next
                onChange(copy)
              }}
              errors={errors}
              disabled={disabled}
              idPrefix={`${idPrefix}-${fieldKey}-${index}`}
              pathPrefix={fieldPath(pathPrefix, `${fieldKey}.${index}`)}
            />
          </div>
        )
      })}
      <button
        type="button"
        disabled={disabled}
        onClick={add}
        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-100 disabled:opacity-40 dark:text-neutral-200 dark:hover:bg-neutral-800"
      >
        <Plus className="h-3.5 w-3.5" aria-hidden />
        Add item
      </button>
    </fieldset>
  )
}

export function SchemaForm({
  schema,
  value,
  onChange,
  errors = [],
  disabled,
  idPrefix = 'schema',
  pathPrefix = '',
}: SchemaFormProps) {
  const properties = schema.properties ?? {}
  const required = new Set(schema.required ?? [])
  const t = schemaType(schema)

  if (t === 'object' || (t == null && Object.keys(properties).length > 0)) {
    return (
      <div className="space-y-3">
        {Object.entries(properties).map(([key, propSchema]) => {
          const path = fieldPath(pathPrefix, key)
          const id = `${idPrefix}-${path.replace(/\./g, '-')}`
          const errorId = `${id}-error`
          const fieldErrors = errorsForPath(errors, path)
          const errorMessage = fieldErrors[0]?.message
          const describedBy = [
            propSchema.description ? `${id}-desc` : null,
            errorMessage ? errorId : null,
          ]
            .filter(Boolean)
            .join(' ') || undefined
          const label = labelForSchema(propSchema, key) + (required.has(key) ? ' *' : '')
          const propType = schemaType(propSchema)
          const current = value[key]

          if (propType === 'object' && propSchema.properties) {
            const nested =
              current && typeof current === 'object' && !Array.isArray(current)
                ? (current as Record<string, unknown>)
                : {}
            return (
              <fieldset
                key={key}
                className="space-y-2 rounded-md border border-slate-200 p-2 dark:border-neutral-700"
              >
                <legend className="px-1 text-xs font-medium text-slate-600 dark:text-neutral-300">
                  {label}
                </legend>
                <SchemaForm
                  schema={propSchema}
                  value={nested}
                  onChange={(next) => onChange(setAtPath(value, key, next))}
                  errors={errors}
                  disabled={disabled}
                  idPrefix={id}
                  pathPrefix={path}
                />
                {errorMessage ? <FieldErrorText id={errorId} message={errorMessage} /> : null}
              </fieldset>
            )
          }

          if (propType === 'array') {
            const itemType = propSchema.items ? schemaType(propSchema.items) : undefined
            if (itemType === 'object' || propSchema.items?.properties) {
              return (
                <ArrayOfObjectsField
                  key={key}
                  schema={propSchema}
                  value={Array.isArray(current) ? current : []}
                  onChange={(next) => onChange(setAtPath(value, key, next))}
                  errors={errors}
                  disabled={disabled}
                  idPrefix={idPrefix}
                  pathPrefix={pathPrefix}
                  fieldKey={key}
                />
              )
            }
            // Primitive arrays: comma-separated for simplicity
            const arr = Array.isArray(current) ? current.map(String) : []
            return (
              <FieldShell
                key={key}
                label={label}
                htmlFor={id}
                description={propSchema.description}
                errorId={errorMessage ? errorId : undefined}
                errorMessage={errorMessage}
              >
                <input
                  id={id}
                  type="text"
                  value={arr.join(', ')}
                  disabled={disabled}
                  aria-describedby={describedBy}
                  onChange={(e) => {
                    const parts = e.target.value
                      .split(',')
                      .map((s) => s.trim())
                      .filter(Boolean)
                    onChange(setAtPath(value, key, parts))
                  }}
                  className={inputClass}
                />
              </FieldShell>
            )
          }

          return (
            <FieldShell
              key={key}
              label={label}
              htmlFor={id}
              description={propSchema.description}
              errorId={errorMessage ? errorId : undefined}
              errorMessage={errorMessage}
            >
              {renderPrimitive(propSchema, current, (v) => onChange(setAtPath(value, key, v)), {
                id,
                disabled,
                errorId,
                describedBy,
              })}
            </FieldShell>
          )
        })}
      </div>
    )
  }

  return (
    <p className="text-xs text-slate-500 dark:text-neutral-400">
      Config schema must be an object with properties.
    </p>
  )
}
