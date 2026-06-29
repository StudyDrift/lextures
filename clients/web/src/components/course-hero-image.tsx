import { type ComponentPropsWithoutRef, useEffect, useState } from 'react'
import { authorizedFetch } from '../lib/api'
import { needsAuthenticatedCourseImageSrc, resolveAuthorizedFetchPath } from '../lib/course-file-image'
import { courseHeroImageSrc, type CourseHeroImageSize } from '../lib/course-hero-image-url'

type Props = ComponentPropsWithoutRef<'img'> & {
  /** `full` preserves original quality (course dashboard); catalog sizes request smaller thumbnails. */
  size?: CourseHeroImageSize
}

/** Renders a hero image, fetching with auth when src is a course-files content URL. */
export function CourseHeroImage({ src, size = 'full', className, alt = '', ...props }: Props) {
  const fetchSrc = courseHeroImageSrc(src, size)

  const [resolvedSrc, setResolvedSrc] = useState<string | undefined>(() =>
    fetchSrc && !needsAuthenticatedCourseImageSrc(fetchSrc) ? fetchSrc : undefined,
  )

  useEffect(() => {
    let cancelled = false
    let blobUrl: string | null = null
    if (!fetchSrc || !needsAuthenticatedCourseImageSrc(fetchSrc)) {
      setResolvedSrc(fetchSrc ?? undefined)
      return
    }
    void authorizedFetch(resolveAuthorizedFetchPath(fetchSrc))
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status))
        return r.blob()
      })
      .then((blob) => {
        if (cancelled) return
        blobUrl = URL.createObjectURL(blob)
        setResolvedSrc(blobUrl)
      })
      .catch(() => {
        if (!cancelled) setResolvedSrc(undefined)
      })
    return () => {
      cancelled = true
      if (blobUrl) URL.revokeObjectURL(blobUrl)
    }
  }, [fetchSrc])

  return (
    <img
      src={resolvedSrc}
      alt={alt}
      className={['lex-content-img', className].filter(Boolean).join(' ')}
      {...props}
    />
  )
}
