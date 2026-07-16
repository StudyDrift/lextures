import type { AnswerShape } from './answer-shape-meta'

export function ShapeIcon({ shape, className = '' }: { shape: AnswerShape; className?: string }) {
  const common = `inline-block ${className}`
  switch (shape) {
    case 'triangle':
      return (
        <svg viewBox="0 0 24 24" className={common} aria-hidden="true">
          <polygon points="12,3 22,21 2,21" fill="currentColor" />
        </svg>
      )
    case 'diamond':
      return (
        <svg viewBox="0 0 24 24" className={common} aria-hidden="true">
          <polygon points="12,2 22,12 12,22 2,12" fill="currentColor" />
        </svg>
      )
    case 'circle':
      return (
        <svg viewBox="0 0 24 24" className={common} aria-hidden="true">
          <circle cx="12" cy="12" r="9" fill="currentColor" />
        </svg>
      )
    case 'square':
      return (
        <svg viewBox="0 0 24 24" className={common} aria-hidden="true">
          <rect x="4" y="4" width="16" height="16" fill="currentColor" />
        </svg>
      )
    case 'pentagon':
      return (
        <svg viewBox="0 0 24 24" className={common} aria-hidden="true">
          <polygon points="12,2 22,9 18,21 6,21 2,9" fill="currentColor" />
        </svg>
      )
    case 'hexagon':
      return (
        <svg viewBox="0 0 24 24" className={common} aria-hidden="true">
          <polygon points="12,2 21,7 21,17 12,22 3,17 3,7" fill="currentColor" />
        </svg>
      )
    default: {
      const _exhaustive: never = shape
      return _exhaustive
    }
  }
}
