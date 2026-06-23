import { type ComponentPropsWithoutRef, useEffect, useState } from 'react'
import { authorizedFetch } from '../lib/api'
import { needsAuthenticatedCourseImageSrc, resolveAuthorizedFetchPath } from '../lib/course-file-image'

type Props = ComponentPropsWithoutRef<'img'>

/** Renders a hero image, fetching with auth when src is a course-files content URL. */
export function CourseHeroImage({ src, className, alt = '', ...props }: Props) {
  const [resolvedSrc, setResolvedSrc] = useState<string | undefined>(() =>
    src && !needsAuthenticatedCourseImageSrc(src) ? src : undefined,
  )

  useEffect(() => {
    let cancelled = false
    let blobUrl: string | null = null
    if (!src || !needsAuthenticatedCourseImageSrc(src)) {
      setResolvedSrc(src ?? undefined)
      return
    }
    void authorizedFetch(resolveAuthorizedFetchPath(src))
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
  }, [src])

  return (
    <img
      src={resolvedSrc}
      alt={alt}
      className={['lex-content-img', className].filter(Boolean).join(' ')}
      {...props}
    />
  )
}
