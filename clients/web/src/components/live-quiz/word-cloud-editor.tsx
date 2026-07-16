import { useTranslation } from 'react-i18next'

/** Word cloud is open short text — no options or correct answers to configure. */
export function WordCloudEditor() {
  const { t } = useTranslation('common')
  return (
    <p className="rounded-md border border-dashed border-slate-300 p-4 text-sm text-slate-600 dark:border-neutral-600 dark:text-neutral-300">
      {t('liveQuiz.editor.wordCloudHint')}
    </p>
  )
}
