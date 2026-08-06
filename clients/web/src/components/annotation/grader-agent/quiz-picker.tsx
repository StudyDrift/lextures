import { Search } from 'lucide-react'
import { useId, useMemo, useState } from 'react'

import type { CourseQuizOption } from './course-quiz-options'
import { filterQuizOptions } from './course-quiz-options'

type QuizPickerProps = {
  quizzes: CourseQuizOption[]
  value: string
  disabled?: boolean
  searchPlaceholder: string
  emptyLabel: string
  noMatchLabel: string
  moduleFallbackLabel: string
  onChange: (quizId: string) => void
}

export function QuizPicker({
  quizzes,
  value,
  disabled,
  searchPlaceholder,
  emptyLabel,
  noMatchLabel,
  moduleFallbackLabel,
  onChange,
}: QuizPickerProps) {
  const [query, setQuery] = useState('')
  const searchId = useId()
  const listId = useId()

  const visibleQuizzes = useMemo(() => filterQuizOptions(quizzes, query), [quizzes, query])

  if (quizzes.length === 0) {
    return <p className="text-sm text-fg-muted">{emptyLabel}</p>
  }

  return (
    <div className="space-y-2">
      <label htmlFor={searchId} className="relative block">
        <span className="sr-only">{searchPlaceholder}</span>
        <Search
          className="pointer-events-none absolute start-2.5 top-1/2 size-3.5 -translate-y-1/2 text-fg-subtle"
          aria-hidden
        />
        <input
          id={searchId}
          type="search"
          value={query}
          disabled={disabled}
          placeholder={searchPlaceholder}
          autoComplete="off"
          onChange={(event) => setQuery(event.target.value)}
          className="w-full rounded-lg border border-border-strong bg-surface-raised py-2 ps-8 pe-3 text-sm text-fg-default outline-none placeholder:text-fg-subtle focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:placeholder:text-neutral-500 dark:focus:border-indigo-400"
        />
      </label>

      <div
        id={listId}
        role="radiogroup"
        aria-labelledby={searchId}
        className="max-h-56 overflow-y-auto rounded-xl border border-border-default bg-slate-50/50 dark:border-border-default/40"
      >
        {visibleQuizzes.length === 0 ? (
          <p className="px-3 py-4 text-sm text-fg-muted">{noMatchLabel}</p>
        ) : (
          visibleQuizzes.map((quiz) => {
            const selected = quiz.id === value
            const moduleLabel = quiz.moduleTitle.trim() || moduleFallbackLabel
            return (
              <label
                key={quiz.id}
                className={`flex cursor-pointer items-start gap-3 border-b border-border-default px-3 py-2.5 transition-[background-color,border-color] last:border-b-0 dark:border-border-subtle ${ selected ? 'bg-indigo-50/80 dark:bg-indigo-950/30' : 'hover:bg-surface-raised dark:hover:bg-neutral-900/60' } ${disabled ? 'cursor-not-allowed opacity-60' : ''}`}
              >
                <input
                  type="radio"
                  name="create-grading-agent-quiz"
                  value={quiz.id}
                  checked={selected}
                  disabled={disabled}
                  onChange={() => onChange(quiz.id)}
                  className="mt-1 shrink-0"
                />
                <span className="min-w-0 flex-1">
                  <span className="block text-sm font-medium text-fg-default">
                    {quiz.title}
                  </span>
                  <span className="mt-0.5 block text-xs text-fg-muted">
                    {moduleLabel}
                  </span>
                </span>
              </label>
            )
          })
        )}
      </div>
    </div>
  )
}