import { createContext, useContext } from 'react'

/**
 * UX.6 — shared field→control wiring for id / describedby / invalid / required.
 * Consumed by Input, Textarea, Select, and other form controls inside `<Field>`.
 */
export type FieldContextValue = {
  /** Control id (also used by the visible label's htmlFor). */
  id: string
  /** Space-separated ids for description + error (aria-describedby). */
  describedBy?: string
  invalid: boolean
  required: boolean
  /** Async validation pending (aria-busy). */
  busy?: boolean
}

export const FieldContext = createContext<FieldContextValue | null>(null)

export function useFieldContext(): FieldContextValue | null {
  return useContext(FieldContext)
}

/** Merge an explicit prop with a context-provided aria-describedby list. */
export function mergeDescribedBy(
  explicit: string | undefined,
  fromContext: string | undefined,
): string | undefined {
  const parts = [explicit, fromContext]
    .filter((s): s is string => typeof s === 'string' && s.trim().length > 0)
    .flatMap((s) => s.split(/\s+/))
  const unique = [...new Set(parts)]
  return unique.length ? unique.join(' ') : undefined
}
