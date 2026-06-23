import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import {
  CreateGradingAgentModal,
  type CreateGradingAgentResult,
} from '../../components/annotation/grader-agent/create-grading-agent-modal'
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
  type CourseGradingAgentSummary,
  type CourseGradingAgentTemplateSummary,
  type RubricDefinition,
} from '../../lib/courses-api'
import { formatAbsolute } from '../../lib/format-datetime'

type CourseGradingAgentsSectionProps = {
  courseCode: string
  createModalOpen: boolean
  onCreateModalOpenChange: (open: boolean) => void
}

type OpenAgentState = {
  itemId: string
  assignmentTitle: string
  rubric: RubricDefinition | null
  maxPoints: number | null
  seedWorkflow: GraderAgentWorkflowSeed | null
}

type OpenTemplateState = {
  templateMode: GraderAgentTemplateMode
  seedWorkflow: GraderAgentWorkflowSeed | null
}

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
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [templates, setTemplates] = useState<CourseGradingAgentTemplateSummary[]>([])
  const [agents, setAgents] = useState<CourseGradingAgentSummary[]>([])
  const [openingItemId, setOpeningItemId] = useState<string | null>(null)
  const [openAgent, setOpenAgent] = useState<OpenAgentState | null>(null)
  const [openTemplate, setOpenTemplate] = useState<OpenTemplateState | null>(null)

  const existingAgentItemIds = useMemo(() => new Set(agents.map((agent) => agent.itemId)), [agents])

  const reload = useCallback(async () => {
    setLoading(true)
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
      setTemplates([])
      setAgents([])
    } finally {
      setLoading(false)
    }
  }, [courseCode, t])

  useEffect(() => {
    void reload()
  }, [reload])

  const openAgentEditor = async (agent: CourseGradingAgentSummary) => {
    setOpeningItemId(agent.itemId)
    try {
      const assignment = await fetchModuleAssignment(courseCode, agent.itemId)
      setOpenAgent({
        itemId: agent.itemId,
        assignmentTitle: agent.assignmentTitle,
        rubric: assignment.rubric ?? null,
        maxPoints: assignment.pointsWorth ?? null,
        seedWorkflow: null,
      })
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

    const assignment = await fetchModuleAssignment(courseCode, result.assignmentId)
    setOpenAgent({
      itemId: result.assignmentId,
      assignmentTitle: assignment.title?.trim() || 'Untitled assignment',
      rubric: assignment.rubric ?? null,
      maxPoints: assignment.pointsWorth ?? null,
      seedWorkflow,
    })
    onCreateModalOpenChange(false)
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
                    {t('gradingAgent.settings.table.template')}
                  </th>
                  <th className="w-52 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.updated')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {templates.map((template) => (
                  <tr
                    key={template.id}
                    className="border-b border-slate-100 last:border-0 dark:border-neutral-800"
                  >
                    <td className="w-px whitespace-nowrap px-4 py-3 text-start font-medium text-slate-900 dark:text-neutral-100">
                      {template.name}
                    </td>
                    <td className="px-4 py-3 text-start text-slate-600 dark:text-neutral-300">
                      {formatAbsolute(template.updatedAt)}
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
          <div className="w-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-900/5 dark:border-neutral-700 dark:bg-neutral-900/40">
            <table className="w-full table-auto text-start text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50/80 dark:border-neutral-700 dark:bg-neutral-800/50">
                  <th className="w-px whitespace-nowrap px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.assignment')}
                  </th>
                  <th className="w-28 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.status')}
                  </th>
                  <th className="w-36 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.autoGrade')}
                  </th>
                  <th className="w-52 px-4 py-3 text-start font-semibold text-slate-900 dark:text-neutral-100">
                    {t('gradingAgent.settings.table.updated')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {agents.map((agent) => {
                  const opening = openingItemId === agent.itemId
                  return (
                    <tr
                      key={agent.id}
                      className="border-b border-slate-100 last:border-0 dark:border-neutral-800"
                    >
                      <td className="w-px whitespace-nowrap px-4 py-3 text-start">
                        <button
                          type="button"
                          disabled={opening}
                          onClick={() => void openAgentEditor(agent)}
                          className="text-start font-medium text-indigo-700 hover:underline disabled:opacity-60 dark:text-indigo-300"
                        >
                          {agent.assignmentTitle}
                          {agent.assignmentArchived ? (
                            <span className="ms-2 text-xs font-normal text-slate-500 dark:text-neutral-400">
                              {t('gradingAgent.settings.archivedAssignment')}
                            </span>
                          ) : null}
                        </button>
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
      </section>

      <CreateGradingAgentModal
        open={createModalOpen}
        courseCode={courseCode}
        templates={templates}
        existingAgentItemIds={existingAgentItemIds}
        onClose={() => onCreateModalOpenChange(false)}
        onContinue={openNewAgentEditor}
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
          assignmentTitle={openAgent.assignmentTitle}
          submissionId={null}
          rubric={openAgent.rubric}
          maxPoints={openAgent.maxPoints}
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