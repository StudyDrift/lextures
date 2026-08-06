import { lazy, Suspense, useEffect, useId, useMemo, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourseAssignments } from '../../../hooks/use-course-assignments'
import { useCourseQuizzes } from '../../../hooks/use-course-quizzes'
import type { CourseGradingAgentTemplateSummary, GradingAgentItemKind } from '../../../lib/courses-api'
import { AssignmentPicker } from './assignment-picker'

const QuizPicker = lazy(() => import('./quiz-picker').then((m) => ({ default: m.QuizPicker })))

export type CreateGradingAgentSource = 'template' | 'assignment' | 'asTemplate'

export type CreateGradingAgentResult = {
  source: CreateGradingAgentSource
  itemKind: GradingAgentItemKind
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
  const itemKindAssignmentId = useId()
  const itemKindQuizId = useId()
  const [source, setSource] = useState<CreateGradingAgentSource>('assignment')
  const [itemKind, setItemKind] = useState<GradingAgentItemKind>('assignment')
  const [templateId, setTemplateId] = useState('')
  const [templateName, setTemplateName] = useState('')
  const [assignmentId, setAssignmentId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { assignments, loading: assignmentsLoading } = useCourseAssignments(
    courseCode,
    open && itemKind === 'assignment',
  )
  const { quizzes, loading: quizzesLoading } = useCourseQuizzes(courseCode, open && itemKind === 'quiz')

  const availableAssignments = useMemo(
    () => assignments.filter((assignment) => !existingAgentItemIds.has(assignment.id)),
    [assignments, existingAgentItemIds],
  )
  const availableQuizzes = useMemo(
    () => quizzes.filter((quiz) => !existingAgentItemIds.has(quiz.id)),
    [quizzes, existingAgentItemIds],
  )

  const availableItems = itemKind === 'quiz' ? availableQuizzes : availableAssignments
  const itemsLoading = itemKind === 'quiz' ? quizzesLoading : assignmentsLoading

  const hasTemplates = templates.length > 0
  const hasItems = availableItems.length > 0

  useEffect(() => {
    if (!open) return
    setSource('assignment')
    setItemKind('assignment')
    setTemplateId(templates[0]?.id ?? '')
    setTemplateName('')
    setAssignmentId('')
    setSubmitting(false)
    setError(null)
  }, [open, hasTemplates, templates])

  useEffect(() => {
    if (!open || itemsLoading) return
    if (assignmentId && availableItems.some((item) => item.id === assignmentId)) return
    setAssignmentId(availableItems[0]?.id ?? '')
  }, [open, itemsLoading, assignmentId, availableItems])

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
      : hasItems &&
        assignmentId !== '' &&
        (source === 'assignment' || (hasTemplates && templateId !== '')))

  const submit = async () => {
    if (!canContinue) return
    setSubmitting(true)
    setError(null)
    try {
      await onContinue({
        source,
        itemKind,
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
        className="w-full max-w-md rounded-2xl bg-surface-raised p-6 shadow-xl dark:bg-surface-raised"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id={titleId} className="text-lg font-semibold text-fg-default">
          {t('gradingAgent.settings.create.title')}
        </h2>
        <p className="mt-2 text-sm text-fg-muted">
          {t('gradingAgent.settings.create.description')}
        </p>

        <fieldset className="mt-5 space-y-3">
          <legend className="sr-only">{t('gradingAgent.settings.create.sourceLegend')}</legend>
          <label
            htmlFor={sourceTemplateId}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-[background-color,color,border-color] ${ source === 'template' ? 'border-indigo-400 bg-indigo-50/70 dark:border-indigo-500 dark:bg-indigo-950/30' : 'border-border-default hover:border-border-strong dark:hover:border-border-default' } ${!hasTemplates ? 'cursor-not-allowed opacity-50' : ''}`}
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
              <span className="block text-sm font-medium text-fg-default">
                {t('gradingAgent.settings.create.sourceTemplate')}
              </span>
              <span className="mt-0.5 block text-xs text-fg-muted">
                {hasTemplates
                  ? t('gradingAgent.settings.create.sourceTemplateHelp')
                  : t('gradingAgent.settings.create.noTemplates')}
              </span>
            </span>
          </label>

          <label
            htmlFor={sourceAssignmentId}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-[background-color,color,border-color] ${ source === 'assignment' ? 'border-indigo-400 bg-indigo-50/70 dark:border-indigo-500 dark:bg-indigo-950/30' : 'border-border-default hover:border-border-strong dark:hover:border-border-default' }`}
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
              <span className="block text-sm font-medium text-fg-default">
                {t('gradingAgent.settings.create.sourceAssignment')}
              </span>
              <span className="mt-0.5 block text-xs text-fg-muted">
                {t('gradingAgent.settings.create.sourceAssignmentHelp')}
              </span>
            </span>
          </label>

          <label
            htmlFor={sourceAsTemplateId}
            className={`flex cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-[background-color,color,border-color] ${ source === 'asTemplate' ? 'border-indigo-400 bg-indigo-50/70 dark:border-indigo-500 dark:bg-indigo-950/30' : 'border-border-default hover:border-border-strong dark:hover:border-border-default' }`}
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
              <span className="block text-sm font-medium text-fg-default">
                {t('gradingAgent.settings.create.sourceAsTemplate')}
              </span>
              <span className="mt-0.5 block text-xs text-fg-muted">
                {t('gradingAgent.settings.create.sourceAsTemplateHelp')}
              </span>
            </span>
          </label>
        </fieldset>

        {source === 'template' && hasTemplates ? (
          <div className="mt-4">
            <label
              htmlFor={templateSelectId}
              className="mb-1.5 block text-xs font-medium text-fg-muted"
            >
              {t('gradingAgent.settings.create.templateLabel')}
            </label>
            <select
              id={templateSelectId}
              value={templateId}
              disabled={submitting}
              onChange={(e) => setTemplateId(e.target.value)}
              className="w-full rounded-lg border border-border-strong bg-surface-raised px-3 py-2 text-sm text-fg-default focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default"
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
              className="mb-1.5 block text-xs font-medium text-fg-muted"
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
              className="w-full rounded-lg border border-border-strong bg-surface-raised px-3 py-2 text-sm text-fg-default focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default"
            />
          </div>
        ) : (
          <>
            <fieldset className="mt-4">
              <legend className="mb-1.5 block text-xs font-medium text-fg-muted">
                {t('gradingAgent.settings.create.itemKindLegend')}
              </legend>
              <div className="flex gap-2">
                <label
                  htmlFor={itemKindAssignmentId}
                  className={`flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-[background-color,border-color] ${ itemKind === 'assignment' ? 'border-indigo-400 bg-indigo-50/70 text-indigo-900 dark:border-indigo-500 dark:bg-indigo-950/30 dark:text-indigo-100' : 'border-border-default text-fg-muted hover:border-border-strong dark:text-fg-default' }`}
                >
                  <input
                    id={itemKindAssignmentId}
                    type="radio"
                    name="create-grading-agent-item-kind"
                    value="assignment"
                    checked={itemKind === 'assignment'}
                    disabled={submitting}
                    onChange={() => setItemKind('assignment')}
                    className="sr-only"
                  />
                  {t('gradingAgent.settings.create.itemKindAssignment')}
                </label>
                <label
                  htmlFor={itemKindQuizId}
                  className={`flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-[background-color,border-color] ${ itemKind === 'quiz' ? 'border-indigo-400 bg-indigo-50/70 text-indigo-900 dark:border-indigo-500 dark:bg-indigo-950/30 dark:text-indigo-100' : 'border-border-default text-fg-muted hover:border-border-strong dark:text-fg-default' }`}
                >
                  <input
                    id={itemKindQuizId}
                    type="radio"
                    name="create-grading-agent-item-kind"
                    value="quiz"
                    checked={itemKind === 'quiz'}
                    disabled={submitting}
                    onChange={() => setItemKind('quiz')}
                    className="sr-only"
                  />
                  {t('gradingAgent.settings.create.itemKindQuiz')}
                </label>
              </div>
            </fieldset>

            <div className="mt-4">
              <label className="mb-1.5 block text-xs font-medium text-fg-muted">
                {itemKind === 'quiz'
                  ? t('gradingAgent.settings.create.quizLabel')
                  : t('gradingAgent.settings.create.assignmentLabel')}
              </label>
              {itemsLoading ? (
                <p className="flex items-center gap-2 text-sm text-fg-muted">
                  <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                  {itemKind === 'quiz'
                    ? t('gradingAgent.settings.create.loadingQuizzes')
                    : t('gradingAgent.settings.create.loadingAssignments')}
                </p>
              ) : !hasItems ? (
                <p className="text-sm text-fg-muted">
                  {itemKind === 'quiz'
                    ? t('gradingAgent.settings.create.noQuizzes')
                    : t('gradingAgent.settings.create.noAssignments')}
                </p>
              ) : itemKind === 'quiz' ? (
                <Suspense fallback={<p className="text-sm text-fg-muted">{t('gradingAgent.settings.create.loadingQuizzes')}</p>}>
                  <QuizPicker
                    quizzes={availableQuizzes}
                    value={assignmentId}
                    disabled={submitting}
                    searchPlaceholder={t('gradingAgent.settings.create.quizSearchPlaceholder')}
                    emptyLabel={t('gradingAgent.settings.create.noQuizzes')}
                    noMatchLabel={t('gradingAgent.settings.create.quizNoMatch')}
                    moduleFallbackLabel={t('gradingAgent.settings.create.quizModuleUnknown')}
                    onChange={setAssignmentId}
                  />
                </Suspense>
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
          </>
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
            className="rounded-lg px-3 py-1.5 text-sm text-fg-muted hover:bg-surface-sunken disabled:opacity-60 dark:text-fg-muted dark:hover:bg-surface-overlay"
          >
            {t('gradingAgent.save.templateCancel')}
          </button>
          <button
            type="button"
            disabled={!canContinue}
            onClick={() => void submit()}
            className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-3 py-1.5 text-sm font-medium text-white hover:bg-accent disabled:opacity-50"
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