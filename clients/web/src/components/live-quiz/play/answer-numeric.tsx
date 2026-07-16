import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export function AnswerNumeric({
  locked,
  onSubmit,
}: {
  locked: boolean
  onSubmit: (value: number) => void
}) {
  const { t } = useTranslation('common')
  const [value, setValue] = useState('')

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault()
        if (locked) return
        const n = Number(value)
        if (!Number.isFinite(n)) return
        onSubmit(n)
      }}
    >
      <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
        {t('liveQuiz.answer.numericLabel')}
        <input
          type="number"
          inputMode="decimal"
          value={value}
          disabled={locked}
          onChange={(e) => setValue(e.target.value)}
          className="mt-1 min-h-12 w-full rounded-xl border border-slate-300 bg-white px-3 text-lg dark:border-neutral-700 dark:bg-neutral-900"
        />
      </label>
      <button
        type="submit"
        disabled={locked || value === ''}
        className="min-h-12 w-full rounded-xl bg-indigo-600 px-4 py-3 text-base font-semibold text-white disabled:opacity-40"
      >
        {t('liveQuiz.answer.submit')}
      </button>
    </form>
  )
}
