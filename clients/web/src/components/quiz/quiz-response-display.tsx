import { MathPlainText } from '../math/math-plain-text'
import type { QuizQuestion } from '../../lib/courses-api'
import { formatQuizResponseText } from './quiz-response-format'

export function QuizResponseDisplay({
  responseJson,
  questionType,
  choices,
}: {
  responseJson: unknown
  questionType: string
  choices?: QuizQuestion['choices']
}) {
  const text = formatQuizResponseText(responseJson, questionType, choices ?? null)
  if (!text) {
    return <p className="text-sm italic text-slate-500 dark:text-neutral-400">No answer recorded.</p>
  }
  return (
    <p className="whitespace-pre-wrap text-sm text-slate-800 dark:text-neutral-100">
      <MathPlainText text={text} />
    </p>
  )
}