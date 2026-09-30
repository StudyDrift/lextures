import { Field, Select } from '../ui'

export type LearnerSwitcherOption = {
  id: string
  name: string
}

export function LearnerSwitcher({
  label,
  allLabel,
  value,
  learners,
  onChange,
}: {
  label: string
  allLabel: string
  value: string
  learners: LearnerSwitcherOption[]
  onChange: (id: string) => void
}) {
  if (learners.length === 0) return null
  return (
    <div className="mt-6 max-w-sm">
      <Field label={label} htmlFor="learner-switcher">
        <Select id="learner-switcher" value={value} onChange={(e) => onChange(e.target.value)}>
          <option value="">{allLabel}</option>
          {learners.map((learner) => (
            <option key={learner.id} value={learner.id}>
              {learner.name}
            </option>
          ))}
        </Select>
      </Field>
    </div>
  )
}
