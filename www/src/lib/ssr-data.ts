/**
 * Build-time data passed into React during SSG and rehydrated on the client
 * via `window.__LEXTURES_SSR__` so the first client render matches the HTML.
 */
import type { PublicMarketplaceCourse, PublicMarketplaceCourseDetail } from './marketplace-api'
import type { ContentArticle, ContentSnapshot } from './content-source'

export type SsrData = {
  /** Path this HTML was generated for (e.g. `/pricing`). */
  path?: string
  article?: ContentArticle | null
  articleIndex?: ContentSnapshot | null
  courseDetail?: PublicMarketplaceCourseDetail | null
  coursesIndex?: {
    courses: PublicMarketplaceCourse[]
    total: number
  } | null
}

declare global {
  interface Window {
    __LEXTURES_SSR__?: SsrData
  }
}

export function readClientSsrData(): SsrData {
  if (typeof window === 'undefined') return {}
  return window.__LEXTURES_SSR__ ?? {}
}
