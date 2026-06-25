import { useEffect, useId, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchCourseSections,
  fetchEnrollmentGroupsTree,
  fetchGraderAgentRunTarget,
  type CourseSection,
  type GraderAgentRunFilter,
  type ModuleAssignmentSubmissionApi,
} from '../../../lib/courses-api'
import type { RunScope } from './use-grader-agent-workflow'

export type GraderAgentRunTarget = 'all' | 'section' | 'group' | 'selected'

export type RunAgentFilterState = {
  target: GraderAgentRunTarget
  sectionId: string | null
  groupId: string | null
  selectedSubmissionIds: string[]
}

export const defaultRunAgentFilterState: RunAgentFilterState = {
  target: 'all',
  sectionId: null,
  groupId: null,
  selectedSubmissionIds: [],
}

export function runFilterFromState(state: RunAgentFilterState): GraderAgentRunFilter | undefined {
  switch (state.target) {
    case 'section':
      return state.sectionId ? { sectionId: state.sectionId } : undefined
    case 'group':
      return state.groupId ? { groupId: state.groupId } : undefined
    case 'selected':
      return state.selectedSubmissionIds.length > 0 ? { submissionIds: state.selectedSubmissionIds } : undefined
    default:
      return undefined
  }
}

type RunAgentFilterPickerProps = {
  enabled: boolean
  courseCode: string
  itemId: string
  runScope: RunScope
  confirmOverwrite: boolean
  filterState: RunAgentFilterState
  setFilterState: (next: RunAgentFilterState | ((prev: RunAgentFilterState) => RunAgentFilterState)) => void
  submissions: ModuleAssignmentSubmissionApi[]
  currentSubmissionId: string | null
}

function submissionPickerLabel(submission: ModuleAssignmentSubmissionApi): string {
  if (submission.blindLabel) return submission.blindLabel
  if (submission.submittedByDisplayName) return submission.submittedByDisplayName
  if (submission.id) return submission.id.slice(0, 8)
  return 'Submission'
}

export function RunAgentFilterPicker({
  enabled,
  courseCode,
  itemId,
  runScope,
  confirmOverwrite,
  filterState,
  setFilterState,
  submissions,
  currentSubmissionId,
}: RunAgentFilterPickerProps) {
  const { t } = useTranslation('common')
  const targetGroupId = useId()
  const [sections, setSections] = useState<CourseSection[]>([])
  const [groups, setGroups] = useState<Array<{ id: string; name: string; setName: string }>>([])
  const [targetSummary, setTargetSummary] = useState<string | null>(null)
  const [targetLoading, setTargetLoading] = useState(false)

  useEffect(() => {
    if (!enabled || !courseCode.trim()) {
      setSections([])
      setGroups([])
      return
    }
    let cancelled = false
    void Promise.all([
      fetchCourseSections(courseCode).catch(() => []),
      fetchEnrollmentGroupsTree(courseCode).catch(() => ({ groupSets: [] })),
    ]).then(([sectionList, tree]) => {
      if (cancelled) return
      setSections(sectionList.filter((s) => s.status === 'active'))
      setGroups(
        tree.groupSets.flatMap((set) =>
          set.groups.map((g) => ({ id: g.id, name: g.name, setName: set.name })),
        ),
      )
    })
    return () => {
      cancelled = true
    }
  }, [enabled, courseCode])

  const gradableSubmissions = useMemo(
    () => submissions.filter((s) => typeof s.id === 'string' && s.id.length > 0),
    [submissions],
  )

  useEffect(() => {
    if (!enabled || !courseCode.trim() || !itemId.trim()) {
      setTargetSummary(null)
      return
    }
    const filter = runFilterFromState(filterState)
    if (filterState.target === 'selected' && (!filter?.submissionIds || filter.submissionIds.length === 0)) {
      setTargetSummary(t('gradingAgent.run.filter.selectSubmissions'))
      return
    }
    if (filterState.target === 'section' && !filterState.sectionId) {
      setTargetSummary(t('gradingAgent.run.filter.selectSection'))
      return
    }
    if (filterState.target === 'group' && !filterState.groupId) {
      setTargetSummary(t('gradingAgent.run.filter.selectGroup'))
      return
    }
    let cancelled = false
    setTargetLoading(true)
    void fetchGraderAgentRunTarget(courseCode, itemId, {
      scope: runScope,
      overwrite: runScope === 'all' && confirmOverwrite,
      submissionId: runScope === 'current' ? currentSubmissionId ?? undefined : undefined,
      filter,
    })
      .then((res) => {
        if (!cancelled) setTargetSummary(res.targetSummary)
      })
      .catch(() => {
        if (!cancelled) setTargetSummary(null)
      })
      .finally(() => {
        if (!cancelled) setTargetLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [enabled, courseCode, itemId, runScope, confirmOverwrite, filterState, currentSubmissionId, t])

  if (!enabled) return null

  const targets: GraderAgentRunTarget[] = ['all', 'section', 'group', 'selected']

  return (
    <fieldset className="mt-3">
      <legend id={targetGroupId} className="mb-2 text-xs font-medium text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.run.filter.label')}
      </legend>
      <div
        role="radiogroup"
        aria-labelledby={targetGroupId}
        className="overflow-hidden rounded-xl bg-slate-50 ring-1 ring-black/5 dark:bg-neutral-800/60 dark:ring-white/10"
      >
        {targets.map((target, index) => {
          const selected = filterState.target === target
          return (
            <label
              key={target}
              className={`flex min-h-10 cursor-pointer items-center gap-3 px-3 py-2.5 text-sm transition-colors ${
                index > 0 ? 'border-t border-slate-200/80 dark:border-neutral-700/80' : ''
              } ${
                selected
                  ? 'bg-white text-indigo-900 dark:bg-neutral-900 dark:text-indigo-100'
                  : 'text-slate-700 hover:bg-white/70 dark:text-neutral-300 dark:hover:bg-neutral-900/50'
              }`}
            >
              <input
                type="radio"
                name="grader-agent-run-target"
                value={target}
                checked={selected}
                onChange={() =>
                  setFilterState((prev) => ({
                    ...prev,
                    target,
                  }))
                }
                className="sr-only"
              />
              <span className="min-w-0 flex-1 leading-snug">{t(`gradingAgent.run.filter.target.${target}`)}</span>
            </label>
          )
        })}
      </div>
      {filterState.target === 'section' ? (
        <label className="mt-2 block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1 block text-xs font-medium text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.run.filter.sectionLabel')}
          </span>
          <select
            className="w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            value={filterState.sectionId ?? ''}
            onChange={(e) =>
              setFilterState((prev) => ({
                ...prev,
                sectionId: e.target.value || null,
              }))
            }
          >
            <option value="">{t('gradingAgent.run.filter.sectionPlaceholder')}</option>
            {sections.map((section) => (
              <option key={section.id} value={section.id}>
                {section.name?.trim() || section.sectionCode}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {filterState.target === 'group' ? (
        <label className="mt-2 block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1 block text-xs font-medium text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.run.filter.groupLabel')}
          </span>
          <select
            className="w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            value={filterState.groupId ?? ''}
            onChange={(e) =>
              setFilterState((prev) => ({
                ...prev,
                groupId: e.target.value || null,
              }))
            }
          >
            <option value="">{t('gradingAgent.run.filter.groupPlaceholder')}</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.setName}: {group.name}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {filterState.target === 'selected' ? (
        <div className="mt-2 max-h-36 overflow-y-auto rounded-xl bg-slate-50 p-2 ring-1 ring-black/5 dark:bg-neutral-800/60 dark:ring-white/10">
          {gradableSubmissions.length === 0 ? (
            <p className="text-xs text-slate-500 dark:text-neutral-400">{t('gradingAgent.run.filter.noSubmissions')}</p>
          ) : (
            gradableSubmissions.map((submission) => {
              const id = submission.id as string
              const checked = filterState.selectedSubmissionIds.includes(id)
              return (
                <label
                  key={id}
                  className="flex min-h-9 cursor-pointer items-center gap-2 px-1 text-sm text-slate-700 dark:text-neutral-200"
                >
                  <input
                    type="checkbox"
                    className="size-4"
                    checked={checked}
                    onChange={(e) => {
                      setFilterState((prev) => {
                        const next = new Set(prev.selectedSubmissionIds)
                        if (e.target.checked) next.add(id)
                        else next.delete(id)
                        return { ...prev, selectedSubmissionIds: [...next] }
                      })
                    }}
                  />
                  <span className="min-w-0 truncate">{submissionPickerLabel(submission)}</span>
                </label>
              )
            })
          )}
        </div>
      ) : null}
      {targetSummary ? (
        <p
          className="mt-2 text-xs text-slate-600 dark:text-neutral-400"
          aria-live="polite"
          aria-busy={targetLoading}
        >
          {targetLoading ? t('gradingAgent.run.filter.loadingTarget') : targetSummary}
        </p>
      ) : null}
    </fieldset>
  )
}
