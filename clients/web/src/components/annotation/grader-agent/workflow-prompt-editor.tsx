import {
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from 'react'
import { useTranslation } from 'react-i18next'
import {
  EXPANDABLE_TEXTAREA_FIELD_CLASSES,
  InspectorTextareaExpandButton,
  InspectorTextareaExpandModal,
} from './inspector-textarea-expand-modal'
import type { GraderWorkflowGraph } from './types'
import {
  filterPromptVariableNodes,
  filterPromptVariableProperties,
  findPromptVariableNode,
  getPromptVariableState,
  type PromptVariableNode,
  type PromptVariableProperty,
  type WorkflowNodeDefaultLabels,
  workflowPromptVariableNodes,
} from './workflow-prompt-variable'
import {
  resolveTextareaPickerPosition,
  type TextareaPickerPosition,
} from './workflow-prompt-caret-position'

type WorkflowPromptEditorProps = {
  value: string
  onChange: (value: string) => void
  graph: GraderWorkflowGraph | null
  promptNodeId: string
  defaults: WorkflowNodeDefaultLabels
  disabled?: boolean
  rows?: number
  expandedRows?: number
  className?: string
  placeholder?: string
  expandTitle?: string
  autoFocus?: boolean
  fillHeight?: boolean
}

type PickerRow =
  | { kind: 'node'; node: PromptVariableNode }
  | { kind: 'property'; node: PromptVariableNode; property: PromptVariableProperty }

export function WorkflowPromptEditor({
  value,
  onChange,
  graph,
  promptNodeId,
  defaults,
  disabled = false,
  rows = 6,
  expandedRows = 20,
  className,
  placeholder,
  expandTitle,
  autoFocus = false,
  fillHeight = false,
}: WorkflowPromptEditorProps) {
  const { t } = useTranslation('common')
  const [expanded, setExpanded] = useState(false)
  const [expandedDraft, setExpandedDraft] = useState(value)
  const listId = useId()
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [caret, setCaret] = useState(value.length)
  const [activeIndex, setActiveIndex] = useState(0)
  const [suppressedStart, setSuppressedStart] = useState<number | null>(null)
  const [pickerPosition, setPickerPosition] = useState<TextareaPickerPosition | null>(null)
  const expandable = Boolean(expandTitle) && !disabled

  const variableNodes = useMemo(
    () => workflowPromptVariableNodes(graph, promptNodeId, defaults),
    [graph, promptNodeId, defaults],
  )

  const variableState = useMemo(() => getPromptVariableState(value, caret), [value, caret])

  const rowsForPicker = useMemo((): PickerRow[] => {
    if (!variableState) return []
    if (variableState.kind === 'node') {
      return filterPromptVariableNodes(variableNodes, variableState.query).map((node) => ({
        kind: 'node' as const,
        node,
      }))
    }
    const node = findPromptVariableNode(variableNodes, variableState.nodeQuery)
    return filterPromptVariableProperties(node, variableState.propertyQuery).map((property) => ({
      kind: 'property' as const,
      node: node!,
      property,
    }))
  }, [variableNodes, variableState])

  const listOpen = Boolean(
    variableState &&
      !disabled &&
      rowsForPicker.length > 0 &&
      variableState.start !== suppressedStart,
  )
  const variablePickerQuery =
    variableState?.kind === 'node' ? variableState.query : variableState?.nodeQuery

  useEffect(() => {
    if (!variableState) setSuppressedStart(null)
  }, [variableState])

  useEffect(() => {
    if (variableState && suppressedStart !== null && variableState.start !== suppressedStart) {
      setSuppressedStart(null)
    }
  }, [variableState, suppressedStart])

  useEffect(() => {
    setActiveIndex(0)
  }, [variableState?.kind, variableState?.start, variablePickerQuery])

  useEffect(() => {
    if (rowsForPicker.length === 0) return
    setActiveIndex((index) => Math.min(index, rowsForPicker.length - 1))
  }, [rowsForPicker.length])

  useEffect(() => {
    if (!autoFocus || !textareaRef.current) return
    textareaRef.current.focus()
  }, [autoFocus])

  const syncCaret = useCallback((element: HTMLTextAreaElement) => {
    setCaret(element.selectionStart ?? value.length)
  }, [value.length])

  const syncPickerPosition = useCallback(() => {
    const textarea = textareaRef.current
    if (!textarea || !listOpen) {
      setPickerPosition(null)
      return
    }
    const caretIndex = textarea.selectionStart ?? caret
    setPickerPosition(resolveTextareaPickerPosition(textarea, caretIndex))
  }, [caret, listOpen])

  useLayoutEffect(() => {
    syncPickerPosition()
  }, [syncPickerPosition, value, rows])

  useEffect(() => {
    if (!listOpen) return
    const textarea = textareaRef.current
    if (!textarea) return

    const onScroll = () => syncPickerPosition()
    textarea.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll)

    return () => {
      textarea.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [listOpen, syncPickerPosition])

  const applyPick = useCallback(
    (row: PickerRow) => {
      if (!variableState || !textareaRef.current) return
      const element = textareaRef.current
      const tokenStart = variableState.start
      const pos = element.selectionStart ?? value.length
      const insertion =
        row.kind === 'node'
          ? `$${row.node.variableName}.`
          : `$${row.node.variableName}.${row.property.property}`
      const next = `${value.slice(0, tokenStart)}${insertion}${value.slice(pos)}`
      if (row.kind === 'property') setSuppressedStart(tokenStart)
      onChange(next)
      const nextCaret = tokenStart + insertion.length
      requestAnimationFrame(() => {
        element.focus()
        element.setSelectionRange(nextCaret, nextCaret)
        setCaret(nextCaret)
      })
    },
    [onChange, value, variableState],
  )

  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (!listOpen) return
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        setActiveIndex((index) => (index + 1) % rowsForPicker.length)
        return
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault()
        setActiveIndex((index) => (index - 1 + rowsForPicker.length) % rowsForPicker.length)
        return
      }
      if (event.key === 'Enter' || event.key === 'Tab') {
        event.preventDefault()
        const row = rowsForPicker[activeIndex]
        if (row) applyPick(row)
        return
      }
      if (event.key === 'Escape') {
        event.preventDefault()
        event.stopPropagation()
        if (variableState) setSuppressedStart(variableState.start)
      }
    },
    [activeIndex, applyPick, listOpen, rowsForPicker, variableState],
  )

  const textareaClassName = [
    className,
    expandable ? EXPANDABLE_TEXTAREA_FIELD_CLASSES : '',
    fillHeight ? 'h-full min-h-0 w-full flex-1 resize-none' : '',
  ]
    .filter(Boolean)
    .join(' ')

  const editor = (
    <div className={fillHeight ? 'relative flex min-h-0 flex-1 flex-col' : 'relative'}>
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(event) => {
          onChange(event.target.value)
          syncCaret(event.target)
          setSuppressedStart((prev) => {
            if (prev === null) return prev
            const nextCaret = event.target.selectionStart ?? event.target.value.length
            const state = getPromptVariableState(event.target.value, nextCaret)
            return state?.start === prev ? null : prev
          })
        }}
        onClick={(event) => syncCaret(event.currentTarget)}
        onKeyUp={(event) => syncCaret(event.currentTarget)}
        onSelect={(event) => syncCaret(event.currentTarget)}
        onKeyDown={onKeyDown}
        rows={rows}
        disabled={disabled}
        className={textareaClassName}
        placeholder={placeholder}
        aria-autocomplete="list"
        aria-expanded={listOpen}
        aria-controls={listOpen ? listId : undefined}
        aria-activedescendant={listOpen ? `${listId}-opt-${activeIndex}` : undefined}
      />
      {expandable ? (
        <InspectorTextareaExpandButton
          label={t('gradingAgent.canvas.inspector.expandTextarea')}
          onClick={() => {
            setExpandedDraft(value)
            setExpanded(true)
          }}
        />
      ) : null}
      {listOpen && pickerPosition ? (
        <ul
          id={listId}
          role="listbox"
          style={{
            top: pickerPosition.top,
            left: pickerPosition.left,
            maxWidth: pickerPosition.maxWidth,
          }}
          className="absolute z-20 min-w-48 max-h-48 overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg dark:border-neutral-600 dark:bg-neutral-900"
        >
          {rowsForPicker.map((row, index) => {
            const selected = index === activeIndex
            const primary =
              row.kind === 'node' ? `$${row.node.variableName}` : row.property.property
            const secondary =
              row.kind === 'node'
                ? row.node.displayLabel
                : `$${row.node.variableName}.${row.property.property}`
            return (
              <li
                key={
                  row.kind === 'node'
                    ? `node-${row.node.nodeId}`
                    : `property-${row.node.nodeId}-${row.property.property}`
                }
                id={`${listId}-opt-${index}`}
                role="option"
                aria-selected={selected}
                className={`cursor-pointer px-3 py-2 ${
                  selected
                    ? 'bg-indigo-50 text-indigo-900 dark:bg-indigo-500/15 dark:text-indigo-100'
                    : 'text-slate-800 hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-800'
                }`}
                onMouseDown={(event) => {
                  event.preventDefault()
                  applyPick(row)
                }}
                onMouseEnter={() => setActiveIndex(index)}
              >
                <span className="font-medium">{primary}</span>
                <span className="ms-2 text-xs text-slate-500 dark:text-neutral-400">{secondary}</span>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )

  return (
    <>
      {editor}
      {expanded && expandTitle ? (
        <InspectorTextareaExpandModal
          title={expandTitle}
          onDone={() => {
            onChange(expandedDraft)
            setExpanded(false)
          }}
          onCancel={() => setExpanded(false)}
        >
          <WorkflowPromptEditor
            value={expandedDraft}
            onChange={setExpandedDraft}
            graph={graph}
            promptNodeId={promptNodeId}
            defaults={defaults}
            rows={expandedRows}
            className={className}
            placeholder={placeholder}
            autoFocus
            fillHeight
          />
        </InspectorTextareaExpandModal>
      ) : null}
    </>
  )
}