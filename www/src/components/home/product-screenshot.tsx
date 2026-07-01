import { ProductPanelChrome } from './product-panel-chrome'

type ProductScreenshotProps = {
  src: string
  alt: string
  filename: string
  term?: string
  className?: string
  /** When true, only the screenshot is shown without window chrome. */
  bare?: boolean
}

export function ProductScreenshot({
  src,
  alt,
  filename,
  term,
  className = '',
  bare = false,
}: ProductScreenshotProps) {
  const image = (
    <img
      src={src}
      alt={alt}
      className="block h-auto w-full"
      loading="lazy"
      decoding="async"
    />
  )

  if (bare) {
    return <div className={className}>{image}</div>
  }

  return (
    <ProductPanelChrome filename={filename} term={term} className={className}>
      {image}
    </ProductPanelChrome>
  )
}
