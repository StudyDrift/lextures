import { describe, expect, it } from 'vitest'
import type { AnswerAck, LiveAnswerPayload, LiveGameStateFrame } from '../live-quiz-realtime'

describe('live-quiz-realtime state frame', () => {
  it('projector-safe question omits correctness until reveal', () => {
    const open: LiveGameStateFrame = {
      type: 'state',
      seq: 1,
      gameId: 'g1',
      phase: 'question_open',
      status: 'running',
      questionIndex: 0,
      joinCode: '123456',
      kitTitle: 'Demo',
      pacing: 'manual',
      players: [],
      questionCount: 1,
      question: {
        index: 0,
        questionType: 'mc_single',
        prompt: 'Q?',
        options: [{ id: 'a', text: 'A' }],
        timeLimitSeconds: 20,
        pointsStyle: 'standard',
      },
    }
    expect(open.question?.correctOptionIds).toBeUndefined()

    const reveal: LiveGameStateFrame = {
      ...open,
      phase: 'question_reveal',
      seq: 2,
      question: {
        ...open.question!,
        correctOptionIds: ['a'],
      },
    }
    expect(reveal.question?.correctOptionIds).toEqual(['a'])
  })

  it('documents answer payload shapes and ack feedback fields', () => {
    const mc: LiveAnswerPayload = { optionId: 'b' }
    const multi: LiveAnswerPayload = { optionIds: ['a', 'c'] }
    const typed: LiveAnswerPayload = { text: 'paris' }
    const numeric: LiveAnswerPayload = { value: 42 }
    const ordering: LiveAnswerPayload = { order: ['1', '2', '3'] }
    expect(mc.optionId).toBe('b')
    expect(multi.optionIds).toHaveLength(2)
    expect(typed.text).toBe('paris')
    expect(numeric.value).toBe(42)
    expect(ordering.order[0]).toBe('1')

    const ack: AnswerAck = {
      type: 'answer_ack',
      ok: true,
      questionIndex: 0,
      isCorrect: true,
      points: 840,
      streak: 2,
      rank: 3,
    }
    expect(ack.points).toBe(840)
    expect(ack.rank).toBe(3)
  })
})
