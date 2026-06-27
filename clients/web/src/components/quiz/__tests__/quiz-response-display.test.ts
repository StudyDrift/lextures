import { describe, expect, it } from 'vitest'
import { extractQuizResponseFiles, formatQuizResponseText } from '../quiz-response-format'

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

describe('extractQuizResponseFiles', () => {
  it('returns [] when there are no files', () => {
    expect(extractQuizResponseFiles({ textAnswer: 'hi' })).toEqual([])
    expect(extractQuizResponseFiles({})).toEqual([])
    expect(extractQuizResponseFiles(null)).toEqual([])
  })

  it('extracts imported file references and skips entries without a content path', () => {
    const files = extractQuizResponseFiles({
      files: [
        {
          fileId: 'abc',
          filename: 'image.png',
          mimeType: 'image/png',
          contentPath: '/api/v1/courses/C-1/course-files/abc/content',
        },
        { filename: 'broken.png' },
      ],
    })
    expect(files).toEqual([
      {
        fileId: 'abc',
        filename: 'image.png',
        mimeType: 'image/png',
        contentPath: '/api/v1/courses/C-1/course-files/abc/content',
      },
    ])
  })

  it('parses a JSON string payload', () => {
    const files = extractQuizResponseFiles(
      JSON.stringify({ files: [{ contentPath: '/x', filename: 'a.pdf' }] }),
    )
    expect(files).toHaveLength(1)
    expect(files[0]?.filename).toBe('a.pdf')
  })
})