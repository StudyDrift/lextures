import type { z } from 'zod'
import {
  formatValidationIssue,
  zodIssueToValidationIssue,
  type TranslateFn,
} from './validation-messages'
import type {
  FieldPath,
  FieldViolation,
  FormApi,
  FormFieldState,
  UseFormOptions,
} from './types'

type Listener = () => void

function getAtPath(obj: unknown, path: FieldPath): unknown {
  if (!path) return obj
  const parts = path.replace(/\[(\d+)\]/g, '.$1').split('.').filter(Boolean)
  let cur: unknown = obj
  for (const p of parts) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[p]
  }
  return cur
}

function setAtPath<T extends Record<string, unknown>>(obj: T, path: FieldPath, value: unknown): T {
  const parts = path.replace(/\[(\d+)\]/g, '.$1').split('.').filter(Boolean)
  if (parts.length === 0) return obj
  const next = { ...obj } as Record<string, unknown>
  let cursor: Record<string, unknown> = next
  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i]!
    const child = cursor[key]
    const clone =
      child != null && typeof child === 'object'
        ? Array.isArray(child)
          ? [...child]
          : { ...(child as Record<string, unknown>) }
        : {}
    cursor[key] = clone
    cursor = clone as Record<string, unknown>
  }
  cursor[parts[parts.length - 1]!] = value
  return next as T
}

function stableStringify(v: unknown): string {
  try {
    return JSON.stringify(v)
  } catch {
    return String(v)
  }
}

let formSeq = 0

export function createFormStore<T extends Record<string, unknown>>(
  options: UseFormOptions<T> & { t?: TranslateFn },
): FormApi<T> {
  const instanceId = `form-${++formSeq}`
  const formId = options.formId ?? instanceId
  const labels = options.labels ?? {}
  const formatError =
    options.formatError ??
    ((issue) =>
      formatValidationIssue(issue, options.t, labels[issue.path]))

  let values: T = { ...options.defaultValues }
  let baseline = stableStringify(values)
  let errors: Record<string, string> = {}
  let touched: Record<string, boolean> = {}
  let showError: Record<string, boolean> = {}
  let busy: Record<string, boolean> = {}
  let submitting = false
  let attempted = false
  let formError: string | null = null

  const listeners = new Map<string | null, Set<Listener>>()

  function notify(path: FieldPath | null) {
    if (path !== null) {
      listeners.get(path)?.forEach((cb) => cb())
    }
    listeners.get(null)?.forEach((cb) => cb())
  }

  function subscribe(path: FieldPath | null, cb: Listener) {
    let set = listeners.get(path)
    if (!set) {
      set = new Set()
      listeners.set(path, set)
    }
    set.add(cb)
    return () => {
      set!.delete(cb)
      if (set!.size === 0) listeners.delete(path)
    }
  }

  function fieldId(name: FieldPath): string {
    const safe = name.replace(/[^\w-]+/g, '-')
    return `${formId}-field-${safe}`
  }

  function runSchema(): { success: true; data: T } | { success: false; issues: ReturnType<typeof zodIssueToValidationIssue>[] } {
    const result = options.schema.safeParse(values)
    if (result.success) {
      return { success: true, data: result.data }
    }
    const issues = result.error.issues.map((iss) =>
      zodIssueToValidationIssue(iss as Parameters<typeof zodIssueToValidationIssue>[0]),
    )
    return { success: false, issues }
  }

  function validateOne(name: FieldPath): string | null {
    const result = runSchema()
    if (result.success) return null
    const hit = result.issues.find((i) => i.path === name || i.path.startsWith(`${name}.`))
    if (!hit) return null
    return formatError(hit)
  }

  function applyIssues(issues: ReturnType<typeof zodIssueToValidationIssue>[], showAll: boolean) {
    const next: Record<string, string> = {}
    for (const issue of issues) {
      if (!next[issue.path]) {
        next[issue.path] = formatError(issue)
      }
    }
    errors = next
    if (showAll) {
      const show: Record<string, boolean> = {}
      for (const path of Object.keys(next)) show[path] = true
      showError = show
    }
  }

  function validateField(name: FieldPath) {
    const msg = validateOne(name)
    const prev = errors[name] ?? null
    if (msg) {
      errors = { ...errors, [name]: msg }
      if (showError[name] || touched[name] || attempted) {
        showError = { ...showError, [name]: true }
      }
    } else {
      if (name in errors) {
        const { [name]: _, ...rest } = errors
        errors = rest
      }
      if (name in showError) {
        const { [name]: _, ...rest } = showError
        showError = rest
      }
    }
    if (prev !== (errors[name] ?? null) || showError[name]) {
      notify(name)
      notify(null)
    }
  }

  function setValue<K extends keyof T & string>(
    name: K,
    value: T[K],
    opts?: { validate?: boolean },
  ) {
    values = setAtPath(values, name, value)
    const wasShowing = Boolean(showError[name])
    if (opts?.validate !== false && (wasShowing || attempted)) {
      validateField(name)
    } else {
      notify(name)
      notify(null)
    }
  }

  function setValues(partial: Partial<T>) {
    values = { ...values, ...partial }
    notify(null)
    for (const key of Object.keys(partial)) notify(key)
  }

  function reset(next?: T) {
    values = { ...(next ?? options.defaultValues) }
    baseline = stableStringify(values)
    errors = {}
    touched = {}
    showError = {}
    busy = {}
    submitting = false
    attempted = false
    formError = null
    notify(null)
  }

  function setServerErrors(fields: FieldViolation[]) {
    const nextErrors: Record<string, string> = { ...errors }
    const nextShow: Record<string, boolean> = { ...showError }
    for (const f of fields) {
      nextErrors[f.path] = formatError({
        path: f.path,
        code: f.code,
        message: f.message,
        params: f.params,
      })
      nextShow[f.path] = true
    }
    errors = nextErrors
    showError = nextShow
    attempted = true
    notify(null)
    for (const f of fields) notify(f.path)
  }

  function setFormError(message: string | null) {
    formError = message
    notify(null)
  }

  function getFieldState(name: FieldPath): FormFieldState {
    return {
      error: showError[name] ? (errors[name] ?? null) : null,
      touched: Boolean(touched[name]),
      showError: Boolean(showError[name]),
      busy: Boolean(busy[name]),
    }
  }

  function register(name: FieldPath) {
    const id = fieldId(name)
    const raw = getAtPath(values, name)
    const value =
      raw === undefined || raw === null
        ? ''
        : typeof raw === 'string' || typeof raw === 'number'
          ? raw
          : String(raw)

    return {
      id,
      name,
      value,
      invalid: Boolean(showError[name] && errors[name]),
      'aria-invalid': showError[name] && errors[name] ? true : undefined,
      onChange: (e: {
        target: { value: string; type?: string; checked?: boolean; name?: string }
      }) => {
        const t = e.target
        let next: unknown = t.value
        if (t.type === 'checkbox') next = Boolean(t.checked)
        setValue(name as keyof T & string, next as T[keyof T & string], {
          validate: Boolean(showError[name]),
        })
      },
      onBlur: () => {
        touched = { ...touched, [name]: true }
        // FR-5: validate on blur once touched (blur itself marks touched).
        const msg = validateOne(name)
        if (msg) {
          errors = { ...errors, [name]: msg }
          showError = { ...showError, [name]: true }
        } else if (name in errors) {
          const { [name]: _, ...rest } = errors
          errors = rest
          const { [name]: __, ...showRest } = showError
          showError = showRest
        }
        notify(name)
        notify(null)
      },
    }
  }

  function summaryErrors() {
    return Object.keys(showError)
      .filter((path) => showError[path] && errors[path])
      .map((path) => ({
        id: fieldId(path),
        path,
        label: labels[path] ?? path,
        message: errors[path]!,
      }))
  }

  async function handleSubmit(e?: { preventDefault?: () => void }) {
    e?.preventDefault?.()
    attempted = true
    formError = null

    const result = runSchema()
    if (!result.success) {
      applyIssues(result.issues, true)
      notify(null)
      for (const issue of result.issues) notify(issue.path)
      emitFormValidationFailed(formId, result.issues.map((i) => i.path))
      return
    }

    values = result.data
    errors = {}
    showError = {}
    submitting = true
    notify(null)

    const helpers = {
      setServerErrors,
      setFormError,
      reset,
    }

    try {
      await options.onSubmit(result.data, helpers)
      // Successful submit without explicit reset: clear dirty baseline if values unchanged.
      baseline = stableStringify(values)
      attempted = false
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Could not save. Try again.'
      setFormError(msg)
    } finally {
      submitting = false
      notify(null)
    }
  }

  const api: FormApi<T> = {
    formId,
    getValues: () => values,
    setValue,
    setValues,
    reset,
    register,
    getFieldState,
    getFieldError: (name) => getFieldState(name).error,
    isDirty: () => stableStringify(values) !== baseline,
    isSubmitting: () => submitting,
    submitAttempted: () => attempted,
    formError: () => formError,
    setFormError,
    setServerErrors,
    summaryErrors,
    handleSubmit,
    subscribe,
    fieldId,
    validateField,
  }

  return api
}

/** Lightweight client telemetry bus — no PII / no field values (UX.6 observability). */
type FormTelemetryListener = (event: {
  event: 'form_validation_failed'
  formId: string
  fields: string[]
}) => void

const telemetryListeners = new Set<FormTelemetryListener>()

export function subscribeFormTelemetry(cb: FormTelemetryListener): () => void {
  telemetryListeners.add(cb)
  return () => telemetryListeners.delete(cb)
}

function emitFormValidationFailed(formId: string, fields: string[]) {
  const payload = { event: 'form_validation_failed' as const, formId, fields }
  for (const cb of telemetryListeners) {
    try {
      cb(payload)
    } catch {
      /* never block UI */
    }
  }
}

// re-export schema type helper for callers
export type { z }
