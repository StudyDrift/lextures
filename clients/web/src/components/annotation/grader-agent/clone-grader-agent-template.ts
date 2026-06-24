import {
  fetchGraderAgentTemplate,
  putGraderAgentConfig,
  type GraderAgentTemplateApi,
  type GraderWorkflowGraphApi,
} from '../../../lib/courses-api'

function requireTemplateWorkflowGraph(
  template: GraderAgentTemplateApi,
): GraderWorkflowGraphApi {
  if (!template.workflowGraph) {
    throw new Error('Template workflow is missing.')
  }
  return template.workflowGraph
}

export async function cloneGraderAgentTemplateToAssignments(
  courseCode: string,
  templateId: string,
  assignmentIds: readonly string[],
  template?: GraderAgentTemplateApi,
): Promise<GraderAgentTemplateApi> {
  const resolved =
    template ?? (await fetchGraderAgentTemplate(courseCode, templateId)).template
  const workflowGraph = requireTemplateWorkflowGraph(resolved)

  await Promise.all(
    assignmentIds.map((assignmentId) =>
      putGraderAgentConfig(courseCode, assignmentId, {
        prompt: resolved.prompt,
        includeAssignmentContent: resolved.includeAssignmentContent,
        includeRubric: resolved.includeRubric,
        status: 'draft',
        autoGradeNew: false,
        workflowGraph,
      }),
    ),
  )

  return resolved
}
