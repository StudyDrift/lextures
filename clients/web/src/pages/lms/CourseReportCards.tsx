import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  fetchCourseReportCards,
  patchReportCard,
  releaseReportCards,
  fetchAICommentSuggestion,
  fetchCommentBank,
  type ReportCard,
  type CommentBankEntry,
} from '../../lib/report-cards-api'
import { authorizedFetch } from '../../lib/api'
import { LmsPage } from './lms-page'

type RosterEntry = { userId: string; email: string; displayName?: string | null }

const INPUT_CLASS =
  'rounded-lg border border-slate-200 bg-white px-2 py-1 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100'

const BTN_PRIMARY =
  'inline-flex items-center rounded-lg bg-indigo-600 px-2 py-1 text-xs font-semibold text-white transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50'

const BTN_SECONDARY =
  'inline-flex items-center rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800'

const BTN_SUCCESS =
  'inline-flex items-center rounded-lg bg-emerald-600 px-2 py-1 text-xs font-semibold text-white transition-[background-color,color,border-color] hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50'

function studentLabel(s: RosterEntry): string {
  return s.displayName?.trim() || s.email
}

function currentQuarter(): string {
  const now = new Date()
  const q = Math.ceil((now.getMonth() + 1) / 3)
  return `Q${q}-${now.getFullYear()}`
}

function reportCardStatusBadgeClass(status: string | undefined): string {
  switch (status) {
    case 'released':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300'
    case 'approved':
      return 'bg-sky-100 text-sky-900 dark:bg-sky-950/40 dark:text-sky-300'
    case 'submitted':
      return 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-300'
    default:
      return 'bg-slate-100 text-slate-600 dark:bg-neutral-800/60 dark:text-neutral-400'
  }
}

export default function CourseReportCards() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const [period, setPeriod] = useState<string>(currentQuarter())
  const [cards, setCards] = useState<ReportCard[]>([])
  const [roster, setRoster] = useState<RosterEntry[]>([])
  const [commentBank, setCommentBank] = useState<CommentBankEntry[]>([])
  const [orgId, setOrgId] = useState<string | null>(null)
  const [courseName, setCourseName] = useState<string>('')

  const [editingCardId, setEditingCardId] = useState<string | null>(null)
  const [draftComments, setDraftComments] = useState<Record<string, string>>({})
  const [aiLoading, setAiLoading] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)
  const [releasing, setReleasing] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successMsg, setSuccessMsg] = useState<string | null>(null)
  const [bankCategory, setBankCategory] = useState<string>('all')
  const announceRef = useRef<HTMLDivElement>(null)

  function announce(msg: string) {
    if (announceRef.current) announceRef.current.textContent = msg
  }

  // Load current user's org and course info
  useEffect(() => {
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/me')
        if (!res.ok) return
        const me = (await res.json()) as { orgId?: string }
        if (me.orgId) setOrgId(me.orgId)
      } catch { /* ignore */ }
    })()
  }, [])

  useEffect(() => {
    if (!courseCode) return
    void (async () => {
      try {
        const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}`)
        if (!res.ok) return
        const body = (await res.json()) as { name?: string }
        setCourseName(body.name ?? courseCode)
      } catch { /* ignore */ }
    })()
  }, [courseCode])

  // Load roster
  useEffect(() => {
    if (!courseCode) return
    void (async () => {
      try {
        const res = await authorizedFetch(
          `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments`,
        )
        if (!res.ok) return
        const body = (await res.json()) as {
          enrollments?: Array<{ userId: string; email: string; displayName?: string | null; role: string }>
        }
        const students = (body.enrollments ?? [])
          .filter((e) => e.role === 'student' || e.role === 'learner')
          .map((e) => ({ userId: e.userId, email: e.email, displayName: e.displayName }))
        setRoster(students)
      } catch { /* ignore */ }
    })()
  }, [courseCode])

  // Load comment bank
  useEffect(() => {
    if (!orgId) return
    void (async () => {
      try {
        const data = await fetchCommentBank(orgId)
        setCommentBank(data.entries)
      } catch { /* comment bank may not be seeded */ }
    })()
  }, [orgId])

  // Load report cards for this course + period
  const loadCards = useCallback(async () => {
    if (!courseCode || !period) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchCourseReportCards(courseCode, period)
      setCards(data.reportCards)
      // Seed drafts from existing comments
      const drafts: Record<string, string> = {}
      for (const c of data.reportCards) {
        drafts[c.id] = c.comment ?? ''
      }
      setDraftComments(drafts)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load report cards.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, period])

  useEffect(() => {
    void loadCards()
  }, [loadCards])

  const cardForStudent = (studentId: string) => cards.find((c) => c.studentId === studentId)

  const handleSaveComment = async (cardId: string) => {
    setSaving(cardId)
    try {
      const updated = await patchReportCard(cardId, { comment: draftComments[cardId] ?? '' })
      setCards((prev) => prev.map((c) => (c.id === cardId ? updated : c)))
      setEditingCardId(null)
      announce('Comment saved.')
      setSuccessMsg('Comment saved.')
      setTimeout(() => setSuccessMsg(null), 3000)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save comment.')
    } finally {
      setSaving(null)
    }
  }

  const handleSubmitCard = async (cardId: string) => {
    setSaving(cardId)
    try {
      const updated = await patchReportCard(cardId, {
        comment: draftComments[cardId] ?? '',
        status: 'submitted',
      })
      setCards((prev) => prev.map((c) => (c.id === cardId ? updated : c)))
      announce('Report card submitted for review.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to submit.')
    } finally {
      setSaving(null)
    }
  }

  const handleApproveCard = async (cardId: string) => {
    setSaving(cardId)
    try {
      const updated = await patchReportCard(cardId, { status: 'approved' })
      setCards((prev) => prev.map((c) => (c.id === cardId ? updated : c)))
      announce('Report card approved.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to approve.')
    } finally {
      setSaving(null)
    }
  }

  const handleRelease = async () => {
    if (!courseCode) return
    setReleasing(true)
    setError(null)
    try {
      const result = await releaseReportCards(courseCode, period)
      setSuccessMsg(result.message)
      announce(result.message)
      await loadCards()
      setTimeout(() => setSuccessMsg(null), 5000)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to release report cards.')
    } finally {
      setReleasing(false)
    }
  }

  const handleAISuggest = async (_studentId: string, cardId: string) => {
    const card = cards.find((c) => c.id === cardId)
    if (!card) return
    setAiLoading(cardId)
    try {
      const absences = 0 // TODO: wire from attendance summary
      const suggestion = await fetchAICommentSuggestion(
        courseName,
        card.finalGradePct ?? 0,
        absences,
      )
      setDraftComments((prev) => ({ ...prev, [cardId]: suggestion }))
      announce('AI suggestion inserted. Review and edit before saving.')
    } catch {
      setError('AI comment suggestion is unavailable.')
    } finally {
      setAiLoading(null)
    }
  }

  const insertBankPhrase = (cardId: string, text: string) => {
    setDraftComments((prev) => ({
      ...prev,
      [cardId]: prev[cardId] ? prev[cardId] + ' ' + text : text,
    }))
    announce('Phrase inserted.')
  }

  const bankCategories = ['all', ...Array.from(new Set(commentBank.map((e) => e.category))).sort()]
  const filteredBank =
    bankCategory === 'all' ? commentBank : commentBank.filter((e) => e.category === bankCategory)

  const approvedCount = cards.filter((c) => c.status === 'approved').length
  const releasedCount = cards.filter((c) => c.status === 'released').length

  return (
    <LmsPage title="Report Cards">
      <div
        ref={announceRef}
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      />

      <div className="space-y-4">
        {/* Period selector */}
        <div className="flex flex-wrap items-center gap-3">
          <label htmlFor="grading-period" className="text-sm font-medium text-slate-900 dark:text-neutral-100">
            Grading Period:
          </label>
          <input
            id="grading-period"
            type="text"
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className={`${INPUT_CLASS} w-32`}
            placeholder="Q1-2026"
            aria-describedby="period-hint"
          />
          <span id="period-hint" className="text-xs text-slate-500 dark:text-neutral-400">
            e.g. Q1-2026, S1-2026
          </span>
          <span className="ms-auto text-sm text-slate-500 dark:text-neutral-400">
            {approvedCount} approved · {releasedCount} released of {roster.length} students
          </span>
        </div>

        {error && (
          <div
            role="alert"
            className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-200"
          >
            {error}
          </div>
        )}
        {successMsg && (
          <div
            role="status"
            className="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-950/40 dark:text-emerald-200"
          >
            {successMsg}
          </div>
        )}

        {/* Comment bank sidebar */}
        {commentBank.length > 0 && editingCardId && (
          <aside
            aria-label="Comment Bank"
            className="rounded-xl border border-slate-200 bg-white p-3 text-sm shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
          >
            <div className="mb-2 flex items-center gap-2">
              <strong className="text-slate-900 dark:text-neutral-100">Comment Bank</strong>
              <select
                value={bankCategory}
                onChange={(e) => setBankCategory(e.target.value)}
                className={`${INPUT_CLASS} text-xs`}
                aria-label="Filter by category"
              >
                {bankCategories.map((cat) => (
                  <option key={cat} value={cat}>
                    {cat === 'all' ? 'All categories' : cat}
                  </option>
                ))}
              </select>
            </div>
            <ul className="flex max-h-40 flex-col gap-1 overflow-y-auto">
              {filteredBank.map((entry) => (
                <li key={entry.id}>
                  <button
                    type="button"
                    onClick={() => insertBankPhrase(editingCardId, entry.text)}
                    className="w-full text-start text-xs text-indigo-700 hover:underline dark:text-indigo-300"
                    aria-label={`Insert: ${entry.text}`}
                  >
                    {entry.text}
                  </button>
                </li>
              ))}
            </ul>
          </aside>
        )}

        {loading && (
          <p className="text-sm text-slate-500 dark:text-neutral-400">Loading report cards…</p>
        )}

        {/* Student table */}
        {!loading && (
          <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
            <table className="w-full text-sm" role="grid" aria-label="Report cards">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50 text-start text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-neutral-400">
                  <th scope="col" className="px-4 py-3">Student</th>
                  <th scope="col" className="px-4 py-3">Final %</th>
                  <th scope="col" className="px-4 py-3">Letter</th>
                  <th scope="col" className="px-4 py-3">Status</th>
                  <th scope="col" className="px-4 py-3">Comment</th>
                  <th scope="col" className="px-4 py-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {roster.map((student) => {
                  const card = cardForStudent(student.userId)
                  const isEditing = card && editingCardId === card.id
                  const isSaving = card && saving === card.id
                  const isAILoading = card && aiLoading === card.id
                  return (
                    <tr
                      key={student.userId}
                      className="border-b border-slate-100 last:border-0 hover:bg-slate-50/80 dark:border-neutral-800 dark:hover:bg-neutral-800/80"
                    >
                      <td className="px-4 py-3 font-medium text-slate-900 dark:text-neutral-100">
                        {studentLabel(student)}
                      </td>
                      <td className="px-4 py-3 tabular-nums text-slate-600 dark:text-neutral-400">
                        {card?.finalGradePct != null ? `${card.finalGradePct.toFixed(1)}%` : '—'}
                      </td>
                      <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">
                        {card?.letterGrade ?? '—'}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${reportCardStatusBadgeClass(card?.status)}`}
                        >
                          {card?.status ?? 'not started'}
                        </span>
                      </td>
                      <td className="max-w-xs px-4 py-3">
                        {isEditing && card ? (
                          <textarea
                            rows={3}
                            className={`${INPUT_CLASS} w-full resize-y text-xs`}
                            value={draftComments[card.id] ?? ''}
                            onChange={(e) =>
                              setDraftComments((prev) => ({ ...prev, [card.id]: e.target.value }))
                            }
                            aria-label={`Comment for ${studentLabel(student)}`}
                            autoFocus
                          />
                        ) : (
                          <span className="line-clamp-2 text-xs text-slate-500 dark:text-neutral-400">
                            {card?.comment || (
                              <em className="text-slate-400 dark:text-neutral-500">No comment</em>
                            )}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {card ? (
                          <div className="flex flex-wrap gap-1">
                            {isEditing ? (
                              <>
                                <button
                                  type="button"
                                  onClick={() => void handleSaveComment(card.id)}
                                  disabled={!!isSaving}
                                  className={BTN_PRIMARY}
                                >
                                  {isSaving ? 'Saving…' : 'Save'}
                                </button>
                                <button
                                  type="button"
                                  onClick={() => void handleSubmitCard(card.id)}
                                  disabled={!!isSaving}
                                  className={BTN_SUCCESS}
                                >
                                  Submit
                                </button>
                                <button
                                  type="button"
                                  onClick={() => void handleAISuggest(student.userId, card.id)}
                                  disabled={!!isAILoading}
                                  className="inline-flex items-center rounded-lg bg-violet-600 px-2 py-1 text-xs font-semibold text-white transition-[background-color,color,border-color] hover:bg-violet-500 disabled:cursor-not-allowed disabled:opacity-50"
                                  aria-label="Get AI comment suggestion"
                                >
                                  {isAILoading ? 'AI…' : 'AI Suggest'}
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setEditingCardId(null)}
                                  className={BTN_SECONDARY}
                                >
                                  Cancel
                                </button>
                              </>
                            ) : (
                              <>
                                {(card.status === 'draft' || card.status === 'submitted') && (
                                  <button
                                    type="button"
                                    onClick={() => setEditingCardId(card.id)}
                                    className={BTN_SECONDARY}
                                  >
                                    Edit
                                  </button>
                                )}
                                {card.status === 'submitted' && (
                                  <button
                                    type="button"
                                    onClick={() => void handleApproveCard(card.id)}
                                    disabled={!!isSaving}
                                    className={BTN_PRIMARY}
                                  >
                                    Approve
                                  </button>
                                )}
                                {card.pdfUrl && (
                                  <a
                                    href={`/api/v1/report-cards/${card.id}/pdf`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="inline-flex items-center rounded-lg bg-slate-600 px-2 py-1 text-xs font-semibold text-white transition-[background-color,color,border-color] hover:bg-slate-500 dark:bg-neutral-700 dark:hover:bg-neutral-600"
                                  >
                                    PDF
                                  </a>
                                )}
                              </>
                            )}
                          </div>
                        ) : (
                          <span className="text-xs text-slate-500 dark:text-neutral-400">No card yet</span>
                        )}
                      </td>
                    </tr>
                  )
                })}
                {roster.length === 0 && !loading && (
                  <tr>
                    <td colSpan={6} className="py-6 text-center text-sm text-slate-500 dark:text-neutral-400">
                      No students enrolled.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* Release action */}
        {approvedCount > 0 && (
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => void handleRelease()}
              disabled={releasing}
              className="inline-flex items-center rounded-xl bg-emerald-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {releasing ? 'Releasing…' : `Release ${approvedCount} Approved Card(s) to Parents`}
            </button>
            <span className="text-xs text-slate-500 dark:text-neutral-400">
              Parents will see released report cards in the parent portal.
            </span>
          </div>
        )}
      </div>
    </LmsPage>
  )
}
