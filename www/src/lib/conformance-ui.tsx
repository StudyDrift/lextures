import type { ConformanceLevel } from './vpat-data'

export function conformanceBadgeClass(level: ConformanceLevel): string {
  switch (level) {
    case 'Supports':
      return 'bg-emerald-50 text-emerald-700'
    case 'Partially Supports':
      return 'bg-amber-50 text-amber-700'
    case 'Does Not Support':
      return 'bg-red-50 text-red-700'
    case 'Not Applicable':
      return 'bg-stone-100 text-stone-600'
    default: {
      const _exhaustive: never = level
      return _exhaustive
    }
  }
}

export function ConformanceBadge({ level }: { level: ConformanceLevel }) {
  return (
    <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${conformanceBadgeClass(level)}`}>
      {level}
    </span>
  )
}
