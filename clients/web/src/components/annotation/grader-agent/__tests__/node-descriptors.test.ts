import { describe, expect, it } from 'vitest'
import { NODE_DESCRIPTORS, paletteNodeDefaults } from '../node-descriptors'
import type { PaletteNodeType } from '../types'

const PALETTE_NODE_TYPES: PaletteNodeType[] = [
  'studentSubmission',
  'activity',
  'codeTestRunner',
  'conditionalRouter',
  'flagForReview',
  'humanReviewGate',
  'scoreAggregator',
  'originality',
  'reference',
  'rubric',
  'criterionGrader',
  'ai',
]

describe('NODE_DESCRIPTORS', () => {
  it('covers every palette node type', () => {
    for (const type of PALETTE_NODE_TYPES) {
      expect(NODE_DESCRIPTORS[type]).toBeDefined()
    }
  })

  it('matches legacy addPaletteNode defaults snapshot', () => {
    const ctx = { itemId: 'assignment-item-1', nodeCount: 3 }
    expect(
      PALETTE_NODE_TYPES.map((type) => ({
        type,
        ...paletteNodeDefaults(type, ctx),
      })),
    ).toMatchInlineSnapshot(`
      [
        {
          "data": {},
          "idPrefix": "sub",
          "position": {
            "x": -640,
            "y": 40,
          },
          "type": "studentSubmission",
        },
        {
          "data": {
            "assignmentItemId": "assignment-item-1",
          },
          "idPrefix": "act",
          "position": {
            "x": -640,
            "y": 240,
          },
          "type": "activity",
        },
        {
          "data": {
            "mapping": {
              "maxPoints": 10,
              "type": "linear",
            },
            "onCompileError": "zero",
            "onTimeout": "zero",
            "runtime": "python3.12",
            "testCases": [
              {
                "expectedOutput": "",
                "id": "t1",
                "input": "",
                "isHidden": false,
              },
            ],
          },
          "idPrefix": "ctr",
          "position": {
            "x": -320,
            "y": 80,
          },
          "type": "codeTestRunner",
        },
        {
          "data": {
            "condition": {
              "field": "isEmpty",
              "operator": "isTrue",
              "value": true,
            },
          },
          "idPrefix": "rtr",
          "position": {
            "x": -320,
            "y": 200,
          },
          "type": "conditionalRouter",
        },
        {
          "data": {
            "priority": "normal",
            "queue": "default",
            "reasonTemplate": "Needs human review",
          },
          "idPrefix": "flag",
          "position": {
            "x": 160,
            "y": 200,
          },
          "type": "flagForReview",
        },
        {
          "data": {
            "confidenceFloor": 0.7,
            "mode": "belowConfidence",
            "queue": "default",
          },
          "idPrefix": "gate",
          "position": {
            "x": 0,
            "y": 160,
          },
          "type": "humanReviewGate",
        },
        {
          "data": {
            "confidence": "min",
            "mergeComments": true,
            "mode": "sum",
            "onMissing": "treatAsZero",
            "weights": {},
          },
          "idPrefix": "agg",
          "position": {
            "x": 0,
            "y": 120,
          },
          "type": "scoreAggregator",
        },
        {
          "data": {
            "flagThreshold": 0.4,
            "metric": "similarity",
          },
          "idPrefix": "orig",
          "position": {
            "x": -160,
            "y": 240,
          },
          "type": "originality",
        },
        {
          "data": {
            "mode": "modelAnswer",
            "text": "",
          },
          "idPrefix": "ref",
          "position": {
            "x": -640,
            "y": 320,
          },
          "type": "reference",
        },
        {
          "data": {
            "source": "assignment",
          },
          "idPrefix": "rub",
          "position": {
            "x": -640,
            "y": 400,
          },
          "type": "rubric",
        },
        {
          "data": {
            "prompt": "",
          },
          "idPrefix": "cg",
          "position": {
            "x": -320,
            "y": 120,
          },
          "type": "criterionGrader",
        },
        {
          "data": {},
          "idPrefix": "ai",
          "position": {
            "x": -320,
            "y": 160,
          },
          "type": "ai",
        },
      ]
    `)
  })
})
