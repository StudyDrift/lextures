import { type HTMLAttributes } from 'react'
import { cx } from './utils'

export type AvatarProps = HTMLAttributes<HTMLSpanElement> & {
  src?: string
  alt: string
  /** Fallback initials when image missing. */
  initials?: string
  size?: 'sm' | 'md' | 'lg'
}

const sizeClass = {
  sm: 'h-6 w-6 text-[10px]',
  md: 'h-9 w-9 text-xs',
  lg: 'h-12 w-12 text-sm',
} as const

export function Avatar({
  src,
  alt,
  initials,
  size = 'md',
  className = '',
  ...props
}: AvatarProps) {
  return (
    <span
      className={cx(
        'inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-accent-surface font-semibold text-accent-fg',
        sizeClass[size],
        className,
      )}
      {...props}
    >
      {src ? (
        <img src={src} alt={alt} className="h-full w-full object-cover" />
      ) : (
        <span aria-label={alt}>{initials ?? alt.slice(0, 2).toUpperCase()}</span>
      )}
    </span>
  )
}
