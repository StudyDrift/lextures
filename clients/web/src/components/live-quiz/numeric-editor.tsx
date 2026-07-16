import { useTranslation } from 'react-i18next'

export type NumericDraft = {
  value: number
  tolerance: number
  unit?: string
}

type Props = {
  value: NumericDraft
  onChange: (v: NumericDraft) => void
  disabled?: boolean
}

export function NumericEditor({ value, onChange, disabled }: Props) {
  const { t } = useTranslation('common')
  return (
    <div className="grid gap-3 sm:grid-cols-3">
      <label className="text-sm">
        <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-200">
          {t('liveQuiz.editor.numericValue')}
        </span>
        <input
          type="number"
          step="any"
          disabled={disabled}
          value={Number.isFinite(value.value) ? value.value : 0}
          onChange={(e) => onChange({ ...value, value: Number(e.target.value) })}
          className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
        />
      </label>
      <label className="text-sm">
        <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-200">
          {t('liveQuiz.editor.numericTolerance')}
        </span>
        <input
          type="number"
          step="any"
          min={0}
          disabled={disabled}
          value={Number.isFinite(value.tolerance) ? value.tolerance : 0}
          onChange={(e) => onChange({ ...value, tolerance: Number(e.target.value) })}
          className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
        />
      </label>
      <label className="text-sm">
        <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-200">
          {t('liveQuiz.editor.numericUnit')}
        </span>
        <input
          disabled={disabled}
          value={value.unit ?? ''}
          onChange={(e) => onChange({ ...value, unit: e.target.value })}
          className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
        />
      </label>
    </div>
  )
}
