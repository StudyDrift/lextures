import { useTranslation } from 'react-i18next'

/** Word cloud is open short text — no options or correct answers to configure. */
export function WordCloudEditor() {
  const { t } = useTranslation('common')
  return (
    <p className="rounded-md border border-dashed border-border-strong p-4 text-sm text-fg-muted dark:border-border-default dark:text-fg-muted">
      {t('liveQuiz.editor.wordCloudHint')}
    </p>
  )
}
