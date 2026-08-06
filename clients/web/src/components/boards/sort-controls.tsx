import { useTranslation } from 'react-i18next'
import type { BoardSortMode } from '../../lib/boards-api'

const MODES: BoardSortMode[] = ['newest', 'oldest', 'author', 'mostReacted']

type SortControlsProps = {
  value: BoardSortMode
  onChange: (mode: BoardSortMode) => void
  /** Hide when layout uses its own ordering (timeline/map/canvas). */
  hidden?: boolean
}

export function SortControls({ value, onChange, hidden }: SortControlsProps) {
  const { t } = useTranslation('common')
  if (hidden) return null

  return (
    <label className="inline-flex items-center gap-2 text-xs text-fg-muted">
      <span>{t('boards.sort.label')}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value as BoardSortMode)}
        className="rounded border border-border-default bg-surface-raised px-2 py-1 text-xs dark:border-border-default dark:bg-surface-overlay"
        aria-label={t('boards.sort.label')}
      >
        {MODES.map((mode) => (
          <option key={mode} value={mode}>
            {t(`boards.sort.${mode}`)}
          </option>
        ))}
      </select>
    </label>
  )
}
