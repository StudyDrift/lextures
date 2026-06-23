import type { ConditionalRouterCondition, ConditionalRouterConditionField } from './types'

const INTRINSIC_FIELDS = new Set<ConditionalRouterConditionField>([
  'submissionLength',
  'wordCount',
  'isEmpty',
  'isLate',
  'submissionText',
  'matchesRegex',
])

export function routerFieldRequiresUpstreamGrade(field: ConditionalRouterConditionField): boolean {
  return field === 'score' || field === 'confidence'
}

export function routerFieldRequiresOriginality(field: ConditionalRouterConditionField): boolean {
  return field === 'originalityScore'
}

export function isIntrinsicRouterField(field: ConditionalRouterConditionField): boolean {
  return INTRINSIC_FIELDS.has(field)
}

export function formatRouterConditionSentence(condition: ConditionalRouterCondition): string {
  if (condition.field === 'isEmpty' || condition.field === 'isLate') {
    return `${condition.field} is true`
  }
  if (condition.field === 'submissionText' || condition.field === 'matchesRegex') {
    return `${condition.field} ${condition.operator} "${String(condition.value)}"`
  }
  return `${condition.field} ${condition.operator} ${String(condition.value)}`
}

export function operatorsForRouterField(
  field: ConditionalRouterConditionField,
): ConditionalRouterCondition['operator'][] {
  if (field === 'isEmpty' || field === 'isLate') {
    return ['isTrue']
  }
  if (field === 'submissionText' || field === 'matchesRegex') {
    return ['contains', 'matchesRegex']
  }
  return ['<', '<=', '==', '>=', '>']
}

export function defaultValueForRouterField(field: ConditionalRouterConditionField): string | number | boolean {
  if (field === 'isEmpty' || field === 'isLate') return true
  if (field === 'submissionText' || field === 'matchesRegex') return ''
  if (field === 'confidence') return 0.6
  return 0
}
