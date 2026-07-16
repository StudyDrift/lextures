import { useTranslation } from 'react-i18next'
import type { BoardConnState } from '../../lib/boards-realtime'

type Props = {
  connState: BoardConnState
}

export function BoardSyncStatus({ connState }: Props) {
  const { t } = useTranslation('common')
  if (connState === 'offline') return null

  let label = t('boards.sync.connecting')
  let className = 'text-slate-500 dark:text-neutral-400'
  if (connState === 'connected') {
    label = t('boards.sync.live')
    className = 'text-emerald-600 dark:text-emerald-400'
  } else if (connState === 'disconnected') {
    label = t('boards.sync.reconnecting')
    className = 'text-amber-600 dark:text-amber-400'
  }

  return (
    <span data-testid="board-sync-status" className={`text-xs font-medium ${className}`}>
      {connState === 'connected' ? (
        <span className="inline-flex items-center gap-1.5">
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-emerald-500" aria-hidden />
          {label}
        </span>
      ) : (
        label
      )}
    </span>
  )
}
