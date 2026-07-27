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

  return (
    <div
      className="fixed inset-0 z-[80] flex items-end justify-center bg-slate-900/40 p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="grade-link-title"
      data-testid="grade-link-dialog"
    >
      <div className="w-full max-w-md rounded-t-lg bg-white p-4 shadow-lg sm:rounded-lg dark:bg-neutral-900">
        <h2 id="grade-link-title" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {t('contentTools.grading.dialogTitle')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
          {t('contentTools.grading.dialogHelp')}
        </p>
        <label className="mt-3 block text-sm">
          <span className="text-slate-700 dark:text-neutral-200">
            {t('contentTools.grading.assignmentItemId')}
          </span>
          <input
            className="mt-1 w-full rounded border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            value={assignmentItemId}
            onChange={(e) => setAssignmentItemId(e.target.value)}
            placeholder="structure item uuid"
          />
        </label>
        <label className="mt-3 block text-sm">
          <span className="text-slate-700 dark:text-neutral-200">
            {t('contentTools.grading.pointsPossible')}
          </span>
          <input
            type="number"
            min={0}
            step="0.5"
            className="mt-1 w-full rounded border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            value={points}
            onChange={(e) => setPoints(e.target.value)}
          />
        </label>
        <label className="mt-3 block text-sm">
          <span className="text-slate-700 dark:text-neutral-200">
            {t('contentTools.grading.latePolicy')}
          </span>
          <select
            className="mt-1 w-full rounded border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            value={policy}
            onChange={(e) => setPolicy(e.target.value)}
          >
            <option value="accept">{t('contentTools.grading.policy.accept')}</option>
            <option value="accept_marked">{t('contentTools.grading.policy.accept_marked')}</option>
            <option value="reject">{t('contentTools.grading.policy.reject')}</option>
          </select>
        </label>
        {error ? (
          <p className="mt-2 text-sm text-rose-600" role="alert">
            {error}
          </p>
        ) : null}
        <div className="mt-4 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            className="rounded px-3 py-1.5 text-sm text-slate-700 dark:text-neutral-200"
            onClick={onClose}
            disabled={saving}
          >
            {t('contentTools.authoring.cancel')}
          </button>
          {link?.countsForGrade ? (
            <button
              type="button"
              className="rounded bg-slate-200 px-3 py-1.5 text-sm dark:bg-neutral-700"
              onClick={() => void disable()}
              disabled={saving}
            >
              {t('contentTools.grading.disable')}
            </button>
          ) : null}
          <button
            type="button"
            className="rounded bg-sky-700 px-3 py-1.5 text-sm text-white"
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
