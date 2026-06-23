import { useCallback, useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from 'react'
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

type WorkflowPromptEditorProps = {
  value: string
  onChange: (value: string) => void
  graph: GraderWorkflowGraph | null
  promptNodeId: string
  defaults: WorkflowNodeDefaultLabels
  disabled?: boolean
  rows?: number
  className?: string
  placeholder?: string
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
  className,
  placeholder,
}: WorkflowPromptEditorProps) {
  const listId = useId()
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [caret, setCaret] = useState(value.length)
  const [activeIndex, setActiveIndex] = useState(0)

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

  const listOpen = Boolean(variableState && !disabled && rowsForPicker.length > 0)
  const variablePickerQuery =
    variableState?.kind === 'node' ? variableState.query : variableState?.nodeQuery

  useEffect(() => {
    setActiveIndex(0)
  }, [variableState?.kind, variableState?.start, variablePickerQuery])

  useEffect(() => {
    if (rowsForPicker.length === 0) return
    setActiveIndex((index) => Math.min(index, rowsForPicker.length - 1))
  }, [rowsForPicker.length])

  const syncCaret = useCallback((element: HTMLTextAreaElement) => {
    setCaret(element.selectionStart ?? value.length)
  }, [value.length])

  const applyPick = useCallback(
    (row: PickerRow) => {
      if (!variableState || !textareaRef.current) return
      const element = textareaRef.current
      const pos = element.selectionStart ?? value.length
      const insertion =
        row.kind === 'node'
          ? `$${row.node.variableName}.`
          : `$${row.node.variableName}.${row.property.property}`
      const next = `${value.slice(0, variableState.start)}${insertion}${value.slice(pos)}`
      onChange(next)
      const nextCaret = variableState.start + insertion.length
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
      }
    },
    [activeIndex, applyPick, listOpen, rowsForPicker],
  )

  return (
    <div className="relative">
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(event) => {
          onChange(event.target.value)
          syncCaret(event.target)
        }}
        onClick={(event) => syncCaret(event.currentTarget)}
        onKeyUp={(event) => syncCaret(event.currentTarget)}
        onSelect={(event) => syncCaret(event.currentTarget)}
        onKeyDown={onKeyDown}
        rows={rows}
        disabled={disabled}
        className={className}
        placeholder={placeholder}
        aria-autocomplete="list"
        aria-expanded={listOpen}
        aria-controls={listOpen ? listId : undefined}
        aria-activedescendant={listOpen ? `${listId}-opt-${activeIndex}` : undefined}
      />
      {listOpen ? (
        <ul
          id={listId}
          role="listbox"
          className="absolute z-20 mt-1 max-h-48 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg dark:border-neutral-600 dark:bg-neutral-900"
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
}