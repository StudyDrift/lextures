import { type ComponentPropsWithoutRef, useEffect, useState } from 'react'
import {
  fetchCourseFileImageBlob,
  needsAuthenticatedCourseImageSrc,
} from '../lib/course-file-image'
import { courseHeroImageSrc, type CourseHeroImageSize } from '../lib/course-hero-image-url'

type Props = ComponentPropsWithoutRef<'img'> & {
  /** `full` keeps the original URL. `banner` and catalog sizes request a cached derivative. */
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
    void fetchCourseFileImageBlob(fetchSrc)
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
      decoding="async"
    />
  )
}
