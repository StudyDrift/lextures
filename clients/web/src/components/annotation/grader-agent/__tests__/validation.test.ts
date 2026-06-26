import { describe, expect, it } from 'vitest'
import { synthesizeDefaultGraph, effectiveWorkflowGraph } from '../default-graph'
import { connectionIsValid, isWorkflowRunnable, validateWorkflowGraph } from '../validation'
import { WORKFLOW_VERSION, type GraderWorkflowGraph } from '../types'

function sampleGraphWithGrader(
  prompt = 'Grade fairly',
  includeContent = true,
  includeRubric = true,
): GraderWorkflowGraph {
  const nodes: GraderWorkflowGraph['nodes'] = [
    { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
    {
      id: 'g1',
      type: 'grader',
      position: { x: -320, y: 0 },
      data: { prompt, modelId: null },
    },
  ]
  const edges: GraderWorkflowGraph['edges'] = [
    { id: 'e1', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
    { id: 'e2', source: 'g1', sourceHandle: 'comments', target: 'output', targetHandle: 'comments' },
  ]

  if (includeContent || includeRubric) {
    nodes.push({
      id: 'act',
      type: 'activity',
      position: { x: -640, y: 80 },
      data: {},
    })
    if (includeContent) {
      edges.push({ id: 'e3', source: 'act', sourceHandle: 'content', target: 'g1', targetHandle: 'content' })
    }
    if (includeRubric) {
      edges.push({ id: 'e4', source: 'act', sourceHandle: 'rubric', target: 'g1', targetHandle: 'rubric' })
    }
  }

  return { version: WORKFLOW_VERSION, nodes, edges }
}

describe('grader agent workflow validation', () => {
  it('tolerates null edges from API deserialization', () => {
    const g = {
      version: WORKFLOW_VERSION,
      nodes: [{ id: 'output', type: 'output' as const, position: { x: 0, y: 0 }, data: {} }],
      edges: null as unknown as GraderWorkflowGraph['edges'],
    }
    expect(() => validateWorkflowGraph(g)).not.toThrow()
    expect(validateWorkflowGraph(g).some((i) => i.field === 'output.grade')).toBe(true)
  })

  it('starts with output-only default graph', () => {
    const g = synthesizeDefaultGraph('Grade fairly', true, true)
    expect(g.nodes).toHaveLength(1)
    expect(g.nodes[0]?.type).toBe('output')
    expect(g.edges).toHaveLength(0)
    expect(isWorkflowRunnable(g)).toBe(false)
  })

  it('accepts a wired grader graph with a prompt', () => {
    const g = sampleGraphWithGrader('Grade fairly', true, true)
    expect(isWorkflowRunnable(g)).toBe(true)
  })

  it('rejects cross-type grade to comments connection', () => {
    const g = sampleGraphWithGrader('Grade fairly', true, true)
    g.edges[1] = { id: 'e2', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'comments' }
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field.includes('output'))).toBe(true)
  })

  it('rejects unconnected grade slot', () => {
    const g = sampleGraphWithGrader('Grade fairly', true, true)
    g.edges = g.edges.filter((e) => e.targetHandle !== 'grade')
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field === 'output.grade')).toBe(true)
  })

  it('synthesizes legacy config into output-only graph', () => {
    const g = effectiveWorkflowGraph(null, 'Legacy prompt', false, true)
    expect(g.nodes.find((n) => n.type === 'grader')).toBeUndefined()
    expect(g.nodes.find((n) => n.type === 'output')).toBeDefined()
  })

  it('validates connection types', () => {
    const g = sampleGraphWithGrader('x', true, true)
    g.edges = []
    expect(connectionIsValid(g, 'g1', 'grade', 'output', 'grade')).toBe(true)
    expect(connectionIsValid(g, 'g1', 'grade', 'output', 'comments')).toBe(false)
  })

  it('requires a prompt on AI nodes', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: '' } },
        { id: 'g1', type: 'grader', position: { x: -160, y: 0 }, data: { prompt: 'Grade fairly' } },
      ],
      edges: [{ id: 'e1', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' }],
    }
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field === 'node:ai1.prompt')).toBe(true)
  })

  it('rejects punctuation-only AI prompts', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: '$' } },
      ],
      edges: [{ id: 'e1', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' }],
    }
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field === 'node:ai1.prompt')).toBe(true)
  })

  it('allows chaining AI nodes from input sources', () => {
    const nodes: GraderWorkflowGraph['nodes'] = [
      { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
      { id: 'sub', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
      { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Summarize' } },
      { id: 'ai2', type: 'ai', position: { x: -160, y: 0 }, data: { prompt: 'Grade' } },
      { id: 'g1', type: 'grader', position: { x: 160, y: 0 }, data: { prompt: 'Grade fairly' } },
    ]
    const emptyEdges: GraderWorkflowGraph['edges'] = []
    const g: GraderWorkflowGraph = { version: WORKFLOW_VERSION, nodes, edges: emptyEdges }
    expect(connectionIsValid(g, 'sub', 'submission', 'ai1', 'input')).toBe(true)
    expect(connectionIsValid(g, 'ai1', 'output', 'ai2', 'input')).toBe(true)

    const wired: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes,
      edges: [
        { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'ai1', targetHandle: 'input' },
        { id: 'e2', source: 'ai1', sourceHandle: 'output', target: 'ai2', targetHandle: 'input' },
        { id: 'e3', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(wired)).toBe(true)
  })

  it('allows activity content and rubric on the same AI input', () => {
    const nodes: GraderWorkflowGraph['nodes'] = [
      { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
      { id: 'act', type: 'activity', position: { x: -640, y: 0 }, data: {} },
      { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      { id: 'g1', type: 'grader', position: { x: 160, y: 0 }, data: { prompt: 'Grade fairly' } },
    ]
    const partial: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes,
      edges: [{ id: 'e1', source: 'act', sourceHandle: 'content', target: 'ai1', targetHandle: 'input' }],
    }
    expect(connectionIsValid(partial, 'act', 'rubric', 'ai1', 'input')).toBe(true)

    const wired: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes,
      edges: [
        { id: 'e1', source: 'act', sourceHandle: 'content', target: 'ai1', targetHandle: 'input' },
        { id: 'e2', source: 'act', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' },
        { id: 'e3', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(wired)).toBe(true)
    expect(connectionIsValid(wired, 'act', 'content', 'ai1', 'input')).toBe(false)
  })

  it('accepts code test runner wired to output grade', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: -80 }, data: {} },
        {
          id: 'ctr1',
          type: 'codeTestRunner',
          position: { x: -320, y: -40 },
          data: {
            runtime: 'python3.12',
            mapping: { type: 'linear', maxPoints: 10 },
            testCases: [{ id: 't1', input: '', expectedOutput: '1' }],
          },
        },
      ],
      edges: [
        { id: 'e-sub', source: 'sub1', sourceHandle: 'submission', target: 'ctr1', targetHandle: 'submission' },
        { id: 'e-grade', source: 'ctr1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(g)).toBe(true)
  })

  it('rejects router else branch that does not reach grade slot', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'sub', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
        {
          id: 'r1',
          type: 'conditionalRouter',
          position: { x: -320, y: 0 },
          data: { condition: { field: 'isEmpty', operator: 'isTrue', value: true } },
        },
        { id: 'ai1', type: 'ai', position: { x: -160, y: 80 }, data: { prompt: 'Grade' } },
      ],
      edges: [
        { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'r1', targetHandle: 'input' },
        { id: 'e2', source: 'r1', sourceHandle: 'then', target: 'output', targetHandle: 'grade' },
        { id: 'e3', source: 'r1', sourceHandle: 'else', target: 'ai1', targetHandle: 'input' },
      ],
    }
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field === 'node:r1.else')).toBe(true)
  })

  it('accepts router with both branches reaching grade slot', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'sub', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
        {
          id: 'r1',
          type: 'conditionalRouter',
          position: { x: -320, y: 0 },
          data: { condition: { field: 'isEmpty', operator: 'isTrue', value: true } },
        },
        { id: 'ai1', type: 'ai', position: { x: -160, y: 80 }, data: { prompt: 'Grade fairly' } },
      ],
      edges: [
        { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'r1', targetHandle: 'input' },
        { id: 'e2', source: 'r1', sourceHandle: 'then', target: 'output', targetHandle: 'grade' },
        { id: 'e3', source: 'r1', sourceHandle: 'else', target: 'ai1', targetHandle: 'input' },
        { id: 'e4', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(g)).toBe(true)
  })

  it('rejects confidence field without upstream grade on router input path', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'sub', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
        {
          id: 'r1',
          type: 'conditionalRouter',
          position: { x: -320, y: 0 },
          data: { condition: { field: 'confidence', operator: '<', value: 0.6 } },
        },
      ],
      edges: [
        { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'r1', targetHandle: 'input' },
        { id: 'e2', source: 'r1', sourceHandle: 'then', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field === 'node:r1.condition.field')).toBe(true)
  })

  it('accepts router with then branch to flag for review and else to grade', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'sub', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
        {
          id: 'r1',
          type: 'conditionalRouter',
          position: { x: -320, y: 0 },
          data: { condition: { field: 'isEmpty', operator: 'isTrue', value: true } },
        },
        {
          id: 'flag1',
          type: 'flagForReview',
          position: { x: 160, y: 0 },
          data: { queue: 'default', priority: 'normal', reasonTemplate: 'Blank submission' },
        },
        { id: 'ai1', type: 'ai', position: { x: -160, y: 80 }, data: { prompt: 'Grade fairly' } },
      ],
      edges: [
        { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'r1', targetHandle: 'input' },
        { id: 'e2', source: 'r1', sourceHandle: 'then', target: 'flag1', targetHandle: 'reason' },
        { id: 'e3', source: 'r1', sourceHandle: 'else', target: 'ai1', targetHandle: 'input' },
        { id: 'e4', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(g)).toBe(true)
  })

  it('flags criterion grader with unknown criterion id', () => {
    const rubric = {
      criteria: [
        {
          id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
          title: 'Thesis',
          description: '',
          levels: [{ label: 'Strong', points: 4 }],
        },
      ],
    }
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        {
          id: 'cg1',
          type: 'criterionGrader',
          position: { x: -320, y: 0 },
          data: { prompt: 'Score thesis', criterionId: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb' },
        },
        { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
      ],
      edges: [
        { id: 'e1', source: 'cg1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
        { id: 'e2', source: 'sub1', sourceHandle: 'submission', target: 'cg1', targetHandle: 'submission' },
      ],
    }
    const issues = validateWorkflowGraph(g, { rubric, assignmentItemId: 'item-1' })
    expect(issues.some((issue) => issue.field === 'node:cg1.criterionId')).toBe(true)
  })

  it('rejects reference wired to output grade slot', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'ref1', type: 'reference', position: { x: -640, y: 0 }, data: { text: 'Key' } },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [
        { id: 'e1', source: 'ref1', sourceHandle: 'reference', target: 'output', targetHandle: 'grade' },
        { id: 'e2', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(connectionIsValid(g, 'ref1', 'reference', 'output', 'grade')).toBe(false)
  })

  it('accepts reference wired to AI input', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'ref1', type: 'reference', position: { x: -640, y: 0 }, data: { text: 'Model answer body' } },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade using $ModelAnswer.Text' } },
        { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: 120 }, data: {} },
      ],
      edges: [
        { id: 'e1', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
        { id: 'e2', source: 'ref1', sourceHandle: 'reference', target: 'ai1', targetHandle: 'input' },
        { id: 'e3', source: 'sub1', sourceHandle: 'submission', target: 'ai1', targetHandle: 'input' },
      ],
    }
    const gWithoutRefEdge: GraderWorkflowGraph = {
      ...g,
      edges: g.edges.filter((edge) => edge.id !== 'e2'),
    }
    expect(connectionIsValid(gWithoutRefEdge, 'ref1', 'reference', 'ai1', 'input')).toBe(true)
    expect(isWorkflowRunnable(g)).toBe(true)
  })

  it('rejects rubric wired to output grade slot', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'rub1', type: 'rubric', position: { x: -640, y: 0 }, data: { source: 'assignment' } },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [
        { id: 'e1', source: 'rub1', sourceHandle: 'rubric', target: 'output', targetHandle: 'grade' },
        { id: 'e2', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(connectionIsValid(g, 'rub1', 'rubric', 'output', 'grade')).toBe(false)
  })

  it('accepts rubric wired to AI input', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        {
          id: 'rub1',
          type: 'rubric',
          position: { x: -640, y: 0 },
          data: {
            source: 'inline',
            rubric: {
              criteria: [{ id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', title: 'Thesis', levels: [{ label: 'Good', points: 10 }] }],
            },
          },
        },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade with rubric' } },
        { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: 120 }, data: {} },
      ],
      edges: [
        { id: 'e1', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
        { id: 'e2', source: 'rub1', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' },
        { id: 'e3', source: 'sub1', sourceHandle: 'submission', target: 'ai1', targetHandle: 'input' },
      ],
    }
    const gWithoutRubricEdge: GraderWorkflowGraph = {
      ...g,
      edges: g.edges.filter((edge) => edge.id !== 'e2'),
    }
    expect(connectionIsValid(gWithoutRubricEdge, 'rub1', 'rubric', 'ai1', 'input')).toBe(true)
    expect(isWorkflowRunnable(g, { rubric: { criteria: [] } })).toBe(true)
  })

  it('requires inline rubric criteria', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'rub1', type: 'rubric', position: { x: -640, y: 0 }, data: { source: 'inline', rubric: { criteria: [] } } },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [{ id: 'e1', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' }],
    }
    expect(validateWorkflowGraph(g).some((issue) => issue.field === 'node:rub1.rubric')).toBe(true)
  })

  it('allows multiple grade edges into score aggregator', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'agg1', type: 'scoreAggregator', position: { x: -80, y: 0 }, data: {} },
        { id: 'cg1', type: 'criterionGrader', position: { x: -320, y: -40 }, data: { prompt: 'A', criterionId: 'c1' } },
        { id: 'cg2', type: 'criterionGrader', position: { x: -320, y: 40 }, data: { prompt: 'B', criterionId: 'c2' } },
      ],
      edges: [
        { id: 'e1', source: 'cg1', sourceHandle: 'grade', target: 'agg1', targetHandle: 'grade' },
        { id: 'e2', source: 'cg2', sourceHandle: 'grade', target: 'agg1', targetHandle: 'grade' },
        { id: 'e3', source: 'agg1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(connectionIsValid(g, 'cg2', 'grade', 'agg1', 'grade')).toBe(true)
    expect(isWorkflowRunnable(g, { rubric: { criteria: [{ id: 'c1', title: 'A', levels: [] }, { id: 'c2', title: 'B', levels: [] }] } })).toBe(true)
  })

  it('allows fan-in grade edges on output node (e.g. conditional router branches)', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'router', type: 'conditionalRouter', position: { x: -80, y: 0 }, data: {} },
        { id: 'g1', type: 'grader', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [
        { id: 'e1', source: 'router', sourceHandle: 'then', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(connectionIsValid(g, 'router', 'else', 'output', 'grade')).toBe(true)
  })

  it('allows quiz question outputs to wire into conditional router input', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'quiz', type: 'quizResponses', position: { x: -320, y: 0 }, data: {} },
        { id: 'router', type: 'conditionalRouter', position: { x: -80, y: 0 }, data: {} },
        { id: 'output', type: 'output', position: { x: 160, y: 0 }, data: {} },
      ],
      edges: [],
    }
    expect(connectionIsValid(g, 'quiz', 'question-0', 'router', 'input')).toBe(true)
    expect(connectionIsValid(g, 'quiz', 'question-4', 'router', 'input')).toBe(true)
    expect(connectionIsValid(g, 'quiz', 'submission', 'router', 'input')).toBe(false)
  })

  it('rejects second edge on the same quiz question grade slot', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'ai1', type: 'ai', position: { x: -160, y: 0 }, data: { prompt: 'Grade Q1' } },
        { id: 'ai2', type: 'ai', position: { x: -160, y: 80 }, data: { prompt: 'Grade Q1 again' } },
      ],
      edges: [
        { id: 'e1', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade-0' },
      ],
    }
    expect(connectionIsValid(g, 'ai2', 'output', 'output', 'grade-0')).toBe(false)
  })

  it('rejects rubricMerge with duplicate criterion sources', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'agg1', type: 'scoreAggregator', position: { x: -80, y: 0 }, data: { mode: 'rubricMerge' } },
        { id: 'cg1', type: 'criterionGrader', position: { x: -320, y: -40 }, data: { prompt: 'A', criterionId: 'same' } },
        { id: 'cg2', type: 'criterionGrader', position: { x: -320, y: 40 }, data: { prompt: 'B', criterionId: 'same' } },
      ],
      edges: [
        { id: 'e1', source: 'cg1', sourceHandle: 'grade', target: 'agg1', targetHandle: 'grade' },
        { id: 'e2', source: 'cg2', sourceHandle: 'grade', target: 'agg1', targetHandle: 'grade' },
        { id: 'e3', source: 'agg1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(
      validateWorkflowGraph(g).some((issue) => issue.field === 'node:agg1.mode' && issue.message.includes('rubricMerge')),
    ).toBe(true)
  })

  it('rejects non-grade source wired to aggregator', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'agg1', type: 'scoreAggregator', position: { x: -80, y: 0 }, data: {} },
        { id: 'orig1', type: 'originality', position: { x: -320, y: 0 }, data: {} },
      ],
      edges: [],
    }
    expect(connectionIsValid(g, 'orig1', 'score', 'agg1', 'grade')).toBe(false)
  })

  it('requires reference text or file', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'ref1', type: 'reference', position: { x: -640, y: 0 }, data: { mode: 'modelAnswer' } },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [{ id: 'e1', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' }],
    }
    expect(validateWorkflowGraph(g).some((issue) => issue.field === 'node:ref1.text')).toBe(true)
  })
})