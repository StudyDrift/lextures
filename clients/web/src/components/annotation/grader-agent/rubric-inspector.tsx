import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { fetchModuleAssignment } from '../../../lib/courses-api'
import type { RubricDefinition } from '../../../lib/courses-api'
import { useCourseAssignments } from '../../../hooks/use-course-assignments'
import { AssignmentPicker } from './assignment-picker'
import {
  createDefaultInlineRubric,
  inlineRubricFromData,
  rubricCriteriaSummary,
  rubricLibraryAssignmentItemId,
  rubricSourceMode,
  updateInlineCriterion,
} from './rubric-node-data'
import type { RubricNodeData, RubricSourceMode } from './types'

const SOURCES: RubricSourceMode[] = ['assignment', 'library', 'inline']

type RubricInspectorProps = {
  courseCode: string
  assignmentTitle?: string
  assignmentHasRubric?: boolean
  maxPoints?: number | null
  data: Record<string, unknown>
  onChange: (patch: Partial<RubricNodeData>) => void
  onDelete: () => void
  onLibraryRubricResolved?: (itemId: string, hasRubric: boolean) => void
  fieldClass: string
}

export function RubricInspector({
  courseCode,
  assignmentTitle,
  assignmentHasRubric = true,
  maxPoints,
  data,
  onChange,
  onDelete,
  onLibraryRubricResolved,
  fieldClass,
}: RubricInspectorProps) {
  const { t } = useTranslation('common')
  const source = rubricSourceMode(data)
  const libraryItemId = rubricLibraryAssignmentItemId(data)
  const inlineRubric = inlineRubricFromData(data)
  const showLibraryPicker = source === 'library'

  const { assignments, loading: assignmentsLoading, error: assignmentsError } = useCourseAssignments(
    courseCode,
    showLibraryPicker,
  )
  const [libraryLoading, setLibraryLoading] = useState(false)
  const [libraryError, setLibraryError] = useState<string | null>(null)
  const [libraryRubric, setLibraryRubric] = useState<RubricDefinition | null>(null)

  useEffect(() => {
    if (source !== 'library' || !libraryItemId) {
      setLibraryRubric(null)
      setLibraryError(null)
      setLibraryLoading(false)
      return
    }
    let cancelled = false
    setLibraryLoading(true)
    setLibraryError(null)
    void fetchModuleAssignment(courseCode, libraryItemId)
      .then((payload) => {
        if (cancelled) return
        const hasRubric = Boolean(payload.rubric?.criteria?.length)
        setLibraryRubric(payload.rubric)
        onLibraryRubricResolved?.(libraryItemId, hasRubric)
        if (!hasRubric) {
          setLibraryError(t('gradingAgent.canvas.inspector.rubricLibraryMissing'))
        }
      })
      .catch((err: unknown) => {
        if (cancelled) return
        setLibraryRubric(null)
        onLibraryRubricResolved?.(libraryItemId, false)
        setLibraryError(
          err instanceof Error ? err.message : t('gradingAgent.canvas.inspector.rubricLibraryError'),
        )
      })
      .finally(() => {
        if (!cancelled) setLibraryLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, libraryItemId, onLibraryRubricResolved, source, t])

  const pickerAssignments = assignments.some((assignment) => assignment.id === libraryItemId)
    ? assignments
    : libraryItemId
      ? [
          ...assignments,
          {
            id: libraryItemId,
            title: assignmentTitle?.trim() || t('gradingAgent.canvas.inspector.rubricAssignmentCurrent'),
          },
        ]
      : assignments

  const summaryRubric =
    source === 'inline'
      ? inlineRubric
      : source === 'library'
        ? libraryRubric
        : null

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p>{t('gradingAgent.canvas.inspector.rubricHelp')}</p>
      <fieldset>
        <legend className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.rubricSource')}
        </legend>
        <div className="flex flex-col gap-1.5">
          {SOURCES.map((value) => (
            <label key={value} className="flex items-center gap-2">
              <input
                type="radio"
                name="rubric-source"
                value={value}
                checked={source === value}
                onChange={() => {
                  const patch: Partial<RubricNodeData> = { source: value }
                  if (value === 'inline' && !inlineRubricFromData(data)) {
                    patch.rubric = createDefaultInlineRubric(maxPoints ?? 10)
                  }
                  onChange(patch)
                }}
              />
              <span>{t(`gradingAgent.canvas.inspector.rubricSource.${value}`)}</span>
            </label>
          ))}
        </div>
      </fieldset>
      {source === 'assignment' ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          {assignmentHasRubric
            ? t('gradingAgent.canvas.inspector.rubricAssignmentHelp')
            : t('gradingAgent.canvas.inspector.rubricAssignmentMissing')}
        </p>
      ) : null}
      {source === 'library' ? (
        <label className="block">
          <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
            {t('gradingAgent.canvas.inspector.rubricLibraryAssignment')}
          </span>
          <AssignmentPicker
            assignments={pickerAssignments}
            value={libraryItemId}
            loading={assignmentsLoading}
            filterPlaceholder={t('gradingAgent.canvas.inspector.rubricLibraryFilter')}
            emptyLabel={t('gradingAgent.canvas.inspector.rubricLibraryEmpty')}
            noMatchLabel={t('gradingAgent.canvas.inspector.rubricLibraryNoMatch')}
            onChange={(assignmentId) => onChange({ rubricAssignmentItemId: assignmentId })}
          />
          <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.inspector.rubricLibraryHelp')}
          </p>
          {assignmentsError ? (
            <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{assignmentsError}</p>
          ) : null}
          {libraryLoading ? (
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.canvas.inspector.rubricLibraryLoading')}
            </p>
          ) : null}
          {libraryError ? (
            <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{libraryError}</p>
          ) : null}
        </label>
      ) : null}
      {source === 'inline' ? (
        <div className="space-y-2">
          <p className="font-medium text-slate-800 dark:text-neutral-100">
            {t('gradingAgent.canvas.inspector.rubricInlineTitle')}
          </p>
          {(inlineRubric?.criteria ?? []).map((criterion, index) => (
            <label key={criterion.id} className="block">
              <span className="mb-1 block text-xs font-medium text-slate-600 dark:text-neutral-400">
                {t('gradingAgent.canvas.inspector.rubricInlineCriterion', { index: index + 1 })}
              </span>
              <input
                type="text"
                value={criterion.title}
                onChange={(e) => {
                  const current = inlineRubric ?? createDefaultInlineRubric(maxPoints ?? 10)
                  onChange({
                    rubric: updateInlineCriterion(current, index, { title: e.target.value }),
                  })
                }}
                className={fieldClass}
              />
            </label>
          ))}
          <button
            type="button"
            onClick={() => {
              const current = inlineRubric ?? createDefaultInlineRubric(maxPoints ?? 10)
              const n = current.criteria.length + 1
              onChange({
                rubric: {
                  ...current,
                  criteria: [
                    ...current.criteria,
                    {
                      id: crypto.randomUUID(),
                      title: `Criterion ${n}`,
                      description: null,
                      levels: [{ label: 'Rating 1', points: maxPoints ?? 10, description: null }],
                    },
                  ],
                },
              })
            }}
            className="text-xs font-medium text-indigo-700 hover:underline dark:text-indigo-300"
          >
            {t('gradingAgent.canvas.inspector.rubricInlineAddCriterion')}
          </button>
        </div>
      ) : null}
      {summaryRubric?.criteria?.length ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.inspector.rubricCriteriaSummary', {
            count: summaryRubric.criteria.length,
            names: rubricCriteriaSummary(summaryRubric),
          })}
        </p>
      ) : null}
      <button
        type="button"
        onClick={onDelete}
        className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
      >
        {t('gradingAgent.canvas.inspector.deleteNode')}
      </button>
    </div>
  )
}