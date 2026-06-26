import type { PaletteNodeType } from './types'
import {
  defaultCodeTestRunnerNodeData,
  defaultConditionalRouterNodeData,
  defaultFlagForReviewNodeData,
  defaultHumanReviewGateNodeData,
  defaultOriginalityNodeData,
  defaultReferenceNodeData,
  defaultRubricNodeData,
  defaultScoreAggregatorNodeData,
  defaultSetScoreNodeData,
} from './types'

export type NodeDescriptorContext = {
  itemId: string
  nodeCount: number
}

export type NodeDescriptor = {
  idPrefix: string
  fallbackPosition: (nodeCount: number) => { x: number; y: number }
  defaultData: (ctx: NodeDescriptorContext) => Record<string, unknown>
}

export const NODE_DESCRIPTORS: Record<PaletteNodeType, NodeDescriptor> = {
  studentSubmission: {
    idPrefix: 'sub',
    fallbackPosition: (nodeCount) => ({ x: -640, y: -80 + nodeCount * 40 }),
    defaultData: () => ({}),
  },
  quizResponses: {
    idPrefix: 'quiz',
    fallbackPosition: () => ({ x: -420, y: 0 }),
    defaultData: () => ({}),
  },
  activity: {
    idPrefix: 'act',
    fallbackPosition: (nodeCount) => ({ x: -640, y: 120 + nodeCount * 40 }),
    defaultData: ({ itemId }) => ({ assignmentItemId: itemId }),
  },
  codeTestRunner: {
    idPrefix: 'ctr',
    fallbackPosition: (nodeCount) => ({ x: -320, y: -40 + nodeCount * 40 }),
    defaultData: () => defaultCodeTestRunnerNodeData(),
  },
  conditionalRouter: {
    idPrefix: 'rtr',
    fallbackPosition: (nodeCount) => ({ x: -320, y: 80 + nodeCount * 40 }),
    defaultData: () => defaultConditionalRouterNodeData(),
  },
  flagForReview: {
    idPrefix: 'flag',
    fallbackPosition: (nodeCount) => ({ x: 160, y: 80 + nodeCount * 40 }),
    defaultData: () => defaultFlagForReviewNodeData(),
  },
  humanReviewGate: {
    idPrefix: 'gate',
    fallbackPosition: (nodeCount) => ({ x: 0, y: 40 + nodeCount * 40 }),
    defaultData: () => defaultHumanReviewGateNodeData(),
  },
  scoreAggregator: {
    idPrefix: 'agg',
    fallbackPosition: (nodeCount) => ({ x: 0, y: 0 + nodeCount * 40 }),
    defaultData: () => defaultScoreAggregatorNodeData(),
  },
  originality: {
    idPrefix: 'orig',
    fallbackPosition: (nodeCount) => ({ x: -160, y: 120 + nodeCount * 40 }),
    defaultData: () => defaultOriginalityNodeData(),
  },
  reference: {
    idPrefix: 'ref',
    fallbackPosition: (nodeCount) => ({ x: -640, y: 200 + nodeCount * 40 }),
    defaultData: () => defaultReferenceNodeData(),
  },
  rubric: {
    idPrefix: 'rub',
    fallbackPosition: (nodeCount) => ({ x: -640, y: 280 + nodeCount * 40 }),
    defaultData: () => defaultRubricNodeData(),
  },
  criterionGrader: {
    idPrefix: 'cg',
    fallbackPosition: (nodeCount) => ({ x: -320, y: 0 + nodeCount * 40 }),
    defaultData: () => ({ prompt: '' }),
  },
  ai: {
    idPrefix: 'ai',
    fallbackPosition: (nodeCount) => ({ x: -320, y: 40 + nodeCount * 40 }),
    defaultData: () => ({}),
  },
  setScore: {
    idPrefix: 'ss',
    fallbackPosition: (nodeCount) => ({ x: 0, y: 80 + nodeCount * 40 }),
    defaultData: () => defaultSetScoreNodeData(),
  },
}

export function paletteNodeDefaults(
  type: PaletteNodeType,
  ctx: NodeDescriptorContext,
): {
  idPrefix: string
  position: { x: number; y: number }
  data: Record<string, unknown>
} {
  const descriptor = NODE_DESCRIPTORS[type]
  return {
    idPrefix: descriptor.idPrefix,
    position: descriptor.fallbackPosition(ctx.nodeCount),
    data: descriptor.defaultData(ctx),
  }
}
