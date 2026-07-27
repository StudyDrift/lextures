export type BlankType = 'numeric' | 'expression' | 'choice' | 'text'

export type BlankPolicy = 'author' | 'progressive' | 'all'

export type ChoiceOption = { id: string; text: string }

export type Blank = {
  type: BlankType
  expected?: string | number
  tolerance?: { kind: 'absolute' | 'relative'; value: number }
  acceptedAnswers?: string[]
  options?: ChoiceOption[]
  correctOptionId?: string
  unit?: string
}

export type Step = {
  id: string
  label?: string
  text: string
  blank?: Blank
  hints?: string[]
  explanation?: string
}

export type AttemptResult = 'correct' | 'incorrect' | 'needs_review'

export type StepProgress = {
  attempts?: Array<{ value: string; result: AttemptResult; at: string }>
  hintsUsed?: number
  revealed?: boolean
  draft?: string
  completedAt?: string
  startedAt?: string
}

export type CheckResult = {
  result?: AttemptResult
  feedback?: string
  attemptsRemaining?: number
  canReveal?: boolean
  nextStep?: string
  stepId?: string
  error?: string
  message?: string
}

export type HintResult = {
  hint?: string
  hintsRemaining?: number
  level?: number
  noMoreHints?: boolean
  stepId?: string
  error?: string
}

export type RevealResult = {
  explanation?: string
  expectedDisplay?: string
  revealed?: boolean
  nextStep?: string
  stepId?: string
  error?: string
  message?: string
}

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}
