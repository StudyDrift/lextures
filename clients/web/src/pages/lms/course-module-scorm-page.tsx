import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ScormLaunchClient } from '../../components/scorm-player'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { fetchModuleScormByItem, type ModuleScormPayload } from '../../lib/courses-api'
import { scormI18n } from '../../lib/scorm-i18n'
import { recordLastVisitedModuleItem } from '../../lib/last-visited-module-item'
import { LmsPage } from './lms-page'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export default function CourseModuleScormPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const [payload, setPayload] = useState<ModuleScormPayload | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchModuleScormByItem(courseCode, itemId)
      setPayload(data)
      recordLastVisitedModuleItem(courseCode, { kind: 'scorm', itemId, title: data.title })
    } catch (e) {
      setError(e instanceof Error ? e.message : scormI18n.error)
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

  const downloadUrl =
    payload?.packageId != null
      ? `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/scorm/${encodeURIComponent(payload.packageId)}/download`
      : undefined
  const scoId = payload?.scos?.[0]?.id ?? ''

  return (
    <LmsPage title={payload?.title ?? 'SCORM activity'}>
      {loading ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400">{scormI18n.loading}</p>
      ) : error ? (
        <p className="text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : payload && scoId ? (
        <ScormLaunchClient
          courseCode={courseCode}
          scoId={scoId}
          title={payload.title}
          downloadUrl={downloadUrl}
          extractStatus={payload.extractStatus}
        />
      ) : null}
    </LmsPage>
  )
}
