import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Search, Trash2 } from 'lucide-react'
import { ConfirmDialog } from '../../components/confirm-dialog'
import {
  CreateGradingAgentModal,
  type CreateGradingAgentResult,
} from '../../components/annotation/grader-agent/create-grading-agent-modal'
import {
  CreateGradingAgentFromTemplateModal,
  type CreateGradingAgentFromTemplateResult,
} from '../../components/annotation/grader-agent/create-grading-agent-from-template-modal'
import { cloneGraderAgentTemplateToAssignments } from '../../components/annotation/grader-agent/clone-grader-agent-template'
import { GraderAgentWorkflowModal } from '../../components/annotation/grader-agent/grader-agent-workflow-modal'
import type {
  GraderAgentTemplateMode,
  GraderAgentWorkflowSeed,
} from '../../components/annotation/grader-agent/use-grader-agent-workflow'
import {
  fetchCourseGradingAgentTemplates,
  fetchCourseGradingAgents,
  fetchGraderAgentTemplate,
  fetchModuleAssignment,
  fetchModuleQuiz,
  deleteGraderAgentTemplate,
  postGraderAgentTemplate,
  type CourseGradingAgentSummary,
  type CourseGradingAgentTemplateSummary,
  type GradingAgentItemKind,
  type QuizQuestion,
  type RubricDefinition,
} from '../../lib/courses-api'
import type { QuizQuestionSlot } from '../../components/annotation/grader-agent/quiz-question-slots'
import { formatAbsolute } from '../../lib/format-datetime'
import { usePlatformFeatures } from '../../context/platform-features-context'

type CourseGradingAgentsSectionProps = {
  courseCode: string
  createModalOpen: boolean
  onCreateModalOpenChange: (open: boolean) => void
}

type OpenAgentState = {
  itemId: string
  itemKind: GradingAgentItemKind
  assignmentTitle: string
  rubric: RubricDefinition | null
  maxPoints: number | null
  quizQuestionSlots: QuizQuestionSlot[]
  quizQuestions: QuizQuestion[]
  seedWorkflow: GraderAgentWorkflowSeed | null
}

type OpenTemplateState = {
  templateMode: GraderAgentTemplateMode
  seedWorkflow: GraderAgentWorkflowSeed | null
}

type AgentSortKey = 'assignmentTitle' | 'updatedAt'
type TemplateSortKey = 'name' | 'updatedAt'
type SortDir = 'ascending' | 'descending'

function statusLabel(
  status: CourseGradingAgentSummary['status'],
  t: (key: string) => string,
): string {
  if (status === 'accepted') return t('gradingAgent.settings.status.accepted')
  if (status === 'archived') return t('gradingAgent.settings.status.archived')
  return t('gradingAgent.settings.status.draft')
}

function statusClass(status: CourseGradingAgentSummary['status']): string {
  if (status === 'accepted') {
    return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200'
  }
  if (status === 'archived') {
    return 'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
  }
  return 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200'
}

export function CourseGradingAgentsSection({
  courseCode,
  createModalOpen,
  onCreateModalOpenChange,
}: CourseGradingAgentsSectionProps) {
  const { t } = useTranslation('common')
  const { graderAgentReviewInboxEnabled } = usePlatformFeatures()
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [templates, setTemplates] = useState<CourseGradingAgentTemplateSummary[]>([])
  const [agents, setAgents] = useState<CourseGradingAgentSummary[]>([])
  const [openingItemId, setOpeningItemId] = useState<string | null>(null)
  const [openAgent, setOpenAgent] = useState<OpenAgentState | null>(null)
  const [openTemplate, setOpenTemplate] = useState<OpenTemplateState | null>(null)
  const [createFromTemplate, setCreateFromTemplate] = useState<CourseGradingAgentTemplateSummary | null>(null)
  const [deleteTemplateTarget, setDeleteTemplateTarget] = useState<CourseGradingAgentTemplateSummary | null>(null)
  const [deletingTemplate, setDeletingTemplate] = useState(false)
  const [agentFilterQuery, setAgentFilterQuery] = useState('')
  const [agentSortKey, setAgentSortKey] = useState<AgentSortKey>('assignmentTitle')
  const [agentSortDir, setAgentSortDir] = useState<SortDir>('ascending')
  const [templateSortKey, setTemplateSortKey] = useState<TemplateSortKey>('name')
  const [templateSortDir, setTemplateSortDir] = useState<SortDir>('ascending')

  const existingAgentItemIds = useMemo(() => new Set(agents.map((agent) => agent.itemId)), [agents])

  const sortedTemplates = useMemo(() => {
    const copy = [...templates]
    const dir = templateSortDir === 'ascending' ? 1 : -1
    copy.sort((a, b) => {
      if (templateSortKey === 'name') {
        return a.name.localeCompare(b.name) * dir
      }
      return (Date.parse(a.updatedAt) - Date.parse(b.updatedAt)) * dir
    })
    return copy
  }, [templates, templateSortDir, templateSortKey])

  const filteredSortedAgents = useMemo(() => {
    const q = agentFilterQuery.trim().toLowerCase()
    const filtered = q
      ? agents.filter((agent) => {
          const title = agent.assignmentTitle.toLowerCase()
          const status = statusLabel(agent.status, t).toLowerCase()
          return title.includes(q) || status.includes(q)
        })
      : agents
    const copy = [...filtered]
    const dir = agentSortDir === 'ascending' ? 1 : -1
    copy.sort((a, b) => {
      if (agentSortKey === 'assignmentTitle') {
        return a.assignmentTitle.localeCompare(b.assignmentTitle) * dir
      }
      return (Date.parse(a.updatedAt) - Date.parse(b.updatedAt)) * dir
    })
    return copy
  }, [agentFilterQuery, agentSortDir, agentSortKey, agents, t])

  const toggleAgentSort = (key: AgentSortKey) => {
    if (agentSortKey === key) {
      setAgentSortDir((dir) => (dir === 'ascending' ? 'descending' : 'ascending'))
    } else {
      setAgentSortKey(key)
      setAgentSortDir('ascending')
    }
  }

  const toggleTemplateSort = (key: TemplateSortKey) => {
    if (templateSortKey === key) {
      setTemplateSortDir((dir) => (dir === 'ascending' ? 'descending' : 'ascending'))
    } else {
      setTemplateSortKey(key)
      setTemplateSortDir('ascending')
    }
  }

  const agentSortAria = (key: AgentSortKey): 'ascending' | 'descending' | 'none' =>
    agentSortKey === key ? agentSortDir : 'none'

  const templateSortAria = (key: TemplateSortKey): 'ascending' | 'descending' | 'none' =>
    templateSortKey === key ? templateSortDir : 'none'

  const reload = useCallback(async (opts?: { silent?: boolean }) => {
    if (!opts?.silent) {
      setLoading(true)
    }
    setLoadError(null)
    try {
      const [templatesRes, agentsRes] = await Promise.all([
        fetchCourseGradingAgentTemplates(courseCode),
        fetchCourseGradingAgents(courseCode),
      ])
      setTemplates(templatesRes.templates)
      setAgents(agentsRes.agents)
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : t('gradingAgent.settings.error.load'))
      if (!opts?.silent) {
        setTemplates([])
        setAgents([])
      }
    } finally {
      if (!opts?.silent) {
        setLoading(false)
      }
    }
  }, [courseCode, t])

  useEffect(() => {
    void reload()
  }, [reload])

  const openAgentEditor = async (agent: CourseGradingAgentSummary) => {
    setOpeningItemId(agent.itemId)
    try {
      const itemKind = agent.itemKind ?? 'assignment'
      if (itemKind === 'quiz') {
        const quiz = await fetchModuleQuiz(courseCode, agent.itemId)
        const { computeQuizQuestionSlots } = await import(
          '../../components/annotation/grader-agent/quiz-question-slots'
        )
        setOpenAgent({
          itemId: agent.itemId,
          itemKind: 'quiz',
          assignmentTitle: agent.assignmentTitle,
          rubric: null,
          maxPoints: quiz.pointsWorth ?? null,
          quizQuestionSlots: computeQuizQuestionSlots(quiz),
          quizQuestions: quiz.questions ?? [],
          seedWorkflow: null,
        })
      } else {
        const assignment = await fetchModuleAssignment(courseCode, agent.itemId)
        setOpenAgent({
          itemId: agent.itemId,
          itemKind: 'assignment',
          assignmentTitle: agent.assignmentTitle,
          rubric: assignment.rubric ?? null,
          maxPoints: assignment.pointsWorth ?? null,
          quizQuestionSlots: [],
          quizQuestions: [],
          seedWorkflow: null,
        })
      }
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : t('gradingAgent.settings.error.open'))
    } finally {
      setOpeningItemId(null)
    }
  }

  const openNewAgentEditor = async (result: CreateGradingAgentResult) => {
    if (result.source === 'asTemplate' && result.templateName) {
      setOpenTemplate({
        templateMode: { name: result.templateName },
        seedWorkflow: null,
      })
      onCreateModalOpenChange(false)
      return
    }

    if (!result.assignmentId) {
      throw new Error(t('gradingAgent.settings.create.error'))
    }

    let seedWorkflow: GraderAgentWorkflowSeed | null = null
    if (result.source === 'template' && result.templateId) {
      const { template } = await fetchGraderAgentTemplate(courseCode, result.templateId)
      seedWorkflow = {
        prompt: template.prompt,
        includeAssignmentContent: template.includeAssignmentContent,
        includeRubric: template.includeRubric,
        workflowGraph: template.workflowGraph,
      }
    }

    if (result.itemKind === 'quiz') {
      const quiz = await fetchModuleQuiz(courseCode, result.assignmentId)
      const { computeQuizQuestionSlots } = await import(
        '../../components/annotation/grader-agent/quiz-question-slots'
      )
      setOpenAgent({
        itemId: result.assignmentId,
        itemKind: 'quiz',
        assignmentTitle: quiz.title?.trim() || 'Untitled quiz',
        rubric: null,
        maxPoints: quiz.pointsWorth ?? null,
        quizQuestionSlots: computeQuizQuestionSlots(quiz),
        quizQuestions: quiz.questions ?? [],
        seedWorkflow,
      })
    } else {
      const assignment = await fetchModuleAssignment(courseCode, result.assignmentId)
      setOpenAgent({
        itemId: result.assignmentId,
        itemKind: 'assignment',
        assignmentTitle: assignment.title?.trim() || 'Untitled assignment',
        rubric: assignment.rubric ?? null,
        maxPoints: assignment.pointsWorth ?? null,
        quizQuestionSlots: [],
        quizQuestions: [],
        seedWorkflow,
      })
    }
    onCreateModalOpenChange(false)
  }

  const createAgentsFromTemplate = async (result: CreateGradingAgentFromTemplateResult) => {
    if (!createFromTemplate) {
      throw new Error(t('gradingAgent.settings.fromTemplate.error'))
    }

    const { template } = await fetchGraderAgentTemplate(courseCode, createFromTemplate.id)
    const workflowGraph = template.workflowGraph
    if (!workflowGraph) {
      throw new Error(t('gradingAgent.settings.fromTemplate.error'))
    }

    if (result.name !== createFromTemplate.name) {
      await postGraderAgentTemplate(courseCode, {
        name: result.name,
        prompt: template.prompt,
        includeAssignmentContent: template.includeAssignmentContent,
        includeRubric: template.includeRubric,
        workflowGraph,
      })
    }

    await cloneGraderAgentTemplateToAssignments(
      courseCode,
      createFromTemplate.id,
      result.assignmentIds,
      template,
    )

    setCreateFromTemplate(null)
    await reload({ silent: true })

    if (result.assignmentIds.length === 1) {
      const assignmentId = result.assignmentIds[0]
      if (!assignmentId) {
        throw new Error(t('gradingAgent.settings.fromTemplate.error'))
      }
      const assignment = await fetchModuleAssignment(courseCode, assignmentId)
      setOpenAgent({
        itemId: assignmentId,
        itemKind: 'assignment',
        assignmentTitle: assignment.title?.trim() || 'Untitled assignment',
        rubric: assignment.rubric ?? null,
        maxPoints: assignment.pointsWorth ?? null,
        quizQuestionSlots: [],
        quizQuestions: [],
        seedWorkflow: null,
      })
    }
  }

  const confirmDeleteTemplate = async () => {
    if (!deleteTemplateTarget) return
    setDeletingTemplate(true)
    setLoadError(null)
    try {
      await deleteGraderAgentTemplate(courseCode, deleteTemplateTarget.id)
      if (createFromTemplate?.id === deleteTemplateTarget.id) {
        setCreateFromTemplate(null)
      }
      setDeleteTemplateTarget(null)
      await reload({ silent: true })
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : t('gradingAgent.settings.deleteTemplate.error'))
    } finally {
      setDeletingTemplate(false)
    }
  }

  if (loading) {
    return (
      <p className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-300">
        <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
        {t('gradingAgent.settings.loading')}
      </p>
    )
  }

  return (
    <div className="w-full space-y-6">
      <p className="text-sm text-slate-600 dark:text-neutral-300">{t('gradingAgent.settings.description')}</p>
      {loadError ? (
        <p className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
          {loadError}
        </p>
      ) : null}

      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('gradingAgent.settings.templatesTitle')}
        </h3>
        {templates.length === 0 ? (
          <p className="text-sm text-slate-500 dark:text-neutral-400">{t('gradingAgent.settings.templatesEmpty')}</p>
        ) : (
          <div className="w-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-900/5 dark:border-neutral-700 dark:bg-neutral-900/40">
            <table className="w-full table-auto text-start text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50/80 dark:border-neutral-700 dark:bg-neutral-800/50">
                  <th className="w-px whitespace-nowrap px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    <button
                      type="button"
                      className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                      onClick={() => toggleTemplateSort('name')}
                      aria-sort={templateSortAria('name')}
                    >
                      {t('gradingAgent.settings.table.template')}
                    </button>
                  </th>
                  <th className="w-52 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    <button
                      type="button"
                      className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                      onClick={() => toggleTemplateSort('updatedAt')}
                      aria-sort={templateSortAria('updatedAt')}
                    >
                      {t('gradingAgent.settings.table.updated')}
                    </button>
                  </th>
                  <th className="w-12 px-4 py-3 text-end font-semibold text-slate-900 dark:text-neutral-100">
                    <span className="sr-only">{t('gradingAgent.settings.table.actions')}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {sortedTemplates.map((template) => (
                  <tr
                    key={template.id}
                    className="border-b border-slate-100 last:border-0 dark:border-neutral-800"
                  >
                    <td className="w-px whitespace-nowrap px-4 py-3 text-start">
                      <button
                        type="button"
                        onClick={() => setCreateFromTemplate(template)}
                        className="text-start font-medium text-indigo-700 hover:underline dark:text-indigo-300"
                      >
                        {template.name}
                      </button>
                    </td>
                    <td className="px-4 py-3 text-start text-slate-600 dark:text-neutral-300">
                      {formatAbsolute(template.updatedAt)}
                    </td>
                    <td className="px-4 py-3 text-end">
                      {!template.isBuiltin ? (
                        <button
                          type="button"
                          onClick={() => setDeleteTemplateTarget(template)}
                          className="inline-flex rounded-lg p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-700 dark:hover:bg-rose-950/40 dark:hover:text-rose-300"
                          aria-label={t('gradingAgent.settings.deleteTemplate.buttonAria', { name: template.name })}
                        >
                          <Trash2 className="h-4 w-4" aria-hidden />
                        </button>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('gradingAgent.settings.agentsTitle')}
        </h3>
        {agents.length === 0 ? (
          <p className="text-sm text-slate-500 dark:text-neutral-400">{t('gradingAgent.settings.empty')}</p>
        ) : (
          <div className="space-y-3">
            <div className="relative max-w-md">
              <Search
                className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 dark:text-neutral-500"
                aria-hidden
              />
              <input
                type="search"
                value={agentFilterQuery}
                onChange={(e) => setAgentFilterQuery(e.target.value)}
                placeholder={t('gradingAgent.settings.table.filterPlaceholder')}
                aria-label={t('gradingAgent.settings.table.filterPlaceholder')}
                className="w-full rounded-xl border border-slate-200 bg-white py-2 ps-10 pe-3 text-sm text-slate-900 shadow-sm outline-none focus:border-indigo-300 focus:ring-4 focus:ring-indigo-500/15 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-indigo-500/50"
              />
            </div>
            {filteredSortedAgents.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.settings.table.noMatch')}
              </p>
            ) : (
          <div className="w-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-900/5 dark:border-neutral-700 dark:bg-neutral-900/40">
            <table className="w-full table-auto text-start text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50/80 dark:border-neutral-700 dark:bg-neutral-800/50">
                  <th className="w-px whitespace-nowrap px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    <button
                      type="button"
                      className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                      onClick={() => toggleAgentSort('assignmentTitle')}
                      aria-sort={agentSortAria('assignmentTitle')}
                    >
                      {t('gradingAgent.settings.table.activity')}
                    </button>
                  </th>
                  <th className="w-28 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.status')}
                  </th>
                  <th className="w-36 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.autoGrade')}
                  </th>
                  <th className="w-52 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    <button
                      type="button"
                      className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                      onClick={() => toggleAgentSort('updatedAt')}
                      aria-sort={agentSortAria('updatedAt')}
                    >
                      {t('gradingAgent.settings.table.updated')}
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredSortedAgents.map((agent) => {
                  const opening = openingItemId === agent.itemId
                  return (
                    <tr
                      key={agent.id}
                      className="border-b border-slate-100 last:border-0 dark:border-neutral-800"
                    >
                      <td className="w-px whitespace-nowrap px-4 py-3 text-start">
                        <div className="space-y-1">
                          <button
                            type="button"
                            disabled={opening}
                            onClick={() => void openAgentEditor(agent)}
                            className="text-start font-medium text-indigo-700 hover:underline disabled:opacity-60 dark:text-indigo-300"
                          >
                            {agent.assignmentTitle}
                            {agent.itemKind === 'quiz' ? (
                              <span className="ms-2 text-xs font-normal text-violet-600 dark:text-violet-300">
                                {t('gradingAgent.settings.quizBadge')}
                              </span>
                            ) : null}
                            {agent.assignmentArchived ? (
                              <span className="ms-2 text-xs font-normal text-slate-500 dark:text-neutral-400">
                                {t('gradingAgent.settings.archivedAssignment')}
                              </span>
                            ) : null}
                          </button>
                          {graderAgentReviewInboxEnabled && (agent.reviewCount ?? 0) > 0 ? (
                            <button
                              type="button"
                              disabled={opening}
                              onClick={() => void openAgentEditor(agent)}
                              className="block text-xs font-semibold text-amber-700 hover:underline disabled:opacity-60 dark:text-amber-300"
                              aria-live="polite"
                            >
                              {t('gradingAgent.review.inbox.reviewLink', { count: agent.reviewCount ?? 0 })}
                            </button>
                          ) : null}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-start">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${statusClass(agent.status)}`}
                        >
                          {statusLabel(agent.status, t)}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-start text-slate-600 dark:text-neutral-300">
                        {agent.autoGradeNew
                          ? t('gradingAgent.settings.autoGradeOn')
                          : t('gradingAgent.settings.autoGradeOff')}
                      </td>
                      <td className="px-4 py-3 text-start text-slate-600 dark:text-neutral-300">
                        {formatAbsolute(agent.updatedAt)}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
            )}
          </div>
        )}
      </section>

      <CreateGradingAgentModal
        open={createModalOpen}
        courseCode={courseCode}
        templates={templates}
        existingAgentItemIds={existingAgentItemIds}
        onClose={() => onCreateModalOpenChange(false)}
        onContinue={openNewAgentEditor}
      />

      <CreateGradingAgentFromTemplateModal
        open={createFromTemplate != null}
        courseCode={courseCode}
        template={createFromTemplate}
        existingAgentItemIds={existingAgentItemIds}
        onClose={() => setCreateFromTemplate(null)}
        onCreate={createAgentsFromTemplate}
      />

      <ConfirmDialog
        open={deleteTemplateTarget != null}
        title={t('gradingAgent.settings.deleteTemplate.title')}
        description={
          deleteTemplateTarget ? (
            <>
              {t('gradingAgent.settings.deleteTemplate.description')}
              <p className="mt-2 font-medium text-slate-900 dark:text-neutral-100">
                {deleteTemplateTarget.name}
              </p>
            </>
          ) : null
        }
        confirmLabel={t('gradingAgent.settings.deleteTemplate.confirm')}
        cancelLabel={t('gradingAgent.settings.deleteTemplate.cancel')}
        variant="danger"
        busy={deletingTemplate}
        onConfirm={() => void confirmDeleteTemplate()}
        onClose={() => {
          if (!deletingTemplate) setDeleteTemplateTarget(null)
        }}
      />

      {openAgent ? (
        <GraderAgentWorkflowModal
          open
          onClose={() => {
            setOpenAgent(null)
            void reload()
          }}
          courseCode={courseCode}
          itemId={openAgent.itemId}
          itemKind={openAgent.itemKind}
          assignmentTitle={openAgent.assignmentTitle}
          submissionId={null}
          rubric={openAgent.rubric}
          maxPoints={openAgent.maxPoints}
          quizQuestionSlots={openAgent.quizQuestionSlots}
          quizQuestions={openAgent.quizQuestions}
          seedWorkflow={openAgent.seedWorkflow}
        />
      ) : null}

      {openTemplate ? (
        <GraderAgentWorkflowModal
          open
          onClose={() => {
            setOpenTemplate(null)
            void reload()
          }}
          courseCode={courseCode}
          itemId=""
          submissionId={null}
          rubric={null}
          maxPoints={null}
          seedWorkflow={openTemplate.seedWorkflow}
          templateMode={openTemplate.templateMode}
        />
      ) : null}
    </div>
  )
}