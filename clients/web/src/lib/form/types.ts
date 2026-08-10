import type { z } from 'zod'

/** Dot/bracket path into form values, e.g. "firstName" or "sections[0].code". */
export type FieldPath = string

export type FieldViolation = {
  path: FieldPath
  code: string
  message: string
  params?: Record<string, unknown>
}

/** UX.6 §9 validation error envelope (HTTP 422). */
export type ValidationErrorResponse = {
  error: 'validation_failed'
  message: string
  fields: FieldViolation[]
}

export type FormFieldState = {
  error: string | null
  /** Whether the field has been blurred at least once. */
  touched: boolean
  /** Whether an error is currently shown (timing contract). */
  showError: boolean
  /** Async validation pending. */
  busy: boolean
}

export type FormSubmitHelpers<T = Record<string, unknown>> = {
  setServerErrors: (fields: FieldViolation[]) => void
  setFormError: (message: string | null) => void
  reset: (values?: T) => void
}

export type FormSubmitHandler<T> = (
  values: T,
  helpers: FormSubmitHelpers<T>,
) => void | Promise<void>

export type UseFormOptions<T extends Record<string, unknown>> = {
  /** Zod schema — single source of client validation (FR-8). */
  schema: z.ZodType<T>
  defaultValues: T
  /** Stable form id for telemetry (no values). */
  formId?: string
  /** Human labels keyed by field path — used by the error summary. */
  labels?: Partial<Record<FieldPath, string>>
  /**
   * Resolve a machine validation code / zod issue to user-facing copy.
   * Default uses `common.validation.*` i18n keys when a translator is provided.
   */
  formatError?: (issue: {
    path: FieldPath
    code: string
    message: string
    params?: Record<string, unknown>
  }) => string
  onSubmit: FormSubmitHandler<T>
}

export type FormControlProps = {
  id: string
  name: string
  value: string | number | readonly string[] | undefined
  invalid: boolean
  'aria-invalid'?: boolean
  onChange: (e: { target: { value: string; type?: string; checked?: boolean; name?: string } }) => void
  onBlur: () => void
}

export type FormApi<T extends Record<string, unknown>> = {
  formId: string
  /** Current values (do not subscribe — use getValues / useFormField). */
  getValues: () => T
  setValue: <K extends keyof T & string>(name: K, value: T[K], opts?: { validate?: boolean }) => void
  setValues: (values: Partial<T>) => void
  reset: (values?: T) => void
  /** Field-level props for controlled native/library inputs. */
  register: (name: FieldPath) => FormControlProps
  getFieldState: (name: FieldPath) => FormFieldState
  getFieldError: (name: FieldPath) => string | null
  /** True when values differ from the last reset/default baseline. */
  isDirty: () => boolean
  isSubmitting: () => boolean
  submitAttempted: () => boolean
  formError: () => string | null
  setFormError: (message: string | null) => void
  setServerErrors: (fields: FieldViolation[]) => void
  /** Entries for ErrorSummary (only fields currently showing errors). */
  summaryErrors: () => { id: string; path: FieldPath; label: string; message: string }[]
  handleSubmit: (e?: { preventDefault?: () => void }) => Promise<void>
  /** Subscribe to store changes. path=null → any change; path=string → that field + form flags. */
  subscribe: (path: FieldPath | null, cb: () => void) => () => void
  /** Control id for a field path (stable for the form instance). */
  fieldId: (name: FieldPath) => string
  validateField: (name: FieldPath) => void
}
