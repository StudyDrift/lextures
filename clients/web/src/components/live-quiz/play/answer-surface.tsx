import { useState } from 'react'
import type { LiveAnswerPayload } from '../../../lib/live-quiz-realtime'
import { AnswerGrid } from './answer-grid'
import { AnswerNumeric } from './answer-numeric'
import { AnswerOrdering } from './answer-ordering'
import { AnswerTrueFalse } from './answer-truefalse'
import { AnswerType } from './answer-type'
import { AnswerWordCloud } from './answer-wordcloud'

export function AnswerSurface({
  questionType,
  options,
  locked,
  onAnswer,
}: {
  questionType: string
  options: Array<{ id: string; text: string }>
  locked: boolean
  onAnswer: (payload: LiveAnswerPayload) => void
}) {
  const [selected, setSelected] = useState<string[]>([])

  switch (questionType) {
    case 'mc_single':
    case 'poll':
      return (
        <AnswerGrid
          options={options}
          locked={locked}
          selectedIds={selected}
          onSelect={(id) => {
            if (locked) return
            setSelected([id])
            onAnswer({ optionId: id })
          }}
        />
      )
    case 'mc_multiple':
      return (
        <AnswerGrid
          options={options}
          locked={locked}
          multi
          selectedIds={selected}
          onSelect={(id) => {
            if (locked) return
            setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
          }}
          onSubmitMulti={() => {
            if (locked || selected.length === 0) return
            onAnswer({ optionIds: selected })
          }}
        />
      )
    case 'true_false':
      return (
        <AnswerTrueFalse
          options={options}
          locked={locked}
          onSelect={(id) => {
            if (locked) return
            onAnswer({ optionId: id })
          }}
        />
      )
    case 'type_answer':
      return (
        <AnswerType
          locked={locked}
          onSubmit={(text) => {
            if (locked) return
            onAnswer({ text })
          }}
        />
      )
    case 'numeric':
      return (
        <AnswerNumeric
          locked={locked}
          onSubmit={(value) => {
            if (locked) return
            onAnswer({ value })
          }}
        />
      )
    case 'ordering':
      return (
        <AnswerOrdering
          options={options}
          locked={locked}
          onSubmit={(order) => {
            if (locked) return
            onAnswer({ order })
          }}
        />
      )
    case 'word_cloud':
      return (
        <AnswerWordCloud
          locked={locked}
          onSubmit={(text) => {
            if (locked) return
            onAnswer({ text })
          }}
        />
      )
    default:
      return (
        <AnswerGrid
          options={options}
          locked={locked}
          selectedIds={selected}
          onSelect={(id) => {
            if (locked) return
            setSelected([id])
            onAnswer({ optionId: id })
          }}
        />
      )
  }
}
