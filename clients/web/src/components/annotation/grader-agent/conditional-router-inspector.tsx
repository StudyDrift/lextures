import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  defaultValueForRouterField,
  formatRouterConditionSentence,
  operatorsForRouterField,
} from './router-condition'
import type {
  ConditionalRouterCondition,
  ConditionalRouterConditionField,
  ConditionalRouterNodeData,
  GraderWorkflowGraph,
} from './types'
import { validateRouterIssues } from './router-validation'

const fieldClass =
  'w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100'

type ConditionalRouterInspectorProps = {
  nodeId: string
  graph: GraderWorkflowGraph
  data: Record<string, unknown>
  title: string
  onChange: (patch: Partial<ConditionalRouterNodeData>) => void
  onDelete: () => void
}

const ALL_FIELDS: ConditionalRouterConditionField[] = [
  'isEmpty',
  'submissionLength',
  'wordCount',
  'isLate',
  'confidence',
  'score',
  'originalityScore',
  'submissionText',
  'matchesRegex',
]

function parseCondition(data: Record<string, unknown>): ConditionalRouterCondition {
  const raw = data.condition
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    const c = raw as ConditionalRouterCondition
    if (c.field && c.operator) return c
  }
  return { field: 'isEmpty', operator: 'isTrue', value: true }
}

export function ConditionalRouterInspector({
  nodeId,
  graph,
  data,
  title,
  onChange,
  onDelete,
}: ConditionalRouterInspectorProps) {
  const { t } = useTranslation('common')
  const condition = parseCondition(data)
  const nodeById = useMemo(() => new Map(graph.nodes.map((n) => [n.id, n])), [graph.nodes])
  const fieldIssue = useMemo(
    () => validateRouterIssues(graph, nodeById).find((i) => i.field === `node:${nodeId}.condition.field`),
    [graph, nodeById, nodeId],
  )
  const branchIssues = useMemo(
    () => validateRouterIssues(graph, nodeById).filter((i) => i.field.startsWith(`node:${nodeId}.`)),
    [graph, nodeById, nodeId],
  )

  const updateCondition = (patch: Partial<ConditionalRouterCondition>) => {
    const next = { ...condition, ...patch }
    if (patch.field && patch.field !== condition.field) {
      next.operator = operatorsForRouterField(patch.field)[0] ?? '=='
      next.value = defaultValueForRouterField(patch.field)
    }
    onChange({ condition: next })
  }

  const showNumericValue = !['isEmpty', 'isLate'].includes(condition.field)
  const showTextValue = condition.field === 'submissionText' || condition.field === 'matchesRegex'

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p className="font-medium text-slate-800 dark:text-neutral-100">{title}</p>
      <p>{t('gradingAgent.canvas.inspector.routerHelp')}</p>

      <fieldset className="space-y-3">
        <legend className="sr-only">{t('gradingAgent.canvas.inspector.conditionBuilder')}</legend>
        <label className="block">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.conditionField')}</span>
          <select
            value={condition.field}
            onChange={(e) => updateCondition({ field: e.target.value as ConditionalRouterConditionField })}
            className={fieldClass}
          >
            {ALL_FIELDS.map((field) => (
              <option key={field} value={field}>
                {t(`gradingAgent.canvas.inspector.conditionFields.${field}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="block">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.conditionOperator')}</span>
          <select
            value={condition.operator}
            onChange={(e) =>
              updateCondition({ operator: e.target.value as ConditionalRouterCondition['operator'] })
            }
            className={fieldClass}
          >
            {operatorsForRouterField(condition.field).map((op) => (
              <option key={op} value={op}>
                {t(`gradingAgent.canvas.inspector.conditionOperators.${op}`)}
              </option>
            ))}
          </select>
        </label>
        {showNumericValue && !showTextValue ? (
          <label className="block">
            <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.conditionValue')}</span>
            <input
              type="number"
              step="any"
              value={typeof condition.value === 'number' ? condition.value : Number(condition.value) || 0}
              onChange={(e) => updateCondition({ value: Number(e.target.value) })}
              className={fieldClass}
            />
          </label>
        ) : null}
        {showTextValue ? (
          <label className="block">
            <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.conditionValue')}</span>
            <input
              type="text"
              value={typeof condition.value === 'string' ? condition.value : String(condition.value ?? '')}
              onChange={(e) => updateCondition({ value: e.target.value })}
              className={fieldClass}
            />
          </label>
        ) : null}
      </fieldset>

      <p className="rounded-lg bg-slate-100 px-3 py-2 text-xs dark:bg-neutral-800" aria-live="polite">
        {t('gradingAgent.canvas.inspector.conditionPreview', {
          sentence: formatRouterConditionSentence(condition),
        })}
      </p>

      {fieldIssue ? (
        <p className="text-xs text-rose-700 dark:text-rose-300" role="alert">
          {fieldIssue.message}
        </p>
      ) : null}
      {branchIssues
        .filter((i) => i.field.endsWith('.then') || i.field.endsWith('.else'))
        .map((issue) => (
          <p key={issue.field} className="text-xs text-amber-700 dark:text-amber-300" role="status">
            {issue.message}
          </p>
        ))}

      <button
        type="button"
        onClick={onDelete}
        className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
      >
        {t('gradingAgent.canvas.inspector.deleteNode')}
      </button>
    </div>
  )
}
