import { useTranslation } from 'react-i18next'

type AgentConfidenceFloorSettingsProps = {
  confidenceFloor: number | null | undefined
  disabled?: boolean
  onChange: (floor: number | null) => void
  compact?: boolean
}

const DEFAULT_FLOOR = 0.8

function floorPercent(floor: number | null | undefined): number {
  if (typeof floor === 'number' && Number.isFinite(floor) && floor > 0) {
    return Math.round(floor * 100)
  }
  return Math.round(DEFAULT_FLOOR * 100)
}

export function AgentConfidenceFloorSettings({
  confidenceFloor,
  disabled = false,
  onChange,
  compact = false,
}: AgentConfidenceFloorSettingsProps) {
  const { t } = useTranslation('common')
  const enabled =
    typeof confidenceFloor === 'number' && Number.isFinite(confidenceFloor) && confidenceFloor > 0
  const percent = floorPercent(confidenceFloor)

  return (
    <div className={compact ? 'space-y-2' : 'space-y-3 text-sm text-slate-700 dark:text-neutral-200'}>
      {!compact ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.settings.confidenceFloor.help')}
        </p>
      ) : null}
      <label className="flex min-h-10 cursor-pointer items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
        <input
          type="checkbox"
          className="size-4"
          checked={enabled}
          disabled={disabled}
          onChange={(e) => {
            if (e.target.checked) {
              onChange(DEFAULT_FLOOR)
              return
            }
            onChange(null)
          }}
        />
        {t('gradingAgent.settings.confidenceFloor.enable')}
      </label>
      {enabled ? (
        <label className="block">
          <span className="mb-1.5 block text-xs font-medium text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.settings.confidenceFloor.threshold', { percent })}
          </span>
          <input
            type="range"
            min={50}
            max={100}
            step={5}
            disabled={disabled}
            value={percent}
            aria-valuemin={50}
            aria-valuemax={100}
            aria-valuenow={percent}
            aria-label={t('gradingAgent.settings.confidenceFloor.threshold', { percent })}
            onChange={(e) => onChange(Number(e.target.value) / 100)}
            className="w-full accent-indigo-600"
          />
          <div className="mt-1 flex justify-between text-xs tabular-nums text-slate-500 dark:text-neutral-400">
            <span>50%</span>
            <span className="font-medium text-slate-700 dark:text-neutral-200">{percent}%</span>
            <span>100%</span>
          </div>
        </label>
      ) : (
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.settings.confidenceFloor.off')}
        </p>
      )}
      {!compact ? (
        <button
          type="button"
          disabled={disabled || !enabled}
          onClick={() => onChange(null)}
          className="text-sm font-medium text-indigo-700 hover:underline disabled:opacity-50 dark:text-indigo-300"
        >
          {t('gradingAgent.settings.confidenceFloor.neverHold')}
        </button>
      ) : null}
    </div>
  )
}