import { useTranslation } from 'react-i18next'

type Props = {
  mediaRef: string
  mediaAlt: string
  onChange: (next: { mediaRef: string; mediaAlt: string }) => void
  disabled?: boolean
}

/** Stores a storage-object key + required alt/caption (upload via existing course files flow). */
export function MediaAttach({ mediaRef, mediaAlt, onChange, disabled }: Props) {
  const { t } = useTranslation('common')
  return (
    <div className="space-y-2 rounded-md border border-slate-200 p-3 dark:border-neutral-700">
      <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">
        {t('liveQuiz.editor.promptMedia')}
      </p>
      <p className="text-xs text-slate-500 dark:text-neutral-400">{t('liveQuiz.editor.mediaHint')}</p>
      <label className="block text-sm">
        <span className="mb-1 block text-slate-600 dark:text-neutral-300">
          {t('liveQuiz.editor.mediaRef')}
        </span>
        <input
          value={mediaRef}
          disabled={disabled}
          onChange={(e) => onChange({ mediaRef: e.target.value, mediaAlt })}
          placeholder="course-files/…"
          className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
        />
      </label>
      <label className="block text-sm">
        <span className="mb-1 block text-slate-600 dark:text-neutral-300">
          {t('liveQuiz.editor.mediaAlt')}
        </span>
        <input
          value={mediaAlt}
          disabled={disabled}
          onChange={(e) => onChange({ mediaRef, mediaAlt: e.target.value })}
          className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
        />
      </label>
    </div>
  )
}
