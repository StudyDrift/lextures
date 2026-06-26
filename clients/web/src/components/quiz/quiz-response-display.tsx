import { MathPlainText } from '../math/math-plain-text'
import { MarkdownArticleView } from '../syllabus/syllabus-markdown-view'
import type { QuizQuestion } from '../../lib/courses-api'
import { formatQuizResponseText } from './quiz-response-format'

// Free-text answer types whose stored value is Markdown (Canvas HTML is converted on import).
const MARKDOWN_ANSWER_TYPES = new Set(['essay', 'short_answer', 'fill_in_blank'])

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
  if (MARKDOWN_ANSWER_TYPES.has(questionType)) {
    return (
      <div className="text-sm text-slate-800 dark:text-neutral-100">
        <MarkdownArticleView markdown={text} />
      </div>
    )
  }
  return (
    <p className="whitespace-pre-wrap text-sm text-slate-800 dark:text-neutral-100">
      <MathPlainText text={text} />
    </p>
  )
}
