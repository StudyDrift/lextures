import type { CSSProperties } from 'react'
import { COURSES_COPY } from '../../lib/courses-copy'
import type { MarketplaceCategory } from '../../lib/marketplace-api'

export type CourseFilterState = {
  q: string
  category: string
  level: string
  language: string
  freeOnly: boolean
  sort: string
}

type CourseFiltersProps = {
  value: CourseFilterState
  categories: MarketplaceCategory[]
  onChange: (next: CourseFilterState) => void
}

const LEVELS = [
  { value: '', label: COURSES_COPY.anyLevel },
  { value: 'beginner', label: COURSES_COPY.beginner },
  { value: 'intermediate', label: COURSES_COPY.intermediate },
  { value: 'advanced', label: COURSES_COPY.advanced },
] as const

const SORTS = [
  { value: 'popular', label: COURSES_COPY.sortPopular },
  { value: 'newest', label: COURSES_COPY.sortNewest },
  { value: 'price', label: COURSES_COPY.sortPrice },
  { value: 'rating', label: COURSES_COPY.sortRating },
] as const

export function CourseFilters({ value, categories, onChange }: CourseFiltersProps) {
  const set = <K extends keyof CourseFilterState>(key: K, v: CourseFilterState[K]) => {
    onChange({ ...value, [key]: v })
  }

  return (
    <div className="flex flex-col gap-5">
      <div>
        <label htmlFor="courses-search" className="sr-only">
          {COURSES_COPY.searchLabel}
        </label>
        <input
          id="courses-search"
          type="search"
          value={value.q}
          onChange={e => set('q', e.target.value)}
          placeholder={COURSES_COPY.searchPlaceholder}
          className="w-full border px-4 py-3 text-[15px] outline-none focus:ring-2"
          style={{
            backgroundColor: 'var(--panel)',
            borderColor: 'var(--line-card)',
            borderRadius: 'var(--radius-card)',
            color: 'var(--ink-nav)',
          }}
        />
      </div>

      <div role="group" aria-label={COURSES_COPY.categoriesLabel} className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => set('category', '')}
          className="cursor-pointer rounded-full px-3 py-1.5 text-[13px] font-medium"
          style={chipStyle(!value.category)}
          aria-pressed={!value.category}
        >
          {COURSES_COPY.allCategories}
        </button>
        {categories.map(c => (
          <button
            key={c.category}
            type="button"
            onClick={() => set('category', c.category)}
            className="cursor-pointer rounded-full px-3 py-1.5 text-[13px] font-medium"
            style={chipStyle(value.category === c.category)}
            aria-pressed={value.category === c.category}
          >
            {c.category}
            <span className="ml-1 opacity-70">({c.count})</span>
          </button>
        ))}
      </div>

      <div className="flex flex-wrap gap-3">
        <label className="flex flex-col gap-1 text-[13px]" style={{ color: 'var(--text-soft)' }}>
          {COURSES_COPY.levelLabel}
          <select
            value={value.level}
            onChange={e => set('level', e.target.value)}
            className="border px-3 py-2 text-[14px]"
            style={selectStyle}
          >
            {LEVELS.map(l => (
              <option key={l.value || 'any'} value={l.value}>
                {l.label}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-[13px]" style={{ color: 'var(--text-soft)' }}>
          {COURSES_COPY.sortLabel}
          <select
            value={value.sort}
            onChange={e => set('sort', e.target.value)}
            className="border px-3 py-2 text-[14px]"
            style={selectStyle}
          >
            {SORTS.map(s => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </label>

        <label
          className="mt-auto flex cursor-pointer items-center gap-2 pb-2 text-[14px]"
          style={{ color: 'var(--ink-nav)' }}
        >
          <input
            type="checkbox"
            checked={value.freeOnly}
            onChange={e => set('freeOnly', e.target.checked)}
          />
          {COURSES_COPY.freeOnly}
        </label>
      </div>
    </div>
  )
}

const selectStyle = {
  backgroundColor: 'var(--panel)',
  borderColor: 'var(--line-card)',
  borderRadius: 'var(--radius-card)',
  color: 'var(--ink-nav)',
} as const

function chipStyle(active: boolean): CSSProperties {
  return active
    ? {
        backgroundColor: 'var(--ink-nav)',
        color: 'var(--cream)',
      }
    : {
        backgroundColor: 'rgba(38,58,60,0.06)',
        color: 'var(--ink-nav)',
      }
}
