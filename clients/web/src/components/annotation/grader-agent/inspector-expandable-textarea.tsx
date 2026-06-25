import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  EXPANDABLE_TEXTAREA_FIELD_CLASSES,
  InspectorTextareaExpandModal,
  InspectorTextareaExpandOverlay,
} from './inspector-textarea-expand-modal'

type InspectorExpandableTextareaProps = {
  value: string
  onChange: (value: string) => void
  expandTitle: string
  rows?: number
  expandedRows?: number
  className?: string
  placeholder?: string
  disabled?: boolean
}

export function InspectorExpandableTextarea({
  value,
  onChange,
  expandTitle,
  rows = 6,
  expandedRows = 20,
  className = '',
  placeholder,
  disabled = false,
}: InspectorExpandableTextareaProps) {
  const { t } = useTranslation('common')
  const [expanded, setExpanded] = useState(false)
  const [draftValue, setDraftValue] = useState(value)
  const expandable = !disabled

  const openExpanded = () => {
    setDraftValue(value)
    setExpanded(true)
  }

  const textarea = (
    <textarea
      value={value}
      onChange={(event) => onChange(event.target.value)}
      rows={rows}
      disabled={disabled}
      placeholder={placeholder}
      className={`${className}${expandable ? ` ${EXPANDABLE_TEXTAREA_FIELD_CLASSES}` : ''}`}
    />
  )

  return (
    <>
      {expandable ? (
        <InspectorTextareaExpandOverlay
          label={t('gradingAgent.canvas.inspector.expandTextarea')}
          onExpand={openExpanded}
        >
          {textarea}
        </InspectorTextareaExpandOverlay>
      ) : (
        textarea
      )}
      {expanded ? (
        <InspectorTextareaExpandModal
          title={expandTitle}
          onDone={() => {
            onChange(draftValue)
            setExpanded(false)
          }}
          onCancel={() => setExpanded(false)}
        >
          <textarea
            autoFocus
            value={draftValue}
            onChange={(event) => setDraftValue(event.target.value)}
            rows={expandedRows}
            placeholder={placeholder}
            className={`${className} h-full min-h-0 w-full flex-1 resize-none`}
          />
        </InspectorTextareaExpandModal>
      ) : null}
    </>
  )
}