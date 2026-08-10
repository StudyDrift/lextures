import { useCallback, useMemo, useRef, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { createFormStore } from './form-store'
import type { FieldPath, FormApi, FormFieldState, UseFormOptions } from './types'

/**
 * UX.6 form hook: zod schema + per-field subscriptions + validation timing contract.
 *
 * Prefer `useFormField(form, name)` inside field rows so keystrokes only re-render
 * that field (NFR performance).
 */
export function useForm<T extends Record<string, unknown>>(
  options: UseFormOptions<T>,
): FormApi<T> {
  const { t } = useTranslation('common')
  const optionsRef = useRef(options)
  optionsRef.current = options

  const store = useMemo(
    () =>
      createFormStore<T>({
        ...options,
        t: (key, opts) => t(key, opts as Record<string, unknown>),
        onSubmit: (values, helpers) => optionsRef.current.onSubmit(values, helpers),
      }),
    // formId + schema identity: recreate when the form identity changes.
    // defaultValues applied via reset() when data loads.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- stable form instance
    [options.formId, options.schema],
  )

  return store
}

/** Subscribe to one field (+ form-level flags that affect it). */
export function useFormField<T extends Record<string, unknown>>(
  form: FormApi<T>,
  name: FieldPath,
): FormFieldState & {
  id: string
  value: unknown
  controlProps: ReturnType<FormApi<T>['register']>
  setValue: (value: unknown) => void
} {
  const subscribe = useCallback(
    (onStoreChange: () => void) => form.subscribe(name, onStoreChange),
    [form, name],
  )
  const getSnapshot = useCallback(() => {
    const state = form.getFieldState(name)
    const value = form.getValues()
    const at = name.split('.').reduce<unknown>((acc, key) => {
      if (acc == null || typeof acc !== 'object') return undefined
      return (acc as Record<string, unknown>)[key]
    }, value)
    // Snapshot token so useSyncExternalStore sees changes.
    return {
      state,
      value: at,
      token: `${state.error}|${state.touched}|${state.showError}|${state.busy}|${String(at)}`,
    }
  }, [form, name])

  const snap = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)
  const controlProps = form.register(name)

  return {
    ...snap.state,
    id: form.fieldId(name),
    value: snap.value,
    controlProps,
    setValue: (v) => form.setValue(name as keyof T & string, v as T[keyof T & string]),
  }
}

/** Subscribe to form-level state (dirty, submitting, summary, formError). */
export function useFormState<T extends Record<string, unknown>>(form: FormApi<T>) {
  const subscribe = useCallback(
    (onStoreChange: () => void) => form.subscribe(null, onStoreChange),
    [form],
  )
  const getSnapshot = useCallback(() => {
    const summary = form.summaryErrors()
    return {
      isDirty: form.isDirty(),
      isSubmitting: form.isSubmitting(),
      submitAttempted: form.submitAttempted(),
      formError: form.formError(),
      summary,
      token: `${form.isDirty()}|${form.isSubmitting()}|${form.submitAttempted()}|${form.formError()}|${summary.map((s) => s.id + s.message).join(';')}`,
    }
  }, [form])

  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot)
}
