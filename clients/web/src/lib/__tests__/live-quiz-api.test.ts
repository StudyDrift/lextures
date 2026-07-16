import { describe, expect, it } from 'vitest'
import {
  liveQuizQuestionSchema,
  listKitsResponseSchema,
  listQuestionsResponseSchema,
  quizKitSchema,
  validateKitResponseSchema,
} from '../live-quiz-api-schemas'

describe('live-quiz-api-schemas', () => {
  it('parses a kit payload', () => {
    const kit = quizKitSchema.parse({
      id: '11111111-1111-1111-1111-111111111111',
      courseId: '22222222-2222-2222-2222-222222222222',
      title: 'Unit 1',
      description: '',
      slug: 'unit-1',
      coverImageRef: null,
      status: 'draft',
      visibility: 'course',
      tags: ['review'],
      questionCount: 0,
      archived: false,
      createdBy: null,
      createdAt: '2026-07-16T12:00:00Z',
      updatedAt: '2026-07-16T12:00:00Z',
    })
    expect(kit.title).toBe('Unit 1')
    expect(kit.status).toBe('draft')
  })

  it('parses a list response', () => {
    const body = listKitsResponseSchema.parse({
      kits: [],
      total: 0,
      page: 1,
      pageSize: 50,
      totalPages: 0,
    })
    expect(body.kits).toEqual([])
  })

  it('parses a live quiz question', () => {
    const q = liveQuizQuestionSchema.parse({
      id: '33333333-3333-3333-3333-333333333333',
      kitId: '11111111-1111-1111-1111-111111111111',
      position: 0,
      questionType: 'mc_single',
      prompt: 'Capital?',
      promptMediaRef: null,
      promptMediaAlt: null,
      options: [
        { id: 'a', text: 'Paris', isCorrect: true },
        { id: 'b', text: 'London', isCorrect: false },
      ],
      correctAnswer: null,
      timeLimitSeconds: 15,
      pointsStyle: 'standard',
      answerShuffle: true,
      explanation: null,
      sourceQuestionId: null,
      version: 1,
      createdAt: '2026-07-16T12:00:00Z',
      updatedAt: '2026-07-16T12:00:00Z',
    })
    expect(q.questionType).toBe('mc_single')
    expect(q.options).toHaveLength(2)
  })

  it('parses list questions and validate payloads', () => {
    const list = listQuestionsResponseSchema.parse({ questions: [] })
    expect(list.questions).toEqual([])
    const v = validateKitResponseSchema.parse({
      isReady: false,
      issues: [{ questionId: 'q1', code: 'missing_correct', message: 'Mark a correct answer.' }],
    })
    expect(v.issues[0]?.code).toBe('missing_correct')
  })
})
