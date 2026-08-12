/**
 * SEO.4 FR-12 — AVIF + WebP with PNG fallback, dimensions, lazy by default.
 */
type OptimizedImageProps = {
  /** Path without extension, e.g. `/assets/screenshots/gradebook` or `/docs-dashboard` */
  srcBase: string
  alt: string
  width: number
  height: number
  className?: string
  /** Only the LCP image should set this. */
  priority?: boolean
}

export function OptimizedImage({
  srcBase,
  alt,
  width,
  height,
  className,
  priority = false,
}: OptimizedImageProps) {
  const loading = priority ? 'eager' : 'lazy'
  const fetchPriority = priority ? 'high' : undefined
  return (
    <picture>
      <source srcSet={`${srcBase}.avif`} type="image/avif" />
      <source srcSet={`${srcBase}.webp`} type="image/webp" />
      <img
        src={`${srcBase}.png`}
        alt={alt}
        width={width}
        height={height}
        className={className}
        loading={loading}
        // @ts-expect-error fetchpriority is valid on img
        fetchpriority={fetchPriority}
        decoding={priority ? 'sync' : 'async'}
      />
    </picture>
  )
}
