import { stripImageDisplayFragment } from './course-file-image'

/** Display context for course hero banners; catalog variants request smaller server thumbnails. */
export type CourseHeroImageSize = 'full' | 'catalog-card' | 'catalog-list' | 'catalog-gallery' | 'catalog-thumb'

type SizeSpec = { w: number; h: number; q: number }

const SIZE_SPECS: Record<Exclude<CourseHeroImageSize, 'full'>, SizeSpec> = {
  'catalog-thumb': { w: 80, h: 80, q: 80 },
  'catalog-list': { w: 224, h: 160, q: 82 },
  'catalog-gallery': { w: 480, h: 360, q: 82 },
  'catalog-card': { w: 640, h: 320, q: 85 },
}

function isResizableCourseFileContentURL(base: string): boolean {
  return (
    (base.includes('/course-files/') || base.includes('/files/items/')) && base.endsWith('/content')
  )
}

/** Resolve the image URL to fetch, appending resize query params for catalog thumbnails. */
export function courseHeroImageSrc(
  src: string | null | undefined,
  size: CourseHeroImageSize = 'full',
): string | undefined {
  if (!src) return undefined
  if (size === 'full') return src

  const { base } = stripImageDisplayFragment(src)
  if (!isResizableCourseFileContentURL(base)) return src

  const spec = SIZE_SPECS[size]
  const url = new URL(base, 'http://local')
  url.searchParams.set('w', String(spec.w))
  url.searchParams.set('h', String(spec.h))
  url.searchParams.set('q', String(spec.q))
  return `${url.pathname}${url.search}`
}