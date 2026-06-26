import { describe, expect, it } from 'vitest'
import { formatQuizResponseText } from '../quiz-response-format'

describe('formatQuizResponseText', () => {
  it('treats empty objects as no answer', () => {
    expect(formatQuizResponseText({}, 'essay')).toBe('')
    expect(formatQuizResponseText('{}', 'essay')).toBe('')
    expect(formatQuizResponseText(null, 'essay')).toBe('')
  })

  it('renders essay text answers', () => {
    expect(
      formatQuizResponseText({ textAnswer: 'My contributions this sprint have been: shipping.' }, 'essay'),
    ).toBe('My contributions this sprint have been: shipping.')
  })

  it('renders multiple choice labels from choices', () => {
    expect(
      formatQuizResponseText(
        { selectedChoiceIndex: 1 },
        'multiple_choice',
        ['No', 'Yes'],
      ),
    ).toBe('Yes')
  })

  it('renders true/false without choices', () => {
    expect(formatQuizResponseText({ selectedChoiceIndex: 0 }, 'true_false')).toBe('True')
  })

  it('does not stringify unknown canvas payloads as raw JSON', () => {
    expect(formatQuizResponseText({ canvasAnswer: { nested: true } }, 'essay')).toBe('')
  })
})