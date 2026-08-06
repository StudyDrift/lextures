import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router-dom'
import {
  fetchModerationReconciliation,
  postModerationReconcile,
  type ModerationReconciliationRow,
} from '../../lib/courses-api'
import { EntityLabel } from '../../components/ui/entity-label'
import { usePrompt } from '../../components/use-prompt'
import { formatEntityLabel } from '../../lib/format-entity-label'
import { LmsPage } from './lms-page'

export default function ModerationDashboard() {
  const { t } = useTranslation('common')
  const { prompt, InputDialogHost } = usePrompt()
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const [rows, setRows] = useState<ModerationReconciliationRow[]>([])
  const [unreconciledFlagged, setUnreconciledFlagged] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchModerationReconciliation(courseCode, itemId)
      setRows(data.rows)
      setUnreconciledFlagged(data.unreconciledFlaggedCount)
    } catch (e) {
      setRows([])
      setUnreconciledFlagged(0)
      setError(e instanceof Error ? e.message : 'Could not load reconciliation data.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  async function reconcile(
    submissionId: string,
    body: {
      action: 'accept_grader' | 'average' | 'override' | 'single'
      graderId?: string
      overrideScore?: number
    },
  ) {
    if (!courseCode || !itemId) return
    setBusyId(submissionId)
    setError(null)
    try {
      await postModerationReconcile(courseCode, itemId, submissionId, body)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Reconciliation failed.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleOverride(row: ModerationReconciliationRow) {
    const max = row.pointsWorth ?? 'max'
    const raw = await prompt({
      title: t('moderation.overrideScore.title', { max }),
      label: t('moderation.overrideScore.label'),
      defaultValue: row.finalScore != null ? String(row.finalScore) : '',
    })
    if (raw == null || raw.trim() === '') return
    const n = Number(raw)
    if (!Number.isFinite(n)) return
    await reconcile(row.submissionId, { action: 'override', overrideScore: n })
  }

  if (!courseCode || !itemId) {
    return (
      <LmsPage title="Moderation" description="">
        <p className="mt-6 text-sm text-fg-muted">Invalid link.</p>
      </LmsPage>
    )
  }

  const back = `/courses/${encodeURIComponent(courseCode)}/modules/assignment/${encodeURIComponent(itemId)}`

  return (
    <LmsPage
      title="Moderated grading"
      description="Compare provisional scores and record the final gradebook score for each submission."
      actions={
        <Link
          to={back}
          className="rounded-xl border border-border-strong bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:hover:bg-surface-raised"
        >
          Back to assignment
        </Link>
      }
    >
      {error ? (
        <p className="mt-4 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200">
          {error}
        </p>
      ) : null}
      {!loading && unreconciledFlagged > 0 ? (
        <p className="mt-4 text-sm font-medium text-amber-800 dark:text-amber-200">
          {unreconciledFlagged} flagged submission{unreconciledFlagged === 1 ? '' : 's'} still need
          reconciliation before the gradebook can be saved for this assignment.
        </p>
      ) : null}
      {loading ? (
        <p className="mt-8 text-sm text-fg-muted">Loading…</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-xl border border-border-default">
          <table
            className="min-w-full border-collapse text-start text-sm"
            role="grid"
            aria-label="Moderated grading reconciliation"
          >
            <thead className="bg-surface-base text-fg-muted dark:bg-surface-raised dark:text-fg-muted">
              <tr>
                <th className="border-b border-border-default px-3 py-2 font-medium dark:border-border-default">
                  Submission
                </th>
                <th className="border-b border-border-default px-3 py-2 font-medium dark:border-border-default">
                  Student
                </th>
                <th className="border-b border-border-default px-3 py-2 font-medium dark:border-border-default">
                  Provisional scores
                </th>
                <th className="border-b border-border-default px-3 py-2 font-medium dark:border-border-default">
                  Status
                </th>
                <th className="border-b border-border-default px-3 py-2 font-medium dark:border-border-default">
                  Final
                </th>
                <th className="border-b border-border-default px-3 py-2 font-medium dark:border-border-default">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr
                  key={r.submissionId}
                  className="border-b border-border-subtle last:border-0 dark:border-border-subtle"
                >
                  <td className="px-3 py-2 text-fg-default">
                    <EntityLabel
                      name={r.submissionLabel ?? r.studentName}
                      fallback={t('entityLabel.unknownSubmission')}
                    />
                  </td>
                  <td className="px-3 py-2 text-fg-default">
                    <EntityLabel name={r.studentName} fallback={t('entityLabel.unknownStudent')} />
                  </td>
                  <td className="px-3 py-2 text-fg-default">
                    {r.provisional.length === 0 ? (
                      <span className="text-fg-subtle">—</span>
                    ) : (
                      <ul className="list-inside list-disc">
                        {r.provisional.map((p) => (
                          <li key={`${p.graderId}-${p.score}`}>
                            {p.score}
                            <span className="ms-1 text-xs text-fg-subtle">
                              (
                              {formatEntityLabel({
                                name: p.graderName,
                                fallback: t('entityLabel.unknownGrader'),
                              })}
                              )
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    {r.flagged ? (
                      <span className="rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900 dark:bg-amber-950/80 dark:text-amber-100">
                        Needs review
                      </span>
                    ) : (
                      <span className="text-xs text-fg-subtle">Within threshold</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-fg-default">
                    {r.finalScore != null ? String(r.finalScore) : '—'}
                    {r.reconciliationSource ? (
                      <span className="ms-1 text-xs text-fg-subtle">({r.reconciliationSource})</span>
                    ) : null}
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-1">
                      {r.provisional.map((p) => (
                        <button
                          key={p.graderId}
                          type="button"
                          disabled={busyId === r.submissionId}
                          onClick={() =>
                            void reconcile(r.submissionId, {
                              action: 'accept_grader',
                              graderId: p.graderId,
                            })
                          }
                          className="rounded-lg border border-border-default bg-surface-raised px-2 py-1 text-xs font-medium text-fg-default hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:hover:bg-surface-raised"
                        >
                          Use {p.score}
                        </button>
                      ))}
                      {r.provisional.length >= 2 ? (
                        <button
                          type="button"
                          disabled={busyId === r.submissionId}
                          onClick={() => void reconcile(r.submissionId, { action: 'average' })}
                          className="rounded-lg border border-border-default bg-surface-raised px-2 py-1 text-xs font-medium text-fg-default hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:hover:bg-surface-raised"
                        >
                          Average
                        </button>
                      ) : null}
                      {r.provisional.length === 1 ? (
                        <button
                          type="button"
                          disabled={busyId === r.submissionId}
                          onClick={() => void reconcile(r.submissionId, { action: 'single' })}
                          className="rounded-lg border border-border-default bg-surface-raised px-2 py-1 text-xs font-medium text-fg-default hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:hover:bg-surface-raised"
                        >
                          Confirm single
                        </button>
                      ) : null}
                      <button
                        type="button"
                        disabled={busyId === r.submissionId}
                        onClick={() => void handleOverride(r)}
                        className="rounded-lg border border-indigo-200 bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-900 hover:bg-indigo-100 disabled:opacity-50 dark:border-indigo-900 dark:bg-indigo-950/60 dark:text-indigo-100 dark:hover:bg-indigo-950"
                      >
                        Override…
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {InputDialogHost}
    </LmsPage>
  )
}
