import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { H5PPlayer } from '../../components/h5p-player'
import { fetchModuleH5PByItem, type ModuleH5PPayload } from '../../lib/courses-api'
import { h5pI18n } from '../../lib/h5p-i18n'
import { recordLastVisitedModuleItem } from '../../lib/last-visited-module-item'
import { LmsPage } from './lms-page'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export default function CourseModuleH5PPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const [payload, setPayload] = useState<ModuleH5PPayload | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchModuleH5PByItem(courseCode, itemId)
      setPayload(data)
      recordLastVisitedModuleItem(courseCode, { kind: 'h5p', itemId, title: data.title })
    } catch (e) {
      setError(e instanceof Error ? e.message : h5pI18n.error)
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  useCoursePageTitle(!loading && payload?.title ? payload.title : null)

  if (!courseCode || !itemId) {
    return null
  }

  const renderUrl =
    payload?.packageId != null
      ? `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/h5p/${encodeURIComponent(payload.packageId)}/render`
      : ''
  const downloadUrl =
    payload?.packageId != null
      ? `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/h5p/${encodeURIComponent(payload.packageId)}/download`
      : undefined

  return (
    <LmsPage title={payload?.title ?? 'Interactive activity'}>
      {loading ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400">{h5pI18n.loading}</p>
      ) : error ? (
        <p className="text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : payload ? (
        <H5PPlayer
          courseCode={courseCode}
          packageId={payload.packageId}
          title={payload.title}
          renderUrl={renderUrl}
          downloadUrl={downloadUrl}
          ready={payload.extractStatus === 'ready'}
        />
      ) : null}
    </LmsPage>
  )
}
