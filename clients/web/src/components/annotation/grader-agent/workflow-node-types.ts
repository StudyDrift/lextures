import {
  OutputNode,
  GraderNode,
  AiNode,
  ActivityNode,
  StudentSubmissionNode,
  SubmissionNode,
  AssignmentContextNode,
} from './workflow-nodes'

export const graderAgentNodeTypes = {
  output: OutputNode,
  grader: GraderNode,
  ai: AiNode,
  activity: ActivityNode,
  studentSubmission: StudentSubmissionNode,
  submission: SubmissionNode,
  assignmentContext: AssignmentContextNode,
}
