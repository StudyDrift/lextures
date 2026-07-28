import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  deleteContentToolGradeLink,
  fetchContentToolGradeLink,
  putContentToolGradeLink,
  type ContentToolGradeLink,
} from '../../../lib/courses-api'

export type GradeLinkDialogProps = {
  open: boolean
  courseCode: string
  instanceId: string
  onClose: () => void
  onSaved?: (link: ContentToolGradeLink) => void
}

export function GradeLinkDialog({
  open,
  courseCode,
  instanceId,
  onClose,
  onSaved,
}: GradeLinkDialogProps) {
  const { t } = useTranslation('contentTools')
  const [link, setLink] = useState<ContentToolGradeLink | null>(null)
  const [points, setPoints] = useState('10')
  const [policy, setPolicy] = useState('accept')
  const [assignmentItemId, setAssignmentItemId] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    void (async () => {
      try {
        const g = await fetchContentToolGradeLink(courseCode, instanceId)
        if (cancelled) return
        setLink(g)
        if (g.pointsPossible != null) setPoints(String(g.pointsPossible))
        setPolicy(g.latePolicy || 'accept')
        setAssignmentItemId(g.assignmentItemId ?? '')
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, courseCode, instanceId])

  if (!open) return null

  async function save(enable: boolean) {
    setSaving(true)
    setError(null)
    try {
      const pts = Number(points)
      const body = {
        countsForGrade: enable,
        latePolicy: policy,
        pointsPossible: Number.isFinite(pts) ? pts : null,
        assignmentItemId: assignmentItemId.trim() || null,
      }
      const saved = await putContentToolGradeLink(courseCode, instanceId, body)
      setLink(saved)
      onSaved?.(saved)
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to save.')
    } finally {
      setSaving(false)
    }
  }

  async function disable() {
    setSaving(true)
    setError(null)
    try {
      await deleteContentToolGradeLink(courseCode, instanceId)
      onSaved?.({
        instanceId,
        countsForGrade: false,
        latePolicy: 'accept',
      })
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to disable.')
    } finally {
      setSaving(false)
    }
  }

  const fieldClass =
    'mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm outline-none focus-visible:border-indigo-400 focus-visible:ring-2 focus-visible:ring-indigo-500/30 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus-visible:border-indigo-500'

  return (
    <div
      className="fixed inset-0 z-[80] flex items-end justify-center bg-slate-900/50 p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="grade-link-title"
      data-testid="grade-link-dialog"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="w-full max-w-md overflow-hidden rounded-t-2xl border border-slate-200 bg-white shadow-xl sm:rounded-2xl dark:border-neutral-600 dark:bg-neutral-950">
        <div className="border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <h2
            id="grade-link-title"
            className="text-base font-semibold text-slate-900 dark:text-neutral-100"
          >
            {t('contentTools.grading.dialogTitle')}
          </h2>
          <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
            {t('contentTools.grading.dialogHelp')}
          </p>
        </div>
        <div className="space-y-3 px-4 py-4">
          <label className="block text-sm">
            <span className="font-medium text-slate-700 dark:text-neutral-200">
              {t('contentTools.grading.assignmentItemId')}
            </span>
            <input
              className={fieldClass}
              value={assignmentItemId}
              onChange={(e) => setAssignmentItemId(e.target.value)}
              placeholder="structure item uuid"
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium text-slate-700 dark:text-neutral-200">
              {t('contentTools.grading.pointsPossible')}
            </span>
            <input
              type="number"
              min={0}
              step="0.5"
              className={fieldClass}
              value={points}
              onChange={(e) => setPoints(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium text-slate-700 dark:text-neutral-200">
              {t('contentTools.grading.latePolicy')}
            </span>
            <select
              className={fieldClass}
              value={policy}
              onChange={(e) => setPolicy(e.target.value)}
            >
              <option value="accept">{t('contentTools.grading.policy.accept')}</option>
              <option value="accept_marked">{t('contentTools.grading.policy.accept_marked')}</option>
              <option value="reject">{t('contentTools.grading.policy.reject')}</option>
            </select>
          </label>
          {error ? (
            <p
              className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-300"
              role="alert"
            >
              {error}
            </p>
          ) : null}
        </div>
        <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 bg-slate-50/80 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900/80">
          <button
            type="button"
            className="rounded-lg px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
            onClick={onClose}
            disabled={saving}
          >
            {t('contentTools.authoring.cancel')}
          </button>
          {link?.countsForGrade ? (
            <button
              type="button"
              className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200"
              onClick={() => void disable()}
              disabled={saving}
            >
              {t('contentTools.grading.disable')}
            </button>
          ) : null}
          <button
            type="button"
            className="rounded-lg bg-indigo-600 px-3.5 py-1.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50 dark:bg-indigo-500 dark:hover:bg-indigo-400"
            onClick={() => void save(true)}
            disabled={saving}
            data-testid="grade-link-enable"
          >
            {t('contentTools.grading.enable')}
          </button>
        </div>
      </div>
    </div>
  )
}
