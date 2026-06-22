import { OutputNode, GraderNode, AssignmentContextNode, SubmissionNode } from './workflow-nodes'

export const graderAgentNodeTypes = {
  output: OutputNode,
  grader: GraderNode,
  assignmentContext: AssignmentContextNode,
  submission: SubmissionNode,
}
