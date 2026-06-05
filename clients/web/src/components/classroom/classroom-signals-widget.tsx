import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  HALL_PASS_DESTINATIONS,
  listActiveHallPasses,
  listCourseQuestions,
  requestHallPass,
  submitAnonymousQuestion,
  updateHallPassStatus,
  type AnonymousQuestion,
  type HallPass,
  type HallPassDestination,
} from '../../lib/classroom-signals-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

export type ClassroomSignalsRole = 'student' | 'teacher'

type Props = {
  /** Role determines which controls are visible. */
  role: ClassroomSignalsRole
  /** Course UUID (required for the anonymous question queue). */
  courseId: string
  /** Section UUID (required for hall passes). */
  sectionId: string
  /** Polling interval in ms for active passes / question queue. Defaults to 8s. */
  pollMs?: number
}

const DEFAULT_POLL_MS = 8000

/**
 * Classroom signals widget (plan 13.9).
 *
 * Lightweight floating panel that hosts the digital hall pass and the
 * anonymous question queue. Designed to render inline on a course page; the
 * caller decides when to mount it (e.g. only when a teacher has flipped the
 * "Start class session" switch).
 */
export function ClassroomSignalsWidget({
  role,
  courseId,
  sectionId,
  pollMs = DEFAULT_POLL_MS,
}: Props) {
  const { ffClassroomSignals } = usePlatformFeatures()
  if (!ffClassroomSignals) return null
  return (
    <aside
      data-testid="classroom-signals-widget"
      aria-label="Classroom signals"
      className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
    >
      <h2 className="mb-3 text-lg font-semibold">Classroom signals</h2>
      <div className="space-y-4">
        <HallPassSection role={role} sectionId={sectionId} pollMs={pollMs} />
        <QuestionQueueSection role={role} courseId={courseId} pollMs={pollMs} />
      </div>
    </aside>
  )
}

// ─── Hall pass ───────────────────────────────────────────────────────────────

function HallPassSection({
  role,
  sectionId,
  pollMs,
}: {
  role: ClassroomSignalsRole
  sectionId: string
  pollMs: number
}) {
  const [destination, setDestination] = useState<HallPassDestination>('bathroom')
  const [estimatedMins, setEstimatedMins] = useState<number>(5)
  const [requesting, setRequesting] = useState(false)
  const [requestError, setRequestError] = useState<string | null>(null)
  const [activePasses, setActivePasses] = useState<HallPass[]>([])

  const refresh = useCallback(async () => {
    const passes = await listActiveHallPasses(sectionId)
    setActivePasses(passes)
  }, [sectionId])

  useEffect(() => {
    void refresh()
    const id = window.setInterval(() => {
      void refresh()
    }, pollMs)
    return () => window.clearInterval(id)
  }, [refresh, pollMs])

  const handleRequest = async (e: React.FormEvent) => {
    e.preventDefault()
    setRequesting(true)
    setRequestError(null)
    try {
      await requestHallPass(sectionId, destination, estimatedMins || undefined)
      await refresh()
    } catch (err) {
      setRequestError(err instanceof Error ? err.message : 'Failed to request hall pass')
    } finally {
      setRequesting(false)
    }
  }

  const handleApprove = async (passId: string) => {
    await updateHallPassStatus(passId, 'approved')
    await refresh()
  }
  const handleDeny = async (passId: string) => {
    await updateHallPassStatus(passId, 'denied')
    await refresh()
  }
  const handleReturn = async (passId: string) => {
    await updateHallPassStatus(passId, 'returned')
    await refresh()
  }

  return (
    <section aria-labelledby="hall-pass-heading" className="space-y-2">
      <h3 id="hall-pass-heading" className="text-sm font-semibold uppercase tracking-wide">
        Hall pass
      </h3>

      {role === 'student' && (
        <form onSubmit={handleRequest} className="flex flex-wrap items-end gap-2">
          <label className="flex flex-col text-xs">
            <span>Destination</span>
            <select
              data-testid="hall-pass-destination"
              value={destination}
              onChange={(e) => setDestination(e.target.value as HallPassDestination)}
              className="rounded border border-gray-300 px-2 py-1 text-sm"
            >
              {HALL_PASS_DESTINATIONS.map((d) => (
                <option key={d} value={d}>
                  {d.charAt(0).toUpperCase() + d.slice(1)}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col text-xs">
            <span>Est. minutes</span>
            <input
              data-testid="hall-pass-minutes"
              type="number"
              min={1}
              max={60}
              value={estimatedMins}
              onChange={(e) => setEstimatedMins(Number(e.target.value))}
              className="w-20 rounded border border-gray-300 px-2 py-1 text-sm"
            />
          </label>
          <button
            type="submit"
            data-testid="hall-pass-request"
            disabled={requesting}
            className="rounded bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {requesting ? 'Requesting…' : 'Request hall pass'}
          </button>
          {requestError && (
            <p role="alert" className="w-full text-xs text-red-600">
              {requestError}
            </p>
          )}
        </form>
      )}

      <div>
        <p className="text-xs font-medium">
          {activePasses.length === 0 ? 'No active passes.' : `Currently out (${activePasses.length}):`}
        </p>
        <ul data-testid="hall-pass-active-list" className="mt-1 space-y-1">
          {activePasses.map((p) => (
            <li
              key={p.id}
              data-testid={`hall-pass-row-${p.id}`}
              data-status={p.status}
              className="flex flex-wrap items-center gap-2 rounded border border-gray-200 px-2 py-1 text-sm"
            >
              <span className="font-mono text-xs">{p.studentId?.slice(0, 8) ?? '—'}</span>
              <span className="capitalize">{p.destination}</span>
              <span className="text-xs text-gray-600">{p.status}</span>
              {p.overdue && (
                <span role="alert" className="text-xs font-semibold text-red-600">
                  Overdue
                </span>
              )}
              {role === 'teacher' && p.status === 'requested' && (
                <>
                  <button
                    type="button"
                    onClick={() => void handleApprove(p.id)}
                    className="rounded bg-green-600 px-2 py-0.5 text-xs text-white"
                    data-testid={`hall-pass-approve-${p.id}`}
                  >
                    Approve
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleDeny(p.id)}
                    className="rounded bg-gray-200 px-2 py-0.5 text-xs"
                    data-testid={`hall-pass-deny-${p.id}`}
                  >
                    Deny
                  </button>
                </>
              )}
              {p.status === 'approved' && (
                <button
                  type="button"
                  onClick={() => void handleReturn(p.id)}
                  className="rounded bg-blue-100 px-2 py-0.5 text-xs"
                  data-testid={`hall-pass-return-${p.id}`}
                >
                  Mark returned
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

// ─── Anonymous question queue ─────────────────────────────────────────────────

function QuestionQueueSection({
  role,
  courseId,
  pollMs,
}: {
  role: ClassroomSignalsRole
  courseId: string
  pollMs: number
}) {
  const [questions, setQuestions] = useState<AnonymousQuestion[]>([])
  const [draft, setDraft] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitMessage, setSubmitMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (role !== 'teacher') return
    const items = await listCourseQuestions(courseId, false)
    setQuestions(items)
  }, [courseId, role])

  useEffect(() => {
    void refresh()
    if (role !== 'teacher') return
    const id = window.setInterval(() => {
      void refresh()
    }, pollMs)
    return () => window.clearInterval(id)
  }, [refresh, role, pollMs])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const text = draft.trim()
    if (!text) return
    setSubmitting(true)
    setError(null)
    setSubmitMessage(null)
    try {
      await submitAnonymousQuestion(courseId, text)
      setDraft('')
      setSubmitMessage('Question submitted anonymously.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit question')
    } finally {
      setSubmitting(false)
    }
  }

  // Stable, accessible empty-state message id; re-derived only when role changes.
  const teacherEmpty = useMemo(() => `empty-${courseId}`, [courseId])
  const inputRef = useRef<HTMLTextAreaElement | null>(null)

  return (
    <section aria-labelledby="qq-heading" className="space-y-2">
      <h3 id="qq-heading" className="text-sm font-semibold uppercase tracking-wide">
        Anonymous questions
      </h3>

      {role === 'student' && (
        <form onSubmit={handleSubmit} className="space-y-1">
          <label className="block text-xs">
            <span className="sr-only">Your question</span>
            <textarea
              ref={inputRef}
              data-testid="question-input"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              maxLength={500}
              rows={2}
              placeholder="Ask anything – your name will not be shared with the class."
              className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
            />
          </label>
          <div className="flex items-center gap-2">
            <button
              type="submit"
              disabled={submitting || draft.trim().length === 0}
              data-testid="question-submit"
              className="rounded bg-blue-600 px-3 py-1 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {submitting ? 'Sending…' : 'Submit anonymously'}
            </button>
            {submitMessage && (
              <p role="status" className="text-xs text-green-700" data-testid="question-confirmation">
                {submitMessage}
              </p>
            )}
            {error && (
              <p role="alert" className="text-xs text-red-600">
                {error}
              </p>
            )}
          </div>
        </form>
      )}

      {role === 'teacher' && (
        <div data-testid="question-queue">
          {questions.length === 0 ? (
            <p id={teacherEmpty} className="text-xs text-gray-500">
              No questions yet.
            </p>
          ) : (
            <ul className="space-y-1">
              {questions.map((q) => (
                <li
                  key={q.id}
                  data-testid={`question-row-${q.id}`}
                  className="rounded border border-gray-200 px-2 py-1 text-sm"
                >
                  {q.question}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  )
}
