import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getAttendanceSession, selfReportAttendance, type AttendanceSessionDetail } from '../../lib/course-attendance-api'
import { LmsPage } from './lms-page'

type CheckinState = 'loading' | 'submitting' | 'done' | 'already_recorded' | 'window_closed' | 'error'

export default function AttendanceCheckin() {
  const { courseCode, sessionId } = useParams<{ courseCode: string; sessionId: string }>()
  const [session, setSession] = useState<AttendanceSessionDetail | null>(null)
  const [state, setState] = useState<CheckinState>('loading')
  const [recordedStatus, setRecordedStatus] = useState<'present' | 'tardy'>('present')
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  useEffect(() => {
    if (!courseCode || !sessionId) return
    void (async () => {
      try {
        const detail = await getAttendanceSession(courseCode, sessionId)
        setSession(detail)
        if (detail.myRecord && detail.myRecord.status !== 'not_recorded') {
          setRecordedStatus(detail.myRecord.status as 'present' | 'tardy')
          setState('already_recorded')
          return
        }
        if (!detail.canSelfReport) {
          setState('window_closed')
          return
        }
        setState('submitting')
        await selfReportAttendance(courseCode, sessionId, 'present')
        setRecordedStatus('present')
        setState('done')
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Check-in failed.'
        if (msg.toLowerCase().includes('window') || msg.toLowerCase().includes('closed')) {
          setState('window_closed')
        } else if (msg.toLowerCase().includes('already')) {
          setState('already_recorded')
        } else {
          setErrorMsg(msg)
          setState('error')
        }
      }
    })()
  }, [courseCode, sessionId])

  const handleMarkTardy = async () => {
    if (!courseCode || !sessionId) return
    setState('submitting')
    try {
      await selfReportAttendance(courseCode, sessionId, 'tardy')
      setRecordedStatus('tardy')
      setState('done')
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : 'Check-in failed.')
      setState('error')
    }
  }

  const sessionTitle = session?.title ?? 'Attendance session'

  return (
    <LmsPage title="Check in">
      <div className="flex flex-col items-center justify-center py-12">
        {state === 'loading' || state === 'submitting' ? (
          <p className="text-sm text-slate-500" aria-busy="true">
            {state === 'submitting' ? 'Recording your attendance…' : 'Loading…'}
          </p>
        ) : state === 'done' ? (
          <div className="w-full max-w-sm rounded-2xl border border-emerald-200 bg-emerald-50 p-8 text-center shadow-sm dark:border-emerald-900 dark:bg-emerald-950/30">
            <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/40">
              <svg className="h-7 w-7 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h2 className="text-lg font-semibold text-emerald-900 dark:text-emerald-100">
              {recordedStatus === 'tardy' ? 'Marked tardy' : 'Checked in!'}
            </h2>
            <p className="mt-1 text-sm text-emerald-700 dark:text-emerald-300">{sessionTitle}</p>
            {recordedStatus === 'present' && (
              <button
                type="button"
                onClick={() => void handleMarkTardy()}
                className="mt-4 text-xs text-emerald-600 underline hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-200"
              >
                Actually, I&apos;m late — mark tardy
              </button>
            )}
          </div>
        ) : state === 'already_recorded' ? (
          <div className="w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Already recorded</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Your attendance for <span className="font-medium">{sessionTitle}</span> has already been recorded
              {recordedStatus ? ` as ${recordedStatus}` : ''}.
            </p>
          </div>
        ) : state === 'window_closed' ? (
          <div className="w-full max-w-sm rounded-2xl border border-amber-200 bg-amber-50 p-8 text-center shadow-sm dark:border-amber-900 dark:bg-amber-950/30">
            <h2 className="text-base font-semibold text-amber-900 dark:text-amber-100">Session closed</h2>
            <p className="mt-1 text-sm text-amber-700 dark:text-amber-300">
              The check-in window for <span className="font-medium">{sessionTitle}</span> is no longer open.
            </p>
          </div>
        ) : (
          <div className="w-full max-w-sm rounded-2xl border border-red-200 bg-red-50 p-8 text-center shadow-sm dark:border-red-900 dark:bg-red-950/30">
            <h2 className="text-base font-semibold text-red-900 dark:text-red-100">Check-in failed</h2>
            <p className="mt-1 text-sm text-red-700 dark:text-red-300">{errorMsg ?? 'Something went wrong. Please try again.'}</p>
          </div>
        )}
      </div>
    </LmsPage>
  )
}
