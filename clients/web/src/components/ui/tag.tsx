import { type ReactNode } from 'react'
import { Badge, type BadgeTone } from './badge'
import { IconButton } from './icon-button'
import { cx } from './utils'

export type TagProps = {
  children: ReactNode
  tone?: BadgeTone
  onRemove?: () => void
  /** Required when onRemove is set. */
  removeLabel?: string
  className?: string
}

export function Tag({ children, tone = 'neutral', onRemove, removeLabel, className = '' }: TagProps) {
  return (
    <Badge tone={tone} className={cx('gap-1 pe-1', className)}>
      {children}
      {onRemove ? (
        <IconButton
          variant="ghost"
          size="sm"
          aria-label={removeLabel ?? 'Remove'}
          className="!min-h-5 !min-w-5 !h-5 !w-5 rounded-full p-0"
          onClick={onRemove}
        >
          <span aria-hidden className="text-xs">
            ×
          </span>
        </IconButton>
      ) : null}
    </Badge>
  )
}
