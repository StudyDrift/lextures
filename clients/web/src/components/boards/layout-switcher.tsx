import { useTranslation } from 'react-i18next'
import {
  LayoutGrid,
  Columns3,
  Rows3,
  Square,
  Map as MapIcon,
  CalendarRange,
  Move,
  Lock,
  Unlock,
} from 'lucide-react'
import { BOARD_LAYOUTS, type BoardLayout } from '../../lib/boards-api'

const ICONS: Record<BoardLayout, typeof LayoutGrid> = {
  wall: Square,
  stream: Rows3,
  grid: LayoutGrid,
  columns: Columns3,
  canvas: Move,
  timeline: CalendarRange,
  map: MapIcon,
}

type LayoutSwitcherProps = {
  layout: BoardLayout
  layoutLocked: boolean
  canManage: boolean
  onChangeLayout: (layout: BoardLayout) => void
  onToggleLock: () => void
}

export function LayoutSwitcher({
  layout,
  layoutLocked,
  canManage,
  onChangeLayout,
  onToggleLock,
}: LayoutSwitcherProps) {
  const { t } = useTranslation('common')

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div
        className="inline-flex rounded-md border border-slate-200 p-0.5 dark:border-neutral-700"
        role="group"
        aria-label={t('boards.layout.switcherAria')}
      >
        {BOARD_LAYOUTS.map((mode) => {
          const Icon = ICONS[mode]
          const active = layout === mode
          return (
            <button
              key={mode}
              type="button"
              disabled={!canManage}
              title={t(`boards.layout.${mode}`)}
              aria-label={t(`boards.layout.${mode}`)}
              aria-pressed={active}
              onClick={() => onChangeLayout(mode)}
              className={`rounded px-2 py-1.5 ${
                active
                  ? 'bg-indigo-600 text-white'
                  : 'text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800'
              } disabled:cursor-not-allowed disabled:opacity-50`}
            >
              <Icon className="size-4" aria-hidden />
            </button>
          )
        })}
      </div>
      {canManage ? (
        <button
          type="button"
          onClick={onToggleLock}
          aria-pressed={layoutLocked}
          className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
        >
          {layoutLocked ? <Lock className="size-3.5" aria-hidden /> : <Unlock className="size-3.5" aria-hidden />}
          {layoutLocked ? t('boards.layout.unlock') : t('boards.layout.lock')}
        </button>
      ) : layoutLocked ? (
        <span className="inline-flex items-center gap-1 text-xs text-slate-500">
          <Lock className="size-3.5" aria-hidden />
          {t('boards.layout.lockedBadge')}
        </span>
      ) : null}
    </div>
  )
}
