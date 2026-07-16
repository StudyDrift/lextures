import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export function AnswerType({
  locked,
  onSubmit,
}: {
  locked: boolean
  onSubmit: (text: string) => void
}) {
  const { t } = useTranslation('common')
  const [value, setValue] = useState('')

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault()
        if (locked || !value.trim()) return
        onSubmit(value.trim())
      }}
    >
      <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
        {t('liveQuiz.answer.typeLabel')}
        <input
          type="text"
          value={value}
          disabled={locked}
          autoComplete="off"
          onChange={(e) => setValue(e.target.value)}
          className="mt-1 min-h-12 w-full rounded-xl border border-slate-300 bg-white px-3 text-lg dark:border-neutral-700 dark:bg-neutral-900"
        />
      </label>
      <button
        type="submit"
        disabled={locked || !value.trim()}
        className="min-h-12 w-full rounded-xl bg-indigo-600 px-4 py-3 text-base font-semibold text-white disabled:opacity-40"
      >
        {t('liveQuiz.answer.submit')}
      </button>
    </form>
  )
}
