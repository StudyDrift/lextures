import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LayoutGrid, Plus } from 'lucide-react'
import { listBoards, type Board } from '../../lib/boards-api'
import { courseItemCreatePermission, fetchCourse } from '../../lib/courses-api'
import { formatDate } from '../../lib/format'
import { usePermissions } from '../../context/use-permissions'
import { CreateBoardDialog } from '../../components/boards/create-board-dialog'
import { LmsPage } from './lms-page'

export default function CourseBoardsPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { courseCode: rawCode } = useParams<{ courseCode: string }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const { allows, loading: permLoading } = usePermissions()
  const canCreate = !permLoading && !!courseCode && allows(courseItemCreatePermission(courseCode))

  const [boards, setBoards] = useState<Board[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const base = `/courses/${encodeURIComponent(courseCode)}/boards`

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      const course = await fetchCourse(courseCode)
      if (!course.visualBoardsEnabled) {
        setError(t('boards.error.disabled'))
        return
      }
      setBoards(await listBoards(courseCode))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('boards.error.loadList'))
    } finally {
      setLoading(false)
    }
  }, [courseCode, t])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <LmsPage title={t('boards.list.title')}>
      {loading ? (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-slate-500 dark:text-neutral-400">{t('common.loading')}</span>
        </div>
      ) : error ? (
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
          {error}
        </div>
      ) : (
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-slate-600 dark:text-neutral-300">{t('boards.list.subtitle')}</p>
            {canCreate ? (
              <button
                type="button"
                onClick={() => setCreating(true)}
                aria-label={t('boards.create.aria')}
                className="inline-flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
              >
                <Plus className="h-4 w-4" aria-hidden="true" />
                {t('boards.create.button')}
              </button>
            ) : null}
          </div>

          <CreateBoardDialog
            open={creating}
            onClose={() => setCreating(false)}
            courseCode={courseCode}
            onCreated={(created) => {
              setCreating(false)
              void navigate(`${base}/${encodeURIComponent(created.id)}`)
            }}
          />

          {boards.length === 0 ? (
            <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
              <LayoutGrid className="h-10 w-10 text-slate-400 dark:text-neutral-500" aria-hidden />
              <p className="text-sm text-slate-600 dark:text-neutral-300">{t('boards.list.empty')}</p>
            </div>
          ) : (
            <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {boards.map((board) => (
                <li key={board.id}>
                  <Link
                    to={`${base}/${encodeURIComponent(board.id)}`}
                    className="block rounded-lg border border-slate-200 p-4 transition hover:border-indigo-300 hover:bg-indigo-50/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-neutral-700 dark:hover:border-indigo-700 dark:hover:bg-indigo-950/20"
                  >
                    <h3 className="font-medium text-slate-900 dark:text-neutral-100">{board.title}</h3>
                    {board.description ? (
                      <p className="mt-1 line-clamp-2 text-sm text-slate-600 dark:text-neutral-300">
                        {board.description}
                      </p>
                    ) : null}
                    <p className="mt-3 text-xs text-slate-500 dark:text-neutral-400">
                      {t('boards.list.updated', { date: formatDate(board.updatedAt) })}
                    </p>
                    <p className="mt-1 text-xs text-slate-400 dark:text-neutral-500">
                      {t('boards.list.contributorsPlaceholder')}
                    </p>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </LmsPage>
  )
}
