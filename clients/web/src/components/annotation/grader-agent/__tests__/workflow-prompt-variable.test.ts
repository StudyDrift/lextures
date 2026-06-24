import { describe, expect, it } from 'vitest'
import type { GraderWorkflowGraph } from '../types'
import {
  filterPromptVariableNodes,
  filterPromptVariableProperties,
  findPromptVariableNode,
  getPromptVariableState,
  substitutePromptVariables,
  workflowNodeVariableName,
  workflowOutputHandleToProperty,
  workflowPromptVariableNodes,
} from '../workflow-prompt-variable'

const defaults = {
  studentSubmission: 'Student Submission',
  activity: 'Activity',
  ai: 'AI',
  grader: 'Grader (LLM)',
  output: 'Student grade',
}

function sampleGraph(): GraderWorkflowGraph {
  return {
    version: 1,
    nodes: [
      { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
      { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
      { id: 'act1', type: 'activity', position: { x: -640, y: 120 }, data: {} },
    ],
    edges: [
      { id: 'e1', source: 'sub1', sourceHandle: 'submission', target: 'ai1', targetHandle: 'input' },
      { id: 'e2', source: 'act1', sourceHandle: 'content', target: 'ai1', targetHandle: 'input' },
      { id: 'e3', source: 'act1', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' },
    ],
  }
}

describe('workflow prompt variables', () => {
  it('removes spaces from node labels for variable names', () => {
    expect(workflowNodeVariableName('Student Submission')).toBe('StudentSubmission')
    expect(workflowNodeVariableName('My Custom Node')).toBe('MyCustomNode')
  })

  it('maps output handles to property names', () => {
    expect(workflowOutputHandleToProperty('submission')).toBe('Submissions')
    expect(workflowOutputHandleToProperty('content')).toBe('Content')
    expect(workflowOutputHandleToProperty('rubric')).toBe('Rubric')
    expect(workflowOutputHandleToProperty('output')).toBe('Output')
    expect(workflowOutputHandleToProperty('reference')).toBe('Text')
  })

  it('lists wired input nodes and their properties', () => {
    const nodes = workflowPromptVariableNodes(sampleGraph(), 'ai1', defaults)
    expect(nodes).toHaveLength(2)
    expect(nodes.find((node) => node.variableName === 'StudentSubmission')?.properties).toEqual([
      { property: 'Submissions', handle: 'submission' },
    ])
    expect(nodes.find((node) => node.variableName === 'Activity')?.properties).toEqual([
      { property: 'Content', handle: 'content' },
      { property: 'Rubric', handle: 'rubric' },
    ])
  })

  it('uses renamed node labels in variable names', () => {
    const graph = sampleGraph()
    graph.nodes = graph.nodes.map((node) =>
      node.id === 'act1' ? { ...node, data: { ...node.data, label: 'Assignment Context' } } : node,
    )
    const nodes = workflowPromptVariableNodes(graph, 'ai1', defaults)
    expect(nodes.find((node) => node.nodeId === 'act1')?.variableName).toBe('AssignmentContext')
  })

  it('detects node and property autocomplete states', () => {
    const text = 'Grade like a TA.\nContent: $Activity.Co'
    expect(getPromptVariableState(text, text.length)).toEqual({
      kind: 'property',
      start: 26,
      nodeQuery: 'Activity',
      propertyQuery: 'Co',
    })
    expect(getPromptVariableState('Use $Stud', 9)).toEqual({
      kind: 'node',
      start: 4,
      query: 'Stud',
    })
    expect(getPromptVariableState('cost is $50', 10)).toBeNull()
  })

  it('filters nodes and properties for autocomplete', () => {
    const nodes = workflowPromptVariableNodes(sampleGraph(), 'ai1', defaults)
    expect(filterPromptVariableNodes(nodes, 'stud').map((node) => node.variableName)).toEqual([
      'StudentSubmission',
    ])
    const activity = findPromptVariableNode(nodes, 'Activity')
    expect(filterPromptVariableProperties(activity, 'rub')).toEqual([{ property: 'Rubric', handle: 'rubric' }])
  })

  it('substitutes variables in prompts', () => {
    const prompt = `Content: $Activity.Content\nRubric: $Activity.Rubric\nSubmission: $StudentSubmission.Submissions`
    const resolved = substitutePromptVariables(prompt, {
      Activity: { Content: 'Essay prompt', Rubric: 'Rubric text' },
      StudentSubmission: { Submissions: 'Essay part one\n\nEssay part two' },
    })
    expect(resolved).toBe(
      'Content: Essay prompt\nRubric: Rubric text\nSubmission: Essay part one\n\nEssay part two',
    )
    expect(
      substitutePromptVariables('Unknown: $Missing.Value', { Activity: { Content: 'x' } }),
    ).toBe('Unknown: $Missing.Value')
  })
})