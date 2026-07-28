import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  postContentToolStateReset,
  restoreContentToolStateReset,
  type ContentToolResetRequest,
  type ContentToolResetResponse,
  type ContentToolResetScope,
} from '../../../lib/courses-api'
import { toastWithUndo } from '../../../lib/lms-toast'
import { ResetJobProgress } from './reset-job-progress'
import { ResetScopePicker } from './reset-scope-picker'

export type ToolResetDialogProps = {
  open: boolean
  courseCode: string
  instanceId: string
  itemId?: string
  enrollmentId?: string | null
  defaultScope?: ContentToolResetScope
  onClose: () => void
  onCompleted?: () => void
}

export function ToolResetDialog({
  open,
  courseCode,
  instanceId,
  itemId,
  enrollmentId,
  defaultScope = 'instance_all',
  onClose,
  onCompleted,
}: ToolResetDialogProps) {
  const { t } = useTranslation('contentTools')
  const [scope, setScope] = useState<ContentToolResetScope>(defaultScope)
  const [reason, setReason] = useState('')
  const [notify, setNotify] = useState(true)
  const [postHandling, setPostHandling] = useState<'keep' | 'remove'>('keep')
  const [schedulingHandling, setSchedulingHandling] = useState<'keep' | 'clear'>('keep')
  const [preview, setPreview] = useState<ContentToolResetResponse | null>(null)
  const [confirmText, setConfirmText] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [jobId, setJobId] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setScope(enrollmentId ? 'instance_enrollment' : defaultScope)
    setReason('')
    setNotify(true)
    setPostHandling('keep')
    setSchedulingHandling('keep')
    setPreview(null)
    setConfirmText('')
    setError(null)
    setJobId(null)
  }, [open, enrollmentId, defaultScope])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setBusy(true)
    setError(null)
    const body: ContentToolResetRequest = {
      scope,
      instanceId,
      dryRun: true,
      notify,
      postHandling,
      schedulingHandling,
      ...(itemId ? { itemId } : {}),
      ...(enrollmentId &&
      (scope === 'instance_enrollment' ||
        scope === 'item_enrollment' ||
        scope === 'course_enrollment')
        ? { enrollmentId }
        : {}),
      ...(reason.trim() ? { reason: reason.trim() } : {}),
    }
    void postContentToolStateReset(courseCode, body)
      .then((res) => {
        if (!cancelled) setPreview(res)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Dry-run failed.')
      })
      .finally(() => {
        if (!cancelled) setBusy(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, courseCode, instanceId, itemId, enrollmentId, scope, notify, reason, postHandling, schedulingHandling])

  if (!open) return null

  const affected = preview?.affectedCount ?? 0
  const needsTypedConfirm = affected > 25
  const confirmOk = !needsTypedConfirm || confirmText.trim().toUpperCase() === 'RESET'

  async function execute() {
    setBusy(true)
    setError(null)
    try {
      const body: ContentToolResetRequest = {
        scope,
        instanceId,
        dryRun: false,
        notify,
        postHandling,
        schedulingHandling,
        idempotencyKey: crypto.randomUUID(),
        ...(itemId ? { itemId } : {}),
        ...(enrollmentId &&
        (scope === 'instance_enrollment' ||
          scope === 'item_enrollment' ||
          scope === 'course_enrollment')
          ? { enrollmentId }
          : {}),
        ...(reason.trim() ? { reason: reason.trim() } : {}),
      }
      const res = await postContentToolStateReset(courseCode, body)
      if (res.jobId) {
        setJobId(res.jobId)
        return
      }
      if (res.batchId) {
        const batchId = res.batchId
        toastWithUndo(t('contentTools.reset.successToast', { count: res.affectedCount }), {
          durationMs: 30_000,
          onUndo: async () => {
            const { fetchContentToolStateResets } = await import('../../../lib/courses-api')
            const snaps = await fetchContentToolStateResets(courseCode, { instanceId })
            const batch = snaps.filter((s) => s.batchId === batchId && !s.restoredAt)
            for (const snap of batch) {
              await restoreContentToolStateReset(courseCode, snap.id)
            }
          },
        })
      } else {
        toastWithUndo(t('contentTools.reset.successToast', { count: res.affectedCount }), {
          durationMs: 30_000,
          onUndo: async () => undefined,
        })
      }
      onCompleted?.()
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Reset failed.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-[90] flex items-end justify-center bg-slate-900/40 p-0 sm:items-center sm:p-4"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="tool-reset-title"
      data-testid="tool-reset-dialog"
    >
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-t-2xl border border-slate-200 bg-white p-4 shadow-xl sm:rounded-lg dark:border-neutral-700 dark:bg-neutral-900">
        <h2
          id="tool-reset-title"
          className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
        >
          {t('contentTools.reset.dialogTitle')}
        </h2>
        <div className="mt-3 space-y-4">
          <ResetScopePicker
            value={scope}
            onChange={setScope}
            allowItem={Boolean(itemId)}
            allowCourse
          />
          <label className="block text-sm">
            <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
              {t('contentTools.reset.reasonLabel')}
            </span>
            <textarea
              className="mt-1 w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
              rows={2}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </label>
          <label className="flex items-center gap-2 text-sm text-slate-800 dark:text-neutral-200">
            <input
              type="checkbox"
              checked={notify}
              onChange={(e) => setNotify(e.target.checked)}
            />
            {t('contentTools.reset.notifyLabel')}
          </label>
          <fieldset className="space-y-1 text-sm">
            <legend className="text-xs font-medium text-slate-600 dark:text-neutral-400">
              {t('contentTools.reset.postHandlingLabel')}
            </legend>
            <label className="flex items-start gap-2 text-slate-800 dark:text-neutral-200">
              <input
                type="radio"
                name="postHandling"
                checked={postHandling === 'keep'}
                onChange={() => setPostHandling('keep')}
              />
              <span>{t('contentTools.reset.postHandlingKeep')}</span>
            </label>
            <label className="flex items-start gap-2 text-slate-800 dark:text-neutral-200">
              <input
                type="radio"
                name="postHandling"
                checked={postHandling === 'remove'}
                onChange={() => setPostHandling('remove')}
              />
              <span>{t('contentTools.reset.postHandlingRemove')}</span>
            </label>
          </fieldset>
          <fieldset className="space-y-2 text-xs">
            <legend className="font-medium text-slate-700 dark:text-neutral-300">
              {t('contentTools.reset.schedulingHandlingLabel')}
            </legend>
            <label className="flex items-start gap-2 text-slate-800 dark:text-neutral-200">
              <input
                type="radio"
                name="schedulingHandling"
                checked={schedulingHandling === 'keep'}
                onChange={() => setSchedulingHandling('keep')}
              />
              <span>{t('contentTools.reset.schedulingHandlingKeep')}</span>
            </label>
            <label className="flex items-start gap-2 text-slate-800 dark:text-neutral-200">
              <input
                type="radio"
                name="schedulingHandling"
                checked={schedulingHandling === 'clear'}
                onChange={() => setSchedulingHandling('clear')}
              />
              <span>{t('contentTools.reset.schedulingHandlingClear')}</span>
            </label>
          </fieldset>
          {preview ? (
            <p className="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-950 dark:bg-amber-950/40 dark:text-amber-100">
              {t('contentTools.reset.dryRunPreview', { count: affected })}
            </p>
          ) : null}
          {needsTypedConfirm ? (
            <label className="block text-sm">
              <span className="text-xs font-medium text-rose-700 dark:text-rose-300">
                {t('contentTools.reset.typeToConfirm')}
              </span>
              <input
                className="mt-1 w-full rounded-md border border-rose-300 px-2 py-1.5 text-sm dark:border-rose-800 dark:bg-neutral-950"
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                autoComplete="off"
              />
            </label>
          ) : null}
          {jobId ? (
            <ResetJobProgress
              courseCode={courseCode}
              jobId={jobId}
              onDone={() => {
                onCompleted?.()
                onClose()
              }}
            />
          ) : null}
          {error ? (
            <p className="text-xs text-rose-600" role="alert">
              {error}
            </p>
          ) : null}
        </div>
        <div className="mt-4 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            autoFocus
            className="rounded-md border border-slate-200 px-3 py-1.5 text-xs font-medium dark:border-neutral-600"
            onClick={onClose}
            disabled={busy && !jobId}
          >
            {t('contentTools.authoring.cancel')}
          </button>
          {!jobId ? (
            <button
              type="button"
              className="rounded-md bg-rose-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
              disabled={busy || !preview || !confirmOk || affected === 0}
              onClick={() => void execute()}
            >
              {t('contentTools.reset.confirm')}
            </button>
          ) : null}
        </div>
      </div>
    </div>
  )
}
