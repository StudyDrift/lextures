import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { LmsPage } from './lms-page'
import { fetchModuleVibeActivityByItem, type ModuleVibeActivityPayload } from '../../lib/courses-api'
import { recordLastVisitedModuleItem } from '../../lib/last-visited-module-item'
import { usePermissions } from '../../context/use-permissions'
import { permCourseItemCreate } from '../../lib/rbac-api'

export default function CourseModuleVibeActivityPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const [payload, setPayload] = useState<ModuleVibeActivityPayload | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const { hasPermission } = usePermissions()
  const canEdit = courseCode ? hasPermission(permCourseItemCreate(courseCode)) : false

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setLoadError(null)
    try {
      const data = await fetchModuleVibeActivityByItem(courseCode, itemId)
      setPayload(data)
      recordLastVisitedModuleItem(courseCode, {
        kind: 'vibe_activity',
        itemId,
        title: data.title,
      })
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Failed to load activity.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  if (!courseCode || !itemId) {
    return <LmsPage title="Vibe Activity">Invalid route.</LmsPage>
  }

  const title = payload?.title ?? (loading ? 'Loading…' : 'Vibe Activity')

  return (
    <LmsPage
      title={title}
      description={canEdit ? 'Instructor-created interactive web activity' : undefined}
      actions={
        <Link
          to={`/courses/${encodeURIComponent(courseCode)}/modules`}
          className="text-sm text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-200"
        >
          ← Back to modules
        </Link>
      }
    >
      {loading && <p className="text-sm text-slate-500">Loading activity…</p>}
      {loadError && (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {loadError}
        </p>
      )}

      {payload && (
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-2 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
            <iframe
              title={payload.title}
              sandbox="allow-scripts allow-forms allow-same-origin"
              srcDoc={payload.html || '<!doctype html><html><body style="font-family:sans-serif;padding:2rem;color:#666">Empty activity. The instructor has not added content yet.</body></html>'}
              className="block h-[70vh] w-full rounded-lg border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-950"
              style={{ minHeight: '480px' }}
            />
          </div>

          <p className="text-xs text-slate-500 dark:text-neutral-500">
            This is a self-contained instructor-authored interactive activity. It runs in a sandboxed frame.
          </p>

          {canEdit && (
            <div className="pt-2 text-sm text-slate-600 dark:text-neutral-400">
              Instructors: edit the HTML via the Vibe Activity creation flow from the modules page (re-vibe or tweak source).
            </div>
          )}
        </div>
      )}
    </LmsPage>
  )
}
