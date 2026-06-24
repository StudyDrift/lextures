import { useEffect, useId, useMemo, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourseAssignments } from '../../../hooks/use-course-assignments'
import type { CourseGradingAgentTemplateSummary } from '../../../lib/courses-api'
import { AssignmentPicker } from './assignment-picker'

export type CreateGradingAgentSource = 'template' | 'assignment' | 'asTemplate'

export type CreateGradingAgentResult = {
  source: CreateGradingAgentSource
  assignmentId?: string
  templateId?: string
  templateName?: string
}

type CreateGradingAgentModalProps = {
  open: boolean
  courseCode: string
  templates: CourseGradingAgentTemplateSummary[]
  existingAgentItemIds: Set<string>
  onClose: () => void
  onContinue: (result: CreateGradingAgentResult) => void | Promise<void>
}

export function CreateGradingAgentModal({
  open,
  courseCode,
  templates,
  existingAgentItemIds,
  onClose,
  onContinue,
}: CreateGradingAgentModalProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const sourceTemplateId = useId()
  const sourceAssignmentId = useId()
  const sourceAsTemplateId = useId()
  const templateSelectId = useId()
  const templateNameInputId = useId()
  const [source, setSource] = useState<CreateGradingAgentSource>('assignment')
  const [templateId, setTemplateId] = useState('')
  const [templateName, setTemplateName] = useState('')
  const [assignmentId, setAssignmentId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { assignments, loading: assignmentsLoading } = useCourseAssignments(courseCode, open)

  const availableAssignments = useMemo(
    () => assignments.filter((assignment) => !existingAgentItemIds.has(assignment.id)),
    [assignments, existingAgentItemIds],
  )

  const hasTemplates = templates.length > 0
  const hasAssignments = availableAssignments.length > 0

  useEffect(() => {
    if (!open) return
    setSource('assignment')
    setTemplateId(templates[0]?.id ?? '')
    setTemplateName('')
    setAssignmentId('')
    setSubmitting(false)
    setError(null)
  }, [open, hasTemplates, templates])

  useEffect(() => {
    if (!open || assignmentsLoading) return
    if (assignmentId && availableAssignments.some((a) => a.id === assignmentId)) return
    setAssignmentId(availableAssignments[0]?.id ?? '')
  }, [open, assignmentsLoading, assignmentId, availableAssignments])

  useEffect(() => {
    if (!open) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !submitting) onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, submitting, onClose])

  if (!open) return null

  const canContinue =
    !submitting &&
    (source === 'asTemplate'
      ? templateName.trim() !== ''
      : hasAssignments &&
        assignmentId !== '' &&
        (source === 'assignment' || (hasTemplates && templateId !== '')))

  const submit = async () => {
    if (!canContinue) return
    setSubmitting(true)
    setError(null)
    try {
      await onContinue({
        source,
        assignmentId: source === 'asTemplate' ? undefined : assignmentId,
        templateId: source === 'template' ? templateId : undefined,
        templateName: source === 'asTemplate' ? templateName.trim() : undefined,
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : t('gradingAgent.settings.create.error'))
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
          {t('gradingAgent.settings.create.title')}
        </h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
          {t('gradingAgent.settings.create.description')}
        </p>

        <fieldset className="mt-5 space-y-3">
          <legend className="sr-only">{t('gradingAgent.settings.create.sourceLegend')}</legend>
          <label
            htmlFor={sourceTemplateId}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-[background-color,color,border-color] ${
              source === 'template'
                ? 'border-indigo-400 bg-indigo-50/70 dark:border-indigo-500 dark:bg-indigo-950/30'
                : 'border-slate-200 hover:border-slate-300 dark:border-neutral-700 dark:hover:border-neutral-600'
            } ${!hasTemplates ? 'cursor-not-allowed opacity-50' : ''}`}
          >
            <input
              id={sourceTemplateId}
              type="radio"
              name="create-grading-agent-source"
              value="template"
              checked={source === 'template'}
              disabled={!hasTemplates || submitting}
              onChange={() => setSource('template')}
              className="mt-0.5"
            />
            <span className="min-w-0">
              <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                {t('gradingAgent.settings.create.sourceTemplate')}
              </span>
              <span className="mt-0.5 block text-xs text-slate-600 dark:text-neutral-400">
                {hasTemplates
                  ? t('gradingAgent.settings.create.sourceTemplateHelp')
                  : t('gradingAgent.settings.create.noTemplates')}
              </span>
            </span>
          </label>

          <label
            htmlFor={sourceAssignmentId}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-[background-color,color,border-color] ${
              source === 'assignment'
                ? 'border-indigo-400 bg-indigo-50/70 dark:border-indigo-500 dark:bg-indigo-950/30'
                : 'border-slate-200 hover:border-slate-300 dark:border-neutral-700 dark:hover:border-neutral-600'
            }`}
          >
            <input
              id={sourceAssignmentId}
              type="radio"
              name="create-grading-agent-source"
              value="assignment"
              checked={source === 'assignment'}
              disabled={submitting}
              onChange={() => setSource('assignment')}
              className="mt-0.5"
            />
            <span className="min-w-0">
              <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                {t('gradingAgent.settings.create.sourceAssignment')}
              </span>
              <span className="mt-0.5 block text-xs text-slate-600 dark:text-neutral-400">
                {t('gradingAgent.settings.create.sourceAssignmentHelp')}
              </span>
            </span>
          </label>

          <label
            htmlFor={sourceAsTemplateId}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-[background-color,color,border-color] ${
              source === 'asTemplate'
                ? 'border-indigo-400 bg-indigo-50/70 dark:border-indigo-500 dark:bg-indigo-950/30'
                : 'border-slate-200 hover:border-slate-300 dark:border-neutral-700 dark:hover:border-neutral-600'
            }`}
          >
            <input
              id={sourceAsTemplateId}
              type="radio"
              name="create-grading-agent-source"
              value="asTemplate"
              checked={source === 'asTemplate'}
              disabled={submitting}
              onChange={() => setSource('asTemplate')}
              className="mt-0.5"
            />
            <span className="min-w-0">
              <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                {t('gradingAgent.settings.create.sourceAsTemplate')}
              </span>
              <span className="mt-0.5 block text-xs text-slate-600 dark:text-neutral-400">
                {t('gradingAgent.settings.create.sourceAsTemplateHelp')}
              </span>
            </span>
          </label>
        </fieldset>

        {source === 'template' && hasTemplates ? (
          <div className="mt-4">
            <label
              htmlFor={templateSelectId}
              className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-neutral-400"
            >
              {t('gradingAgent.settings.create.templateLabel')}
            </label>
            <select
              id={templateSelectId}
              value={templateId}
              disabled={submitting}
              onChange={(e) => setTemplateId(e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            >
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}

        {source === 'asTemplate' ? (
          <div className="mt-4">
            <label
              htmlFor={templateNameInputId}
              className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-neutral-400"
            >
              {t('gradingAgent.settings.create.newTemplateNameLabel')}
            </label>
            <input
              id={templateNameInputId}
              type="text"
              value={templateName}
              disabled={submitting}
              onChange={(e) => setTemplateName(e.target.value)}
              placeholder={t('gradingAgent.save.templateNamePlaceholder')}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            />
          </div>
        ) : (
        <div className="mt-4">
          <label className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-neutral-400">
            {t('gradingAgent.settings.create.assignmentLabel')}
          </label>
          {assignmentsLoading ? (
            <p className="flex items-center gap-2 text-sm text-slate-500 dark:text-neutral-400">
              <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
              {t('gradingAgent.settings.create.loadingAssignments')}
            </p>
          ) : !hasAssignments ? (
            <p className="text-sm text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.settings.create.noAssignments')}
            </p>
          ) : (
            <AssignmentPicker
              assignments={availableAssignments}
              value={assignmentId}
              disabled={submitting}
              loading={assignmentsLoading}
              filterPlaceholder={t('gradingAgent.canvas.inspector.activityAssignmentFilter')}
              emptyLabel={t('gradingAgent.canvas.inspector.activityAssignmentEmpty')}
              noMatchLabel={t('gradingAgent.canvas.inspector.activityAssignmentNoMatch')}
              onChange={setAssignmentId}
            />
          )}
        </div>
        )}

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
            disabled={!canContinue}
            onClick={() => void submit()}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                {t('gradingAgent.settings.create.continuing')}
              </>
            ) : (
              t('gradingAgent.settings.create.continue')
            )}
          </button>
        </div>
      </div>
    </div>
  )
}