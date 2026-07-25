/** Shared K-12 grade level tokens (plan 13.6). */

export type GradeLevelOption = {
  value: string
  label: string
}

export const GRADE_LEVEL_OPTIONS: GradeLevelOption[] = [
  { value: 'K', label: 'Kindergarten' },
  { value: '1', label: 'Grade 1' },
  { value: '2', label: 'Grade 2' },
  { value: '3', label: 'Grade 3' },
  { value: '4', label: 'Grade 4' },
  { value: '5', label: 'Grade 5' },
  { value: '6', label: 'Grade 6' },
  { value: '7', label: 'Grade 7' },
  { value: '8', label: 'Grade 8' },
  { value: '9', label: 'Grade 9' },
  { value: '10', label: 'Grade 10' },
  { value: '11', label: 'Grade 11' },
  { value: '12', label: 'Grade 12' },
  { value: 'K-2', label: 'K–2 (multi-grade)' },
  { value: '3-5', label: '3–5 (multi-grade)' },
  { value: '6-8', label: '6–8 (multi-grade)' },
  { value: '9-12', label: '9–12 (multi-grade)' },
  { value: 'K-12', label: 'K–12 (all grades)' },
]

const labelByValue = new Map(GRADE_LEVEL_OPTIONS.map((o) => [o.value, o.label]))

export function gradeLevelLabel(value: string): string {
  return labelByValue.get(value) ?? value
}

/** Stable sort matching GRADE_LEVEL_OPTIONS order. */
export function sortGradeLevels(values: string[]): string[] {
  const order = new Map(GRADE_LEVEL_OPTIONS.map((o, i) => [o.value, i]))
  return [...values].sort((a, b) => {
    const ia = order.get(a) ?? 999
    const ib = order.get(b) ?? 999
    return ia - ib
  })
}

export function formatGradeLevelsSummary(values: string[]): string {
  if (values.length === 0) return 'No grade level — higher ed or unspecified'
  if (values.length === 1) return gradeLevelLabel(values[0]!)
  if (values.length <= 3) {
    return sortGradeLevels(values)
      .map((v) => gradeLevelLabel(v))
      .join(', ')
  }
  return `${values.length} grade levels selected`
}

/** Normalize API payload: prefer gradeLevels array, fall back to single gradeLevel. */
export function gradeLevelsFromCourse(course: {
  gradeLevels?: string[] | null
  gradeLevel?: string | null
}): string[] {
  if (Array.isArray(course.gradeLevels) && course.gradeLevels.length > 0) {
    return sortGradeLevels(
      course.gradeLevels.map((g) => g.trim()).filter(Boolean),
    )
  }
  const single = course.gradeLevel?.trim()
  return single ? [single] : []
}

export function gradeLevelsEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  const sa = sortGradeLevels(a)
  const sb = sortGradeLevels(b)
  return sa.every((v, i) => v === sb[i])
}
