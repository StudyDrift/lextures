import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import type { QuizQuestion } from '../../../lib/courses-api'
import { MathPlainText } from '../../math/math-plain-text'
import {
  buildQuizAnswerPreview,
  quizQuestionForSlot,
  type QuizAnswerPreview,
} from './quiz-question-preview'
import type { QuizQuestionSlot } from './quiz-question-slots'

type QuizResponsesInspectorProps = {
  slots: QuizQuestionSlot[]
  questions: QuizQuestion[]
}

const QUESTION_TYPE_KEYS: Record<string, string> = {
  multiple_choice: 'gradingAgent.canvas.inspector.quizResponses.type.multipleChoice',
  true_false: 'gradingAgent.canvas.inspector.quizResponses.type.trueFalse',
  fill_in_blank: 'gradingAgent.canvas.inspector.quizResponses.type.fillInBlank',
  short_answer: 'gradingAgent.canvas.inspector.quizResponses.type.shortAnswer',
  essay: 'gradingAgent.canvas.inspector.quizResponses.type.essay',
  matching: 'gradingAgent.canvas.inspector.quizResponses.type.matching',
  ordering: 'gradingAgent.canvas.inspector.quizResponses.type.ordering',
  hotspot: 'gradingAgent.canvas.inspector.quizResponses.type.hotspot',
  numeric: 'gradingAgent.canvas.inspector.quizResponses.type.numeric',
  formula: 'gradingAgent.canvas.inspector.quizResponses.type.formula',
  code: 'gradingAgent.canvas.inspector.quizResponses.type.code',
  file_upload: 'gradingAgent.canvas.inspector.quizResponses.type.fileUpload',
  audio_response: 'gradingAgent.canvas.inspector.quizResponses.type.audioResponse',
  video_response: 'gradingAgent.canvas.inspector.quizResponses.type.videoResponse',
}

function questionTypeLabel(questionType: string, t: TFunction<'common'>): string {
  const key = QUESTION_TYPE_KEYS[questionType]
  return key ? t(key) : questionType.replaceAll('_', ' ')
}

function AnswerPreview({ preview, t }: { preview: QuizAnswerPreview; t: TFunction<'common'> }) {
  switch (preview.kind) {
    case 'choices':
      if (preview.labels.length === 0) {
        return (
          <p className="text-xs italic text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.inspector.quizResponses.noChoices')}
          </p>
        )
      }
      return (
        <ol className="mt-1.5 list-decimal space-y-1 ps-4 text-sm text-slate-700 dark:text-neutral-200">
          {preview.labels.map((label, choiceIndex) => (
            <li key={choiceIndex} className="break-words">
              <MathPlainText text={label} />
            </li>
          ))}
        </ol>
      )
    case 'matching':
      if (preview.pairs.length === 0) {
        return (
          <p className="text-xs italic text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.inspector.quizResponses.noMatchingPairs')}
          </p>
        )
      }
      return (
        <ul className="mt-1.5 space-y-1.5 text-sm text-slate-700 dark:text-neutral-200">
          {preview.pairs.map((pair, pairIndex) => (
            <li
              key={pairIndex}
              className="rounded-lg border border-slate-200 bg-slate-50/80 px-2.5 py-2 dark:border-neutral-700 dark:bg-neutral-950/60"
            >
              <span className="font-medium text-slate-600 dark:text-neutral-300">
                <MathPlainText text={pair.left || '—'} />
              </span>
              <span className="mx-1.5 text-slate-400 dark:text-neutral-500" aria-hidden>
                →
              </span>
              <span>
                <MathPlainText text={pair.right || '—'} />
              </span>
            </li>
          ))}
        </ul>
      )
    case 'ordering':
      if (preview.items.length === 0) {
        return (
          <p className="text-xs italic text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.inspector.quizResponses.noOrderingItems')}
          </p>
        )
      }
      return (
        <ol className="mt-1.5 list-decimal space-y-1 ps-4 text-sm text-slate-700 dark:text-neutral-200">
          {preview.items.map((item, itemIndex) => (
            <li key={itemIndex} className="break-words">
              <MathPlainText text={item} />
            </li>
          ))}
        </ol>
      )
    case 'code':
      return (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('gradingAgent.canvas.inspector.quizResponses.codeLanguage', { language: preview.language })}
        </p>
      )
    case 'media':
      return (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {preview.mediaKind === 'file'
            ? t('gradingAgent.canvas.inspector.quizResponses.fileUpload')
            : preview.mediaKind === 'audio'
              ? t('gradingAgent.canvas.inspector.quizResponses.audioResponse')
              : t('gradingAgent.canvas.inspector.quizResponses.videoResponse')}
        </p>
      )
    case 'hotspot':
      return (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('gradingAgent.canvas.inspector.quizResponses.hotspot')}
        </p>
      )
    case 'numeric':
      return (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('gradingAgent.canvas.inspector.quizResponses.numeric')}
        </p>
      )
    case 'formula':
      return (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('gradingAgent.canvas.inspector.quizResponses.formula')}
        </p>
      )
    case 'open':
      return (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('gradingAgent.canvas.inspector.quizResponses.openResponse')}
        </p>
      )
    default:
      return null
  }
}

function SlotPreviewCard({
  slot,
  question,
  t,
}: {
  slot: QuizQuestionSlot
  question: QuizQuestion | null
  t: TFunction<'common'>
}) {
  const preview = question ? buildQuizAnswerPreview(question) : null
  const showAnswers =
    preview?.kind === 'choices' || preview?.kind === 'matching' || preview?.kind === 'ordering'

  return (
    <article className="rounded-lg border border-slate-200 bg-white p-3 dark:border-neutral-700 dark:bg-neutral-950">
      <div className="flex flex-wrap items-center gap-2">
        <h4 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">{slot.label}</h4>
        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
          {questionTypeLabel(slot.questionType, t)}
        </span>
        <span className="text-[10px] font-medium text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.inspector.quizResponses.points', { points: slot.maxPoints })}
        </span>
        {slot.isPoolSlot ? (
          <span className="rounded-full bg-violet-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:text-violet-300">
            {t('gradingAgent.canvas.nodes.quizResponses.poolBadge')}
          </span>
        ) : null}
        {slot.isShuffled ? (
          <span className="rounded-full bg-violet-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:text-violet-300">
            {t('gradingAgent.canvas.nodes.quizResponses.shuffledBadge')}
          </span>
        ) : null}
      </div>

      {slot.isPoolSlot ? (
        <p className="mt-2 text-xs text-violet-700 dark:text-violet-300">
          {t('gradingAgent.canvas.inspector.quizResponses.poolNote')}
        </p>
      ) : null}
      {slot.isShuffled ? (
        <p className="mt-2 text-xs text-violet-700 dark:text-violet-300">
          {t('gradingAgent.canvas.inspector.quizResponses.shuffledNote')}
        </p>
      ) : null}

      {question ? (
        <>
          <p className="mt-2 text-sm font-medium text-slate-800 dark:text-neutral-100">
            <MathPlainText text={question.prompt?.trim() || '—'} />
          </p>
          <div className="mt-3">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              {showAnswers
                ? t('gradingAgent.canvas.inspector.quizResponses.answersHeading')
                : t('gradingAgent.canvas.inspector.quizResponses.responseHeading')}
            </p>
            {preview ? <AnswerPreview preview={preview} t={t} /> : null}
          </div>
        </>
      ) : (
        <p className="mt-2 text-sm text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.inspector.quizResponses.missingQuestion')}
        </p>
      )}
    </article>
  )
}

export function QuizResponsesInspector({ slots, questions }: QuizResponsesInspectorProps) {
  const { t } = useTranslation('common')

  if (slots.length === 0) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.canvas.inspector.quizResponses.empty')}
      </p>
    )
  }

  return (
    <div className="space-y-3">
      {slots.map((slot) => (
        <SlotPreviewCard
          key={slot.index}
          slot={slot}
          question={quizQuestionForSlot(questions, slot.index)}
          t={t}
        />
      ))}
    </div>
  )
}