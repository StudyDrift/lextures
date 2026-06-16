import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  fetchPublicCatalogCategories,
  fetchPublicCatalogCourse,
  formatPrice,
  searchPublicCatalog,
} from '../public-catalog-api'

function mockFetchOnce(body: unknown, init?: { ok?: boolean; status?: number }) {
  const res = {
    ok: init?.ok ?? true,
    status: init?.status ?? 200,
    json: async () => body,
  } as Response
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(res))
  return globalThis.fetch as unknown as ReturnType<typeof vi.fn>
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('formatPrice', () => {
  it('renders Free for zero and below', () => {
    expect(formatPrice(0)).toBe('Free')
    expect(formatPrice(-100)).toBe('Free')
  })
  it('renders dollars for positive prices', () => {
    expect(formatPrice(4999)).toBe('$49.99')
    expect(formatPrice(100)).toBe('$1.00')
  })
})

describe('searchPublicCatalog', () => {
  it('builds query params and returns results', async () => {
    const fetchMock = mockFetchOnce({ courses: [], total: 0, nextCursor: '' })
    await searchPublicCatalog({ q: 'python', level: 'beginner', priceMax: 0, sort: 'rating' })
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain('/api/v1/public/catalog/courses?')
    expect(url).toContain('q=python')
    expect(url).toContain('level=beginner')
    expect(url).toContain('price_max=0')
    expect(url).toContain('sort=rating')
  })

  it('throws a friendly message when the catalog is disabled (404)', async () => {
    mockFetchOnce({}, { ok: false, status: 404 })
    await expect(searchPublicCatalog({})).rejects.toThrow('Course catalog is not available.')
  })
})

describe('fetchPublicCatalogCategories', () => {
  it('returns the categories array', async () => {
    mockFetchOnce({ categories: [{ category: 'Science', count: 3 }] })
    const cats = await fetchPublicCatalogCategories()
    expect(cats).toEqual([{ category: 'Science', count: 3 }])
  })
})

describe('fetchPublicCatalogCourse', () => {
  it('encodes the slug and returns the detail', async () => {
    const fetchMock = mockFetchOnce({ course: { slug: 'a b' }, jsonLd: {} })
    const detail = await fetchPublicCatalogCourse('a b')
    expect((fetchMock.mock.calls[0][0] as string)).toContain('a%20b')
    expect(detail.course.slug).toBe('a b')
  })
})
