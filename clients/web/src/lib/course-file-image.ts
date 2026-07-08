/** Parse `#w=…&h=…` display size appended to image URLs in Markdown (TipTap round-trip). */
export function stripImageDisplayFragment(src: string): {
  base: string
  displayWidth?: number
  displayHeight?: number
} {
  const hash = src.indexOf('#')
  if (hash < 0) return { base: src }
  const base = src.slice(0, hash)
  const frag = src.slice(hash + 1)
  const wm = /(?:^|&)w=(\d+)/.exec(frag)
  const hm = /(?:^|&)h=(\d+)/.exec(frag)
  if (wm && hm) {
    return { base, displayWidth: parseInt(wm[1], 10), displayHeight: parseInt(hm[1], 10) }
  }
  return { base: src }
}

/** Pathname for course-file content URLs, ignoring `#w=&h=` fragments and `?resize` query params. */
export function courseFileContentPathname(src: string): string {
  const { base } = stripImageDisplayFragment(src)
  const pathOnly = base.split('?')[0]?.split('#')[0] ?? base
  try {
    if (pathOnly.startsWith('http://') || pathOnly.startsWith('https://')) {
      return new URL(pathOnly).pathname
    }
  } catch {
    // fall through with the raw path
  }
  return pathOnly
}

/** True when `src` points at a course file blob that requires `Authorization`. */
export function needsAuthenticatedCourseImageSrc(src: string): boolean {
  const pathname = courseFileContentPathname(src)
  return (
    (pathname.includes('/course-files/') || pathname.includes('/files/items/')) &&
    pathname.endsWith('/content')
  )
}

/** Path for `authorizedFetch` (path-only `/api/…`); strips display-size fragment. */
export function resolveAuthorizedFetchPath(src: string): string {
  const { base } = stripImageDisplayFragment(src)
  if (base.startsWith('/api/')) return base
  try {
    const u = new URL(base)
    if (u.pathname.startsWith('/api/')) return `${u.pathname}${u.search}`
    return base
  } catch {
    return base
  }
}
