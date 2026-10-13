import { Radio, RadioGroup } from '../ui'
import type { MessageRecipient } from '../../lib/communication-api'
import { recipientLabel } from '../../lib/inbox-copy'

/**
 * Kid-safe Compose "To" (#775): a managed learner picks from the people their parent allows.
 * There is no free-text address field.
 */
export function KidRecipientPicker({
  recipients,
  value,
  onChange,
  loading,
  error,
}: {
  recipients: MessageRecipient[]
  value: string
  onChange: (id: string) => void
  loading?: boolean
  error?: string | null
}) {
  if (loading) {
    return <p className="text-sm text-fg-muted">Loading who you can message…</p>
  }
  if (error) {
    return (
      <p className="text-sm text-danger-fg" role="alert">
        {error}
      </p>
    )
  }
  if (recipients.length === 0) {
    return (
      <p className="text-sm text-fg-muted">
        No one to message yet. Ask your parent to set up messaging for you.
      </p>
    )
  }
  return (
    <RadioGroup name="kid-compose-to" legend="Send to" value={value} onChange={onChange}>
      {recipients.map((r) => (
        <Radio key={r.id} value={r.id} label={recipientLabel(r)} />
      ))}
    </RadioGroup>
  )
}
