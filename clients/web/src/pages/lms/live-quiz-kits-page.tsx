import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Gamepad2, MoreHorizontal, Plus } from 'lucide-react'
import {
  archiveQuizKit,
  createQuizKit,
  duplicateQuizKit,
  listQuizKits,
  restoreQuizKit,
  type QuizKit,
} from '../../lib/live-quiz-api'
import { courseItemCreatePermission, fetchCourse } from '../../lib/courses-api'
import { formatDate } from '../../lib/format'
import { toastMutationError } from '../../lib/lms-toast'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

export default function LiveQuizKitsPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { courseCode: rawCode } = useParams<{ courseCode: string }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const { allows, loading: permLoading } = usePermissions()
  const { ffInteractiveQuizzes } = usePlatformFeatures()
  const canCreate = !permLoading && !!courseCode && allows(courseItemCreatePermission(courseCode))

  const [kits, setKits] = useState<QuizKit[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newDescription, setNewDescription] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [search, setSearch] = useState('')
  const [showArchived, setShowArchived] = useState(false)
  const [menuKitId, setMenuKitId] = useState<string | null>(null)

  const base = `/courses/${encodeURIComponent(courseCode)}/live-quizzes`

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      if (!ffInteractiveQuizzes) {
        setError(t('liveQuiz.error.disabled'))
        return
      }
      const course = await fetchCourse(courseCode)
      if (!course.interactiveQuizzesEnabled) {
        setError(t('liveQuiz.error.disabled'))
        return
      }
      const result = await listQuizKits(courseCode, {
        q: search.trim() || undefined,
        includeArchived: showArchived,
      })
      setKits(result.kits)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('liveQuiz.error.loadList'))
    } finally {
      setLoading(false)
    }
  }, [courseCode, ffInteractiveQuizzes, search, showArchived, t])

  useEffect(() => {
    void load()
  }, [load])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!newTitle.trim()) return
    setSubmitting(true)
    try {
      const created = await createQuizKit(courseCode, newTitle.trim(), newDescription.trim())
      setNewTitle('')
      setNewDescription('')
      setCreating(false)
      void navigate(`${base}/${encodeURIComponent(created.id)}`)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleArchive(kitId: string) {
    setMenuKitId(null)
    try {
      await archiveQuizKit(courseCode, kitId)
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleRestore(kitId: string) {
    setMenuKitId(null)
    try {
      await restoreQuizKit(courseCode, kitId)
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleDuplicate(kitId: string) {
    setMenuKitId(null)
    try {
      const copy = await duplicateQuizKit(courseCode, kitId)
      void navigate(`${base}/${encodeURIComponent(copy.id)}`)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <LmsPage title={t('liveQuiz.gallery.title')}>
      {loading ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4" aria-busy="true">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-36 motion-safe:animate-pulse rounded-lg border border-slate-200 bg-slate-100 dark:border-neutral-700 dark:bg-neutral-800"
            />
          ))}
        </div>
      ) : error ? (
        <div className="space-y-3">
          <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
            {error}
          </div>
          <button
            type="button"
            onClick={() => void load()}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {t('liveQuiz.gallery.retry')}
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-slate-600 dark:text-neutral-300">{t('liveQuiz.gallery.subtitle')}</p>
            {canCreate && !creating ? (
              <button
                type="button"
                onClick={() => setCreating(true)}
                aria-label={t('liveQuiz.kit.createAria')}
                className="inline-flex min-h-11 items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
              >
                <Plus className="h-4 w-4" aria-hidden="true" />
                {t('liveQuiz.kit.create')}
              </button>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <label className="sr-only" htmlFor="live-quiz-search">
              {t('liveQuiz.gallery.searchLabel')}
            </label>
            <input
              id="live-quiz-search"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('liveQuiz.gallery.searchPlaceholder')}
              className="min-h-11 w-full max-w-sm rounded-md border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
            />
            <label className="inline-flex min-h-11 items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
              <input
                type="checkbox"
                checked={showArchived}
                onChange={(e) => setShowArchived(e.target.checked)}
                className="rounded border-slate-300"
              />
              {t('liveQuiz.gallery.showArchived')}
            </label>
          </div>

          {creating ? (
            <form
              onSubmit={(e) => {
                void handleCreate(e)
              }}
              className="rounded-lg border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-800 dark:bg-indigo-950/30"
            >
              <div className="space-y-3">
                <div>
                  <label
                    className="block text-sm font-medium text-slate-700 dark:text-neutral-300"
                    htmlFor="kit-title"
                  >
                    {t('liveQuiz.kit.titleLabel')}
                  </label>
                  <input
                    id="kit-title"
                    type="text"
                    value={newTitle}
                    onChange={(e) => setNewTitle(e.target.value)}
                    maxLength={200}
                    required
                    autoFocus
                    className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                  />
                </div>
                <div>
                  <label
                    className="block text-sm font-medium text-slate-700 dark:text-neutral-300"
                    htmlFor="kit-description"
                  >
                    {t('liveQuiz.kit.descriptionLabel')}
                  </label>
                  <textarea
                    id="kit-description"
                    value={newDescription}
                    onChange={(e) => setNewDescription(e.target.value)}
                    rows={2}
                    className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    type="submit"
                    disabled={submitting || !newTitle.trim()}
                    className="min-h-11 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {submitting ? t('common.loading') : t('liveQuiz.kit.createSubmit')}
                  </button>
                  <button
                    type="button"
                    onClick={() => setCreating(false)}
                    className="min-h-11 rounded-md px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
                  >
                    {t('dialogs.cancel')}
                  </button>
                </div>
              </div>
            </form>
          ) : null}

          {kits.length === 0 ? (
            <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
              <Gamepad2 className="h-10 w-10 text-slate-400 dark:text-neutral-500" aria-hidden />
              <p className="text-sm text-slate-600 dark:text-neutral-300">{t('liveQuiz.gallery.empty')}</p>
            </div>
          ) : (
            <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {kits.map((kit) => (
                <li key={kit.id} className="relative">
                  <Link
                    to={`${base}/${encodeURIComponent(kit.id)}`}
                    className="block min-h-11 rounded-lg border border-slate-200 p-4 transition hover:border-indigo-300 hover:bg-indigo-50/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-neutral-700 dark:hover:border-indigo-700 dark:hover:bg-indigo-950/20"
                    aria-label={kit.title}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <h3 className="font-medium text-slate-900 dark:text-neutral-100">{kit.title}</h3>
                      <span className="shrink-0 rounded bg-slate-100 px-1.5 py-0.5 text-xs capitalize text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                        {t(`liveQuiz.kit.status.${kit.status}`)}
                      </span>
                    </div>
                    {kit.description ? (
                      <p className="mt-1 line-clamp-2 text-sm text-slate-600 dark:text-neutral-300">
                        {kit.description}
                      </p>
                    ) : null}
                    <p className="mt-3 text-xs text-slate-500 dark:text-neutral-400">
                      {t('liveQuiz.gallery.questionCount', { count: kit.questionCount })}
                    </p>
                    <p className="mt-1 text-xs text-slate-400 dark:text-neutral-500">
                      {t('liveQuiz.gallery.updated', { date: formatDate(kit.updatedAt) })}
                    </p>
                  </Link>
                  {canCreate ? (
                    <div className="absolute end-2 top-10">
                      <button
                        type="button"
                        aria-label={t('liveQuiz.kit.menuAria', { title: kit.title })}
                        aria-expanded={menuKitId === kit.id}
                        aria-haspopup="menu"
                        onClick={() => setMenuKitId(menuKitId === kit.id ? null : kit.id)}
                        className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-md text-slate-500 hover:bg-slate-100 hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
                      >
                        <MoreHorizontal className="h-4 w-4" aria-hidden />
                      </button>
                      {menuKitId === kit.id ? (
                        <div
                          role="menu"
                          className="absolute end-0 z-10 mt-1 w-40 rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
                        >
                          <button
                            type="button"
                            role="menuitem"
                            className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                            onClick={() => void handleDuplicate(kit.id)}
                          >
                            {t('liveQuiz.kit.duplicate')}
                          </button>
                          {kit.archived ? (
                            <button
                              type="button"
                              role="menuitem"
                              className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                              onClick={() => void handleRestore(kit.id)}
                            >
                              {t('liveQuiz.kit.restore')}
                            </button>
                          ) : (
                            <button
                              type="button"
                              role="menuitem"
                              className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                              onClick={() => void handleArchive(kit.id)}
                            >
                              {t('liveQuiz.kit.archive')}
                            </button>
                          )}
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </LmsPage>
  )
}
