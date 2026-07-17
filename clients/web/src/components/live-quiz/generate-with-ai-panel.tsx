import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Sparkles, X } from 'lucide-react'
import {
  cancelQuizKitGenerationJob,
  fetchQuizKitGenerationJob,
  startQuizKitGeneration,
  type LiveQuizGenSourceType,
  type LiveQuizGenerationJob,
  type LiveQuizQuestionType,
} from '../../lib/live-quiz-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { toastMutationError } from '../../lib/lms-toast'

const TYPE_OPTIONS: LiveQuizQuestionType[] = [
  'mc_single',
  'mc_multiple',
  'true_false',
  'type_answer',
  'numeric',
  'poll',
  'ordering',
]

type Props = {
  courseCode: string
  kitId: string
  open: boolean
  onClose: () => void
  onComplete: () => void
  likeQuestionId?: string | null
}

export function GenerateWithAiPanel({
  courseCode,
  kitId,
  open,
  onClose,
  onComplete,
  likeQuestionId,
}: Props) {
  const { t } = useTranslation('common')
  const { ffIqAiGeneration, aiConfigured } = usePlatformFeatures()
  const [sourceType, setSourceType] = useState<LiveQuizGenSourceType>('topic')
  const [topic, setTopic] = useState('')
  const [passage, setPassage] = useState('')
  const [contentId, setContentId] = useState('')
  const [count, setCount] = useState(5)
  const [types, setTypes] = useState<LiveQuizQuestionType[]>(['mc_single', 'true_false'])
  const [difficulty, setDifficulty] = useState<'easy' | 'medium' | 'hard'>('medium')
  const [gradeBand, setGradeBand] = useState('')
  const [language, setLanguage] = useState('en')
  const [includeExplanations, setIncludeExplanations] = useState(true)
  const [job, setJob] = useState<LiveQuizGenerationJob | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const unavailableReason = !ffIqAiGeneration
    ? t('liveQuiz.ai.unavailable.flag')
    : !aiConfigured
      ? t('liveQuiz.ai.unavailable.ai')
      : null

  useEffect(() => {
    if (!job || job.status === 'succeeded' || job.status === 'failed' || job.status === 'canceled') {
      return
    }
    const id = window.setInterval(() => {
      void (async () => {
        try {
          const next = await fetchQuizKitGenerationJob(courseCode, kitId, job.id)
          setJob(next)
          if (next.status === 'succeeded') {
            onComplete()
          }
        } catch {
          /* ignore transient poll errors */
        }
      })()
    }, 1500)
    return () => window.clearInterval(id)
  }, [job, courseCode, kitId, onComplete])

  if (!open) return null

  function toggleType(qt: LiveQuizQuestionType) {
    setTypes((prev) =>
      prev.includes(qt) ? prev.filter((x) => x !== qt) : [...prev, qt],
    )
  }

  async function handleGenerate() {
    if (unavailableReason) return
    setSubmitting(true)
    try {
      const sourceRef: Record<string, unknown> =
        sourceType === 'topic'
          ? { topic }
          : sourceType === 'passage'
            ? { passage }
            : { contentId }
      const created = await startQuizKitGeneration(courseCode, kitId, {
        sourceType,
        sourceRef,
        params: {
          count,
          types: types.length ? types : ['mc_single'],
          difficulty,
          gradeBand: gradeBand || undefined,
          language,
          includeExplanations,
          likeQuestionId: likeQuestionId || undefined,
        },
      })
      setJob(created)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleCancel() {
    if (!job) return
    try {
      const next = await cancelQuizKitGenerationJob(courseCode, kitId, job.id)
      setJob(next)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  const generating = job?.status === 'queued' || job?.status === 'running'

  return (
    <div
      className="fixed inset-0 z-40 flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="live-quiz-ai-title"
    >
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-lg bg-white p-4 shadow-lg dark:bg-neutral-900">
        <div className="mb-3 flex items-start justify-between gap-2">
          <div>
            <h2
              id="live-quiz-ai-title"
              className="flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-neutral-100"
            >
              <Sparkles className="h-5 w-5" aria-hidden />
              {t('liveQuiz.ai.title')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
              {t('liveQuiz.ai.subtitle')}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="min-h-11 min-w-11 rounded-md p-2 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
            aria-label={t('liveQuiz.ai.close')}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div
          className="mb-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100"
          role="status"
        >
          {t('liveQuiz.ai.reviewBanner')}
        </div>

        {unavailableReason ? (
          <p className="text-sm text-slate-600 dark:text-neutral-400">{unavailableReason}</p>
        ) : (
          <div className="space-y-3">
            <fieldset>
              <legend className="mb-1 text-sm font-medium text-slate-800 dark:text-neutral-200">
                {t('liveQuiz.ai.sourceLabel')}
              </legend>
              <div className="flex flex-wrap gap-2">
                {(
                  [
                    ['topic', 'liveQuiz.ai.source.topic'],
                    ['passage', 'liveQuiz.ai.source.passage'],
                    ['course_content_ref', 'liveQuiz.ai.source.courseContent'],
                  ] as const
                ).map(([value, key]) => (
                  <label
                    key={value}
                    className="inline-flex min-h-11 items-center gap-2 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                  >
                    <input
                      type="radio"
                      name="ai-source"
                      checked={sourceType === value}
                      onChange={() => setSourceType(value)}
                      disabled={generating}
                    />
                    {t(key)}
                  </label>
                ))}
              </div>
            </fieldset>

            {sourceType === 'topic' ? (
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.topicLabel')}</span>
                <textarea
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                  rows={3}
                  disabled={generating}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                  placeholder={t('liveQuiz.ai.topicPlaceholder')}
                />
              </label>
            ) : null}
            {sourceType === 'passage' ? (
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.passageLabel')}</span>
                <textarea
                  value={passage}
                  onChange={(e) => setPassage(e.target.value)}
                  rows={6}
                  disabled={generating}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                  placeholder={t('liveQuiz.ai.passagePlaceholder')}
                />
              </label>
            ) : null}
            {sourceType === 'course_content_ref' ? (
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.contentIdLabel')}</span>
                <input
                  value={contentId}
                  onChange={(e) => setContentId(e.target.value)}
                  disabled={generating}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                  placeholder={t('liveQuiz.ai.contentIdPlaceholder')}
                />
              </label>
            ) : null}

            <div className="grid grid-cols-2 gap-3">
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.countLabel')}</span>
                <input
                  type="number"
                  min={1}
                  max={25}
                  value={count}
                  onChange={(e) => setCount(Number(e.target.value) || 5)}
                  disabled={generating}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                />
              </label>
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.difficultyLabel')}</span>
                <select
                  value={difficulty}
                  onChange={(e) => setDifficulty(e.target.value as 'easy' | 'medium' | 'hard')}
                  disabled={generating}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                >
                  <option value="easy">{t('liveQuiz.ai.difficulty.easy')}</option>
                  <option value="medium">{t('liveQuiz.ai.difficulty.medium')}</option>
                  <option value="hard">{t('liveQuiz.ai.difficulty.hard')}</option>
                </select>
              </label>
            </div>

            <label className="block text-sm">
              <span className="mb-1 block font-medium">{t('liveQuiz.ai.typesLabel')}</span>
              <div className="flex flex-wrap gap-2">
                {TYPE_OPTIONS.map((qt) => (
                  <label
                    key={qt}
                    className="inline-flex min-h-11 items-center gap-2 rounded-md border border-slate-300 px-2 py-1 text-xs dark:border-neutral-600"
                  >
                    <input
                      type="checkbox"
                      checked={types.includes(qt)}
                      onChange={() => toggleType(qt)}
                      disabled={generating}
                    />
                    {qt}
                  </label>
                ))}
              </div>
            </label>

            <div className="grid grid-cols-2 gap-3">
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.gradeLabel')}</span>
                <input
                  value={gradeBand}
                  onChange={(e) => setGradeBand(e.target.value)}
                  disabled={generating}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                  placeholder={t('liveQuiz.ai.gradePlaceholder')}
                />
              </label>
              <label className="block text-sm">
                <span className="mb-1 block font-medium">{t('liveQuiz.ai.languageLabel')}</span>
                <input
                  value={language}
                  onChange={(e) => setLanguage(e.target.value)}
                  disabled={generating}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                />
              </label>
            </div>

            <label className="inline-flex min-h-11 items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={includeExplanations}
                onChange={(e) => setIncludeExplanations(e.target.checked)}
                disabled={generating}
              />
              {t('liveQuiz.ai.includeExplanations')}
            </label>

            {job ? (
              <div className="rounded-md border border-slate-200 p-3 text-sm dark:border-neutral-700" aria-live="polite">
                <p>
                  {t('liveQuiz.ai.jobStatus', { status: job.status, progress: job.progress })}
                </p>
                {job.error ? <p className="mt-1 text-red-600 dark:text-red-400">{job.error}</p> : null}
                {job.status === 'succeeded' && job.resultSummary ? (
                  <p className="mt-1 text-slate-600 dark:text-neutral-400">
                    {t('liveQuiz.ai.jobSummary', {
                      inserted: job.resultSummary.inserted ?? 0,
                      dropped: job.resultSummary.dropped ?? 0,
                    })}
                  </p>
                ) : null}
              </div>
            ) : null}

            <div className="flex flex-wrap justify-end gap-2 pt-2">
              {generating ? (
                <button
                  type="button"
                  onClick={() => {
                    void handleCancel()
                  }}
                  className="min-h-11 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                >
                  {t('liveQuiz.ai.cancel')}
                </button>
              ) : null}
              <button
                type="button"
                onClick={() => {
                  void handleGenerate()
                }}
                disabled={!!unavailableReason || submitting || generating}
                className="inline-flex min-h-11 items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
              >
                {submitting || generating ? (
                  <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                ) : (
                  <Sparkles className="h-4 w-4" aria-hidden />
                )}
                {generating ? t('liveQuiz.ai.generating') : t('liveQuiz.ai.generate')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
