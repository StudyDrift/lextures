import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { ShapeIcon } from './answer-shapes'

export function AnswerTrueFalse({
  options,
  locked,
  onSelect,
}: {
  options: Array<{ id: string; text: string }>
  locked: boolean
  onSelect: (id: string) => void
}) {
  const { t } = useTranslation('common')
  const trueOpt = options.find((o) => /true|yes|vrai|verdadero/i.test(o.text)) ?? options[0]
  const falseOpt = options.find((o) => o.id !== trueOpt?.id) ?? options[1]

  useEffect(() => {
    if (locked) return
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
      const k = e.key.toLowerCase()
      if ((k === 't' || k === '1') && trueOpt) {
        e.preventDefault()
        onSelect(trueOpt.id)
      }
      if ((k === 'f' || k === '2') && falseOpt) {
        e.preventDefault()
        onSelect(falseOpt.id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [falseOpt, locked, onSelect, trueOpt])

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {trueOpt && (
        <button
          type="button"
          disabled={locked}
          onClick={() => onSelect(trueOpt.id)}
          className="flex min-h-20 items-center justify-center gap-3 rounded-xl bg-emerald-600 px-4 py-5 text-xl font-semibold text-white disabled:opacity-70"
        >
          <ShapeIcon shape="circle" className="h-8 w-8" />
          {trueOpt.text || t('liveQuiz.answer.true')}
        </button>
      )}
      {falseOpt && (
        <button
          type="button"
          disabled={locked}
          onClick={() => onSelect(falseOpt.id)}
          className="flex min-h-20 items-center justify-center gap-3 rounded-xl bg-rose-600 px-4 py-5 text-xl font-semibold text-white disabled:opacity-70"
        >
          <ShapeIcon shape="square" className="h-8 w-8" />
          {falseOpt.text || t('liveQuiz.answer.false')}
        </button>
      )}
      {!locked && (
        <p className="col-span-full text-center text-xs text-slate-500 dark:text-neutral-400">
          {t('liveQuiz.answer.tfHint')}
        </p>
      )}
    </div>
  )
}
