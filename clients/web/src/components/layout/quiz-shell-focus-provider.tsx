import { useCallback, useMemo, useState } from 'react'
import { QuizShellFocusContext, type QuizShellFocusMode, type QuizShellFocusProviderProps } from './quiz-shell-focus-context'

function quizShellFocusEqual(a: QuizShellFocusMode | null, b: QuizShellFocusMode | null): boolean {
  if (a === b) return true
  if (a === null || b === null) return false
  return (
    a.quizTitle === b.quizTitle &&
    a.timeRemainingLabel === b.timeRemainingLabel &&
    a.timeUrgent === b.timeUrgent &&
    a.questionProgress === b.questionProgress &&
    a.saveStatusText === b.saveStatusText &&
    a.lockdownAccent === b.lockdownAccent &&
    a.flaggedForCurrent === b.flaggedForCurrent &&
    a.onToggleFlagForReview === b.onToggleFlagForReview
  )
}

export function QuizShellFocusProvider({ children }: QuizShellFocusProviderProps) {
  const [focus, setFocus] = useState<QuizShellFocusMode | null>(null)
  const setQuizShellFocus = useCallback((next: QuizShellFocusMode | null) => {
    setFocus((prev) => (quizShellFocusEqual(prev, next) ? prev : next))
  }, [])
  const value = useMemo(() => ({ focus, setQuizShellFocus }), [focus, setQuizShellFocus])
  return <QuizShellFocusContext.Provider value={value}>{children}</QuizShellFocusContext.Provider>
}
