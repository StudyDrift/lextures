export type Option = { id: string; text: string; correct?: boolean; feedback?: string }
export type Question = {
  id: string
  type: string
  prompt: string
  options?: Option[]
  acceptedAnswers?: string[]
  correctValue?: number
  tolerance?: { kind: 'absolute' | 'relative'; value: number }
  unit?: string
  explanation?: string
  points?: number
  caseSensitive?: boolean
  normalizePunctuation?: boolean
}

export const QUESTION_TYPES = ['single', 'multi', 'true_false', 'short_text', 'numeric'] as const
export const REVEAL_OPTIONS = ['first_attempt', 'last_attempt', 'never'] as const

export const fieldClass =
  'w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm focus:border-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-200 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-neutral-500 dark:focus:ring-neutral-700'
export const labelClass = 'block text-xs font-medium text-slate-700 dark:text-neutral-300'
export const sectionClass =
  'space-y-4 rounded-lg border border-slate-200 bg-slate-50/50 p-4 dark:border-neutral-700 dark:bg-neutral-900/40'

export function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

export function asQuestions(value: Record<string, unknown>): Question[] {
  if (!Array.isArray(value.questions)) return []
  return value.questions as Question[]
}

export function defaultOptions(type: string): Option[] {
  if (type === 'true_false') {
    return [
      { id: 'true', text: 'True', correct: true },
      { id: 'false', text: 'False', correct: false },
    ]
  }
  return [
    { id: newId('opt'), text: '', correct: true },
    { id: newId('opt'), text: '', correct: false },
  ]
}
