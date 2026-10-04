import { stripImageDisplayFragment } from './course-file-image'

/** Display context for course hero images. `banner` matches the 5:1 course-page slot. */
export type CourseHeroImageSize =
  | 'full'
  | 'banner'
  | 'catalog-card'
  | 'catalog-list'
  | 'catalog-gallery'
  | 'catalog-thumb'

type SizeSpec = { w: number; h: number; q: number }

/** Banner dimensions match server imageproxy.BannerWidth/Height/Quality. */
const SIZE_SPECS: Record<Exclude<CourseHeroImageSize, 'full'>, SizeSpec> = {
  // 1920×384 covers a typical content column at 2× without upscaling on the server.
  banner: { w: 1920, h: 384, q: 85 },
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

/** Resolve the image URL to fetch, appending resize query params for banner and catalog sizes. */
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