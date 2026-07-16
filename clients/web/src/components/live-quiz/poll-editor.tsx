import { useTranslation } from 'react-i18next'
import type { LiveQuizOption } from '../../lib/live-quiz-api'
import { McOptionList } from './mc-option-list'

type Props = {
  options: LiveQuizOption[]
  onChange: (options: LiveQuizOption[]) => void
  disabled?: boolean
}

export function PollEditor({ options, onChange, disabled }: Props) {
  const { t } = useTranslation('common')
  return (
    <div className="space-y-2">
      <p className="text-xs text-slate-500 dark:text-neutral-400">{t('liveQuiz.editor.pollHint')}</p>
      <McOptionList options={options} onChange={onChange} allowCorrect={false} disabled={disabled} />
    </div>
  )
}
