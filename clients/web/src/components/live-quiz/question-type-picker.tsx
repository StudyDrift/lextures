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
      <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-200">
        {t('liveQuiz.editor.questionType')}
      </span>
      <select
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value as LiveQuizQuestionType)}
        className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
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
