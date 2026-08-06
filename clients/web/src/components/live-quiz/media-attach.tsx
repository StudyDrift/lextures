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
    <div className="space-y-2 rounded-md border border-border-default p-3 dark:border-border-default">
      <p className="text-sm font-medium text-fg-default">
        {t('liveQuiz.editor.promptMedia')}
      </p>
      <p className="text-xs text-fg-muted">{t('liveQuiz.editor.mediaHint')}</p>
      <label className="block text-sm">
        <span className="mb-1 block text-fg-muted">
          {t('liveQuiz.editor.mediaRef')}
        </span>
        <input
          value={mediaRef}
          disabled={disabled}
          onChange={(e) => onChange({ mediaRef: e.target.value, mediaAlt })}
          placeholder="course-files/…"
          className="w-full min-h-11 rounded-md border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default"
        />
      </label>
      <label className="block text-sm">
        <span className="mb-1 block text-fg-muted">
          {t('liveQuiz.editor.mediaAlt')}
        </span>
        <input
          value={mediaAlt}
          disabled={disabled}
          onChange={(e) => onChange({ mediaRef, mediaAlt: e.target.value })}
          className="w-full min-h-11 rounded-md border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default"
        />
      </label>
    </div>
  )
}
