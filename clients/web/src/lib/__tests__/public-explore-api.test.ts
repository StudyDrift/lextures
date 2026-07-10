import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  fetchExploreCatalogCategories,
  fetchExploreCourse,
  fetchExploreCourseReviews,
  searchExploreCatalog,
} from '../public-explore-api'

function mockFetchSequence(responses: Array<{ body: unknown; ok?: boolean; status?: number }>) {
  const fetchMock = vi.fn()
  for (const response of responses) {
    fetchMock.mockResolvedValueOnce({
      ok: response.ok ?? true,
      status: response.status ?? 200,
      json: async () => response.body,
    } as Response)
  }
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('searchExploreCatalog', () => {
  it('uses the public catalog when available', async () => {
    const fetchMock = mockFetchSequence([
      { body: { courses: [{ slug: 'from-catalog' }], total: 1, nextCursor: '' } },
    ])
    const res = await searchExploreCatalog({ q: 'python' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect((fetchMock.mock.calls[0][0] as string)).toContain('/api/v1/public/catalog/courses?')
    expect(res.courses[0]?.slug).toBe('from-catalog')
  })

  it('falls back to marketplace when the catalog is disabled', async () => {
    const fetchMock = mockFetchSequence([
      { body: {}, ok: false, status: 404 },
      {
        body: {
          courses: [
            {
              id: '1',
              slug: 'intro-python',
              courseCode: 'PY101',
              title: 'Intro',
              description: 'Learn Python',
              heroImageUrl: null,
              category: 'Programming',
              level: 'beginner',
              language: 'en',
              priceCents: 0,
              priceCurrency: 'usd',
              listPriceCents: null,
              enrollmentCount: 10,
              averageRating: null,
              ratingCount: 0,
              instructorName: 'Ada',
              createdAt: '2026-01-01T00:00:00Z',
            },
          ],
          total: 1,
          nextCursor: '',
        },
      },
    ])
    const res = await searchExploreCatalog({})
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect((fetchMock.mock.calls[1][0] as string)).toContain('/api/v1/public/marketplace/courses')
    expect(res.courses[0]?.slug).toBe('intro-python')
    expect(res.courses[0]?.difficultyLevel).toBe('beginner')
  })
})

describe('fetchExploreCourse', () => {
  it('falls back to marketplace detail when the catalog is disabled', async () => {
    mockFetchSequence([
      { body: {}, ok: false, status: 404 },
      {
        body: {
          course: {
            id: '1',
            slug: 'introduction-to-python',
            courseCode: 'PY101',
            title: 'Introduction to Python',
            description: 'Basics',
            heroImageUrl: null,
            category: 'Programming',
            level: 'beginner',
            language: 'en',
            priceCents: 0,
            priceCurrency: 'usd',
            listPriceCents: null,
            enrollmentCount: 5,
            averageRating: 4.5,
            ratingCount: 2,
            instructorName: 'Ada',
            createdAt: '2026-01-01T00:00:00Z',
          },
          whatsIncluded: { moduleCount: 1, itemCount: 3 },
          jsonLd: { '@type': 'Course' },
        },
      },
    ])
    const detail = await fetchExploreCourse('introduction-to-python')
    expect(detail.course.title).toBe('Introduction to Python')
    expect(detail.course.difficultyLevel).toBe('beginner')
    expect(detail.jsonLd).toEqual({ '@type': 'Course' })
  })
})

describe('fetchExploreCatalogCategories', () => {
  it('falls back to marketplace categories when the catalog is disabled', async () => {
    mockFetchSequence([
      { body: {}, ok: false, status: 404 },
      { body: { categories: [{ category: 'Science', count: 2 }] } },
    ])
    const cats = await fetchExploreCatalogCategories()
    expect(cats).toEqual([{ category: 'Science', count: 2 }])
  })
})

describe('fetchExploreCourseReviews', () => {
  it('falls back to marketplace reviews when catalog reviews return 404', async () => {
    mockFetchSequence([
      { body: {}, ok: false, status: 404 },
      {
        body: {
          summary: { ratingCount: 1, distribution: {} },
          reviews: [{ id: 'r1', rating: 5, reviewerDisplayName: 'Sam', createdAt: '2026-01-01' }],
        },
      },
    ])
    const reviews = await fetchExploreCourseReviews('intro-python')
    expect(reviews.reviews).toHaveLength(1)
  })
})
