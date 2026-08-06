import { useTranslation } from 'react-i18next'
import type { LiveQuizQuestionType } from '../../lib/live-quiz-api'

const TYPES: LiveQuizQuestionType[] = [
  'mc_single',
  'mc_multiple',
  'true_false',
  'type_answer',
  'numeric',
  'poll',
  'ordering',
  'word_cloud',
]

type Props = {
  value: LiveQuizQuestionType
  onChange: (t: LiveQuizQuestionType) => void
  disabled?: boolean
}

export function QuestionTypePicker({ value, onChange, disabled }: Props) {
  const { t } = useTranslation('common')
  return (
    <label className="block text-sm">
      <span className="mb-1 block font-medium text-fg-default">
        {t('liveQuiz.editor.questionType')}
      </span>
      <select
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value as LiveQuizQuestionType)}
        className="w-full min-h-11 rounded-md border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default"
      >
        {TYPES.map((type) => (
          <option key={type} value={type}>
            {t(`liveQuiz.qtype.${type}`)}
          </option>
        ))}
      </select>
    </label>
  )
}
