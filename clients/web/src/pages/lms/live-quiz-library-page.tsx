import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Library } from 'lucide-react'
import {
  importQuizLibraryKit,
  previewQuizLibraryKit,
  searchQuizLibrary,
  type LiveQuizQuestion,
  type QuizKit,
} from '../../lib/live-quiz-api'
import { toastMutationError } from '../../lib/lms-toast'
import { LmsPage } from './lms-page'

export default function LiveQuizLibraryPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { courseCode: rawCode } = useParams<{ courseCode: string }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''

  const [q, setQ] = useState('')
  const [subject, setSubject] = useState('')
  const [grade, setGrade] = useState('')
  const [lang, setLang] = useState('')
  const [tag, setTag] = useState('')
  const [kits, setKits] = useState<QuizKit[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [preview, setPreview] = useState<{ kit: QuizKit; questions: LiveQuizQuestion[] } | null>(
    null,
  )
  const [importing, setImporting] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await searchQuizLibrary({
        q: q.trim() || undefined,
        subject: subject.trim() || undefined,
        grade: grade.trim() || undefined,
        lang: lang.trim() || undefined,
        tag: tag.trim() || undefined,
      })
      setKits(result.kits)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('liveQuiz.library.errorLoad'))
    } finally {
      setLoading(false)
    }
  }, [q, subject, grade, lang, tag, t])

  useEffect(() => {
    void load()
  }, [load])

  async function openPreview(kitId: string) {
    try {
      const data = await previewQuizLibraryKit(kitId)
      setPreview(data)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleImport(kitId: string) {
    if (!courseCode) return
    setImporting(true)
    try {
      const created = await importQuizLibraryKit(kitId, courseCode)
      setPreview(null)
      void navigate(
        `/courses/${encodeURIComponent(courseCode)}/live-quizzes/${encodeURIComponent(created.id)}`,
      )
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setImporting(false)
    }
  }

  return (
    <LmsPage title={t('liveQuiz.library.title')}>
      <div className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-fg-muted">{t('liveQuiz.library.subtitle')}</p>
          {courseCode ? (
            <Link
              to={`/courses/${encodeURIComponent(courseCode)}/live-quizzes`}
              className="text-sm font-medium text-accent-fg hover:underline dark:text-indigo-400"
            >
              {t('liveQuiz.kit.backToGallery')}
            </Link>
          ) : null}
        </div>

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5" role="search">
          <label className="text-sm lg:col-span-2">
            <span className="sr-only">{t('liveQuiz.library.search')}</span>
            <input
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder={t('liveQuiz.library.searchPlaceholder')}
              className="min-h-11 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            />
          </label>
          <label className="text-sm">
            <span className="sr-only">{t('liveQuiz.library.subject')}</span>
            <input
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder={t('liveQuiz.library.subject')}
              className="min-h-11 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            />
          </label>
          <label className="text-sm">
            <span className="sr-only">{t('liveQuiz.library.grade')}</span>
            <input
              value={grade}
              onChange={(e) => setGrade(e.target.value)}
              placeholder={t('liveQuiz.library.grade')}
              className="min-h-11 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            />
          </label>
          <label className="text-sm">
            <span className="sr-only">{t('liveQuiz.library.language')}</span>
            <input
              value={lang}
              onChange={(e) => setLang(e.target.value)}
              placeholder={t('liveQuiz.library.language')}
              className="min-h-11 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            />
          </label>
        </div>
        <label className="block max-w-xs text-sm">
          <span className="sr-only">{t('liveQuiz.library.tag')}</span>
          <input
            value={tag}
            onChange={(e) => setTag(e.target.value)}
            placeholder={t('liveQuiz.library.tag')}
            className="min-h-11 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
          />
        </label>

        {loading ? (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" aria-busy="true">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="h-32 motion-safe:animate-pulse rounded-lg border border-border-default bg-surface-sunken dark:border-border-default dark:bg-surface-overlay"
              />
            ))}
          </div>
        ) : error ? (
          <div className="rounded-md bg-red-50 p-4 text-sm text-danger-fg dark:bg-red-950/30 dark:text-red-400">
            {error}
          </div>
        ) : kits.length === 0 ? (
          <div className="flex flex-col items-center gap-3 py-16 text-center">
            <Library className="h-10 w-10 text-fg-subtle" aria-hidden />
            <p className="text-sm text-fg-muted">{t('liveQuiz.library.empty')}</p>
          </div>
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {kits.map((kit) => (
              <li key={kit.id}>
                <button
                  type="button"
                  onClick={() => void openPreview(kit.id)}
                  className="block w-full rounded-lg border border-border-default p-4 text-start transition-[background-color,color,border-color] hover:border-indigo-300 hover:bg-indigo-50/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-border-default dark:hover:border-indigo-700"
                >
                  <h3 className="font-medium text-fg-default">{kit.title}</h3>
                  {kit.description ? (
                    <p className="mt-1 line-clamp-2 text-sm text-fg-muted">
                      {kit.description}
                    </p>
                  ) : null}
                  <p className="mt-2 text-xs text-fg-muted">
                    {[kit.subject, kit.gradeBand, kit.language].filter(Boolean).join(' · ') ||
                      t('liveQuiz.gallery.questionCount', { count: kit.questionCount })}
                  </p>
                  {kit.catalogStatus === 'pending' ? (
                    <span className="mt-2 inline-block rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-800">
                      {t('liveQuiz.library.pending')}
                    </span>
                  ) : null}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {preview ? (
        <div
          className="fixed inset-0 z-50 flex justify-end bg-black/40"
          role="dialog"
          aria-modal="true"
          aria-labelledby="library-preview-title"
        >
          <div className="flex h-full w-full max-w-md flex-col bg-surface-raised shadow-xl dark:bg-surface-raised">
            <div className="flex items-start justify-between gap-3 border-b border-border-default p-4 dark:border-border-default">
              <div>
                <h2
                  id="library-preview-title"
                  className="text-lg font-semibold text-fg-default"
                >
                  {preview.kit.title}
                </h2>
                <p className="mt-1 text-sm text-fg-muted">
                  {t('liveQuiz.library.previewReadOnly')}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setPreview(null)}
                className="min-h-11 rounded-md px-3 text-sm hover:bg-surface-sunken dark:hover:bg-surface-overlay"
              >
                {t('dialogs.cancel')}
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-4">
              <ol className="space-y-3">
                {preview.questions.map((question, idx) => (
                  <li key={question.id} className="rounded-md border border-border-default p-3 text-sm dark:border-border-default">
                    <p className="font-medium">
                      {idx + 1}. {question.prompt}
                    </p>
                    {question.promptMediaAlt ? (
                      <p className="mt-1 text-xs text-fg-muted">
                        {t('liveQuiz.library.mediaAlt')}: {question.promptMediaAlt}
                      </p>
                    ) : null}
                  </li>
                ))}
              </ol>
            </div>
            {courseCode ? (
              <div className="border-t border-border-default p-4 dark:border-border-default">
                <button
                  type="button"
                  disabled={importing}
                  onClick={() => void handleImport(preview.kit.id)}
                  className="min-h-11 w-full rounded-md bg-accent-solid px-3 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-50"
                >
                  {importing ? t('common.loading') : t('liveQuiz.library.import')}
                </button>
              </div>
            ) : null}
          </div>
        </div>
      ) : null}
    </LmsPage>
  )
}
