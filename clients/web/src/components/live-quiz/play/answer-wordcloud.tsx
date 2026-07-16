import { AnswerType } from './answer-type'

/** Word-cloud uses the same short-text submit surface as type-answer. */
export function AnswerWordCloud({
  locked,
  onSubmit,
}: {
  locked: boolean
  onSubmit: (text: string) => void
}) {
  return <AnswerType locked={locked} onSubmit={onSubmit} />
}
