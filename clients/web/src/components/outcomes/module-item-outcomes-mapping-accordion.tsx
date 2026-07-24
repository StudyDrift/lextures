import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { usePermissions } from '../../context/use-permissions'
import {
  courseItemCreatePermission,
  fetchCourseOutcomes,
  type CourseOutcome,
} from '../../lib/courses-api'
import { countOutcomeLinksForItem } from './outcome-links-helpers'
import { OutcomeLinksEditor } from './outcome-links-editor'

type ModuleItemOutcomesMappingAccordionProps = {
  courseCode: string
  itemId: string
  mode: 'assignment' | 'quiz'
  disabled?: boolean
  /** Saved quiz questions (ids) — used when mapping a single question. */
  quizQuestions?: { id: string; prompt: string }[]
  /** Optional: report total link count to parent (e.g. accordion badge). */
  onLinkCountChange?: (count: number) => void
}

export function ModuleItemOutcomesMappingAccordion({
  courseCode,
  itemId,
  mode,
  disabled,
  quizQuestions = [],
  onLinkCountChange,
}: ModuleItemOutcomesMappingAccordionProps) {
  const { allows, loading: permLoading } = usePermissions()
  const canMap = !permLoading && allows(courseItemCreatePermission(courseCode))

  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [outcomes, setOutcomes] = useState<CourseOutcome[]>([])
  const [questionId, setQuestionId] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setLoadError(null)
    try {
      const data = await fetchCourseOutcomes(courseCode)
      setOutcomes(data.outcomes)
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Could not load outcomes.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    setQuestionId((cur) => {
      if (quizQuestions.some((q) => q.id === cur)) return cur
      return quizQuestions[0]?.id ?? ''
    })
  }, [quizQuestions])

  const linkCount = useMemo(
    () => countOutcomeLinksForItem(outcomes, itemId, mode),
    [outcomes, itemId, mode],
  )

  useEffect(() => {
    onLinkCountChange?.(linkCount)
  }, [linkCount, onLinkCountChange])

  const settingsOutcomesUrl = `/courses/${encodeURIComponent(courseCode)}/settings/outcomes`

  const selectedQuestionLabel = useMemo(() => {
    const q = quizQuestions.find((it) => it.id === questionId)
    if (!q) return ''
    const prompt = (q.prompt || q.id).replace(/\s+/g, ' ')
    return prompt.length > 72 ? `${prompt.slice(0, 72)}…` : prompt
  }, [quizQuestions, questionId])

  return (
    <div className="space-y-3 pt-1">
      <p className="text-[11px] leading-snug text-slate-400 dark:text-neutral-500">
        Link this {mode === 'assignment' ? 'assignment' : 'quiz'} to course learning outcomes with
        measurement and intensity.{' '}
        <Link
          to={settingsOutcomesUrl}
          className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
        >
          Open full outcomes page
        </Link>
        .
      </p>

      {!canMap ? (
        <p className="text-[11px] text-slate-500 dark:text-neutral-500">
          You need course edit permission to change outcome mappings.
        </p>
      ) : null}

      {loadError ? (
        <p className="text-[11px] text-rose-600 dark:text-rose-400">{loadError}</p>
      ) : null}

      {loading ? (
        <p className="flex items-center gap-1.5 text-[11px] text-slate-500 dark:text-neutral-500">
          <Loader2 className="h-3.5 w-3.5 motion-safe:animate-spin" aria-hidden />
          Loading…
        </p>
      ) : null}

      {!loading && canMap && mode === 'assignment' ? (
        <OutcomeLinksEditor
          courseCode={courseCode}
          itemId={itemId}
          targetKind="assignment"
          disabled={disabled}
          variant="settings"
          outcomes={outcomes}
          outcomesLoading={false}
          outcomesError={null}
          onOutcomesChange={load}
          hideHeaderHint
          emptyLabel="No outcome links for this assignment yet."
        />
      ) : null}

      {!loading && canMap && mode === 'quiz' ? (
        <div className="space-y-4">
          <section className="space-y-2">
            <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              Whole quiz
            </h4>
            <OutcomeLinksEditor
              courseCode={courseCode}
              itemId={itemId}
              targetKind="quiz"
              disabled={disabled}
              variant="settings"
              outcomes={outcomes}
              outcomesLoading={false}
              outcomesError={null}
              onOutcomesChange={load}
              hideHeaderHint
              emptyLabel="No whole-quiz outcome links yet."
            />
          </section>

          <section className="space-y-2 border-t border-slate-100 pt-3 dark:border-neutral-800/80">
            <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              By question
            </h4>
            {quizQuestions.length === 0 ? (
              <p className="text-[11px] text-slate-500 dark:text-neutral-500">
                Add questions in the quiz editor to map individual questions.
              </p>
            ) : (
              <>
                <div>
                  <label
                    htmlFor={`outcomes-question-pick-${itemId}`}
                    className="mb-0.5 block text-[11px] font-medium text-slate-500 dark:text-neutral-400"
                  >
                    Question
                  </label>
                  <select
                    id={`outcomes-question-pick-${itemId}`}
                    value={questionId}
                    onChange={(e) => setQuestionId(e.target.value)}
                    disabled={disabled}
                    className="w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-indigo-500 dark:focus:ring-indigo-500"
                  >
                    {quizQuestions.map((q) => (
                      <option key={q.id} value={q.id}>
                        {(q.prompt || q.id).replace(/\s+/g, ' ').slice(0, 72)}
                        {(q.prompt || '').length > 72 ? '…' : ''}
                      </option>
                    ))}
                  </select>
                  {selectedQuestionLabel ? (
                    <p className="mt-1 text-[11px] text-slate-400 dark:text-neutral-500">
                      Mapping for: {selectedQuestionLabel}
                    </p>
                  ) : null}
                </div>
                {questionId ? (
                  <OutcomeLinksEditor
                    courseCode={courseCode}
                    itemId={itemId}
                    targetKind="quiz_question"
                    quizQuestionId={questionId}
                    disabled={disabled}
                    variant="settings"
                    outcomes={outcomes}
                    outcomesLoading={false}
                    outcomesError={null}
                    onOutcomesChange={load}
                    hideHeaderHint
                    emptyLabel="No outcome links for this question yet."
                  />
                ) : null}
              </>
            )}
          </section>
        </div>
      ) : null}
    </div>
  )
}
