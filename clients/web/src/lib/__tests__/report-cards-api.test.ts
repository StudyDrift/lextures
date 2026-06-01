import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it } from 'vitest'
import { setAccessToken } from '../auth'
import {
  fetchCourseReportCards,
  patchReportCard,
  fetchCommentBank,
  createCommentBankEntry,
  deleteCommentBankEntry,
  fetchAICommentSuggestion,
  releaseReportCards,
  fetchParentReportCards,
  type ReportCard,
  type CommentBankEntry,
} from '../report-cards-api'
import { server } from '../../test/mocks/server'

const sampleCard: ReportCard = {
  id: 'rc-1',
  studentId: 'stu-1',
  courseId: 'course-1',
  gradingPeriod: 'Q1-2026',
  finalGradePct: 94.5,
  letterGrade: 'A',
  comment: 'Great work this quarter.',
  status: 'draft',
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
}

const sampleEntry: CommentBankEntry = {
  id: 'cbe-1',
  orgId: 'org-1',
  category: 'Academic Effort',
  text: 'Shows strong effort throughout the quarter.',
  active: true,
}

describe('fetchCourseReportCards', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.get('http://localhost:8080/api/v1/courses/CS101/report-cards/Q1-2026', () => {
        return HttpResponse.json({ reportCards: [sampleCard], period: 'Q1-2026' })
      }),
    )
  })

  it('returns report cards list', async () => {
    const res = await fetchCourseReportCards('CS101', 'Q1-2026')
    expect(res.reportCards).toHaveLength(1)
    expect(res.reportCards[0]!.id).toBe('rc-1')
    expect(res.reportCards[0]!.finalGradePct).toBe(94.5)
    expect(res.period).toBe('Q1-2026')
  })
})

describe('patchReportCard', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.patch('http://localhost:8080/api/v1/report-cards/rc-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        return HttpResponse.json({ ...sampleCard, comment: body.comment as string })
      }),
    )
  })

  it('sends patch body and returns updated card', async () => {
    const updated = await patchReportCard('rc-1', { comment: 'Updated comment.' })
    expect(updated.comment).toBe('Updated comment.')
  })
})

describe('fetchCommentBank', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.get(
        'http://localhost:8080/api/v1/admin/orgs/org-1/report-cards/comment-bank',
        () => {
          return HttpResponse.json({ entries: [sampleEntry] })
        },
      ),
    )
  })

  it('returns comment bank entries', async () => {
    const res = await fetchCommentBank('org-1')
    expect(res.entries).toHaveLength(1)
    expect(res.entries[0]!.category).toBe('Academic Effort')
  })
})

describe('createCommentBankEntry', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.post(
        'http://localhost:8080/api/v1/admin/orgs/org-1/report-cards/comment-bank',
        async ({ request }) => {
          const body = (await request.json()) as Record<string, unknown>
          return HttpResponse.json(
            { ...sampleEntry, text: body.text as string },
            { status: 201 },
          )
        },
      ),
    )
  })

  it('creates an entry and returns it', async () => {
    const entry = await createCommentBankEntry('org-1', 'Academic Effort', 'New phrase.')
    expect(entry.text).toBe('New phrase.')
  })
})

describe('deleteCommentBankEntry', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.delete(
        'http://localhost:8080/api/v1/admin/orgs/org-1/report-cards/comment-bank/cbe-1',
        () => new HttpResponse(null, { status: 204 }),
      ),
    )
  })

  it('sends DELETE and resolves without error', async () => {
    await expect(deleteCommentBankEntry('org-1', 'cbe-1')).resolves.toBeUndefined()
  })
})

describe('fetchAICommentSuggestion', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.post('http://localhost:8080/api/v1/ai/report-card-comment', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        return HttpResponse.json({
          suggestion: `Great job in ${body.courseName as string} with ${body.gradePct as number}%!`,
        })
      }),
    )
  })

  it('returns AI suggestion string', async () => {
    const suggestion = await fetchAICommentSuggestion('Mathematics', 94, 2)
    expect(suggestion).toContain('Mathematics')
    expect(suggestion).toContain('94')
  })
})

describe('releaseReportCards', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.post('http://localhost:8080/api/v1/courses/CS101/report-cards/Q1-2026/release', () => {
        return HttpResponse.json({ released: 5, message: '5 report card(s) released.' })
      }),
    )
  })

  it('returns released count', async () => {
    const res = await releaseReportCards('CS101', 'Q1-2026')
    expect(res.released).toBe(5)
  })
})

describe('fetchParentReportCards', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.get(
        'http://localhost:8080/api/v1/parent/students/stu-1/report-cards',
        () => HttpResponse.json({ reportCards: [{ ...sampleCard, status: 'released' }] }),
      ),
    )
  })

  it('returns released cards for student', async () => {
    const res = await fetchParentReportCards('stu-1')
    expect(res.reportCards[0]!.status).toBe('released')
  })
})
