import { useEffect, useId, useMemo, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourseAssignments } from '../../../hooks/use-course-assignments'
import type { CourseGradingAgentTemplateSummary } from '../../../lib/courses-api'
import { AssignmentMultiPicker } from './assignment-multi-picker'

export type CreateGradingAgentFromTemplateResult = {
  name: string
  assignmentIds: string[]
}

type CreateGradingAgentFromTemplateModalProps = {
  open: boolean
  courseCode: string
  template: CourseGradingAgentTemplateSummary | null
  existingAgentItemIds: Set<string>
  onClose: () => void
  onCreate: (result: CreateGradingAgentFromTemplateResult) => void | Promise<void>
}

export function CreateGradingAgentFromTemplateModal({
  open,
  courseCode,
  template,
  existingAgentItemIds,
  onClose,
  onCreate,
}: CreateGradingAgentFromTemplateModalProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const nameInputId = useId()
  const [name, setName] = useState('')
  const [selectedAssignmentIds, setSelectedAssignmentIds] = useState<Set<string>>(() => new Set())
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { assignments, loading: assignmentsLoading } = useCourseAssignments(courseCode, open)

  const availableAssignments = useMemo(
    () => assignments.filter((assignment) => !existingAgentItemIds.has(assignment.id)),
    [assignments, existingAgentItemIds],
  )

  useEffect(() => {
    if (!open || !template) return
    setName(template.name)
    setSelectedAssignmentIds(new Set())
    setSubmitting(false)
    setError(null)
  }, [open, template])

  useEffect(() => {
    if (!open) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !submitting) onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, submitting, onClose])

  if (!open || !template) return null

  const canCreate =
    !submitting && name.trim() !== '' && selectedAssignmentIds.size > 0 && availableAssignments.length > 0

  const submit = async () => {
    if (!canCreate) return
    setSubmitting(true)
    setError(null)
    try {
      const orderedIds = availableAssignments
        .filter((assignment) => selectedAssignmentIds.has(assignment.id))
        .map((assignment) => assignment.id)
      await onCreate({
        name: name.trim(),
        assignmentIds: orderedIds,
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : t('gradingAgent.settings.fromTemplate.error'))
      setSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-[520] flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onClick={() => {
        if (!submitting) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {t('gradingAgent.settings.fromTemplate.title', { template: template.name })}
        </h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
          {t('gradingAgent.settings.fromTemplate.description')}
        </p>

        <div className="mt-5 space-y-4">
          <div>
            <label
              htmlFor={nameInputId}
              className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-neutral-400"
            >
              {t('gradingAgent.settings.fromTemplate.nameLabel')}
            </label>
            <input
              id={nameInputId}
              type="text"
              value={name}
              disabled={submitting}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('gradingAgent.settings.fromTemplate.namePlaceholder')}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            />
          </div>

          <div>
            <p className="mb-1.5 text-xs font-medium text-slate-600 dark:text-neutral-400">
              {t('gradingAgent.settings.fromTemplate.assignmentsLabel')}
            </p>
            <AssignmentMultiPicker
              assignments={availableAssignments}
              selectedIds={selectedAssignmentIds}
              disabled={submitting}
              loading={assignmentsLoading}
              filterPlaceholder={t('gradingAgent.canvas.inspector.activityAssignmentFilter')}
              emptyLabel={
                assignmentsLoading
                  ? t('gradingAgent.settings.create.loadingAssignments')
                  : t('gradingAgent.settings.create.noAssignments')
              }
              noMatchLabel={t('gradingAgent.canvas.inspector.activityAssignmentNoMatch')}
              onChange={setSelectedAssignmentIds}
            />
            {selectedAssignmentIds.size === 1 ? (
              <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.settings.fromTemplate.singleAssignmentHint')}
              </p>
            ) : selectedAssignmentIds.size > 1 ? (
              <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.settings.fromTemplate.multiAssignmentHint')}
              </p>
            ) : null}
          </div>
        </div>

        {error ? (
          <p className="mt-4 text-sm text-rose-600 dark:text-rose-400" role="alert">
            {error}
          </p>
        ) : null}

        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            disabled={submitting}
            onClick={onClose}
            className="rounded-lg px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 disabled:opacity-60 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            {t('gradingAgent.save.templateCancel')}
          </button>
          <button
            type="button"
            disabled={!canCreate}
            onClick={() => void submit()}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                {t('gradingAgent.settings.fromTemplate.creating')}
              </>
            ) : (
              t('gradingAgent.settings.fromTemplate.create')
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
