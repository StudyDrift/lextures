import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useEffect, useState } from 'react'
import { fetchCourse, type CoursePublic } from '../../lib/courses-api'
import { ScreenShareConsole } from './screen-share-console'

export default function ScreenSharePage() {
  const { t } = useTranslation('common')
  const { courseCode: raw } = useParams<{ courseCode: string }>()
  const courseCode = raw ? decodeURIComponent(raw) : ''
  const [course, setCourse] = useState<CoursePublic | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!courseCode) return
    let cancelled = false
    void (async () => {
      try {
        const c = await fetchCourse(courseCode)
        if (!cancelled) {
          if (!c.screenShareEnabled) {
            setError(t('screenShare.error.flagOff'))
            return
          }
          setCourse(c)
        }
      } catch {
        if (!cancelled) setError(t('screenShare.error.loadFailed'))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, t])

  const roles = course?.viewerEnrollmentRoles ?? []
  const canHost = roles.some((r) => r === 'teacher' || r === 'ta' || r === 'designer')

  if (error) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p role="alert">{error}</p>
      </div>
    )
  }
  if (!course) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p>{t('screenShare.state.loading')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <ScreenShareConsole courseCode={courseCode} canHost={canHost} />
    </div>
  )
}
