import { useEffect, useState } from 'react'
import { Button, Dialog, InlineAlert, SegmentedControl } from '../ui'
import { disableLearnerSignIn, setLearnerSignIn } from '../../lib/family-sign-in-api'
import type { ManagedDependent } from '../../lib/managed-learners-api'
import { PictureSequenceInput } from './picture-sequence-input'
import { PinPad } from './pin-pad'
import { PICTURE_MAX, PICTURE_MIN, PIN_MAX, PIN_MIN, isValidPin } from './picture-choices'

type Method = 'pin' | 'picture'

type Props = {
  learner: ManagedDependent | null
  onClose: () => void
  onSaved: (updated: ManagedDependent) => void
}

/** Parent sets, resets, or turns off a learner's family-device sign-in (#771). */
export function LearnerSignInDialog({ learner, onClose, onSaved }: Props) {
  const [method, setMethod] = useState<Method>('pin')
  const [pin, setPin] = useState('')
  const [confirmPin, setConfirmPin] = useState('')
  const [picture, setPicture] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!learner) return
    setMethod(learner.signInMethod === 'picture' ? 'picture' : 'pin')
    setPin('')
    setConfirmPin('')
    setPicture([])
    setError(null)
  }, [learner])

  const enabled = learner?.signInMethod === 'pin' || learner?.signInMethod === 'picture'
  const pinValid = isValidPin(pin) && pin === confirmPin
  const pictureValid = picture.length >= PICTURE_MIN && picture.length <= PICTURE_MAX
  const canSave = method === 'pin' ? pinValid : pictureValid

  async function save() {
    if (!learner || !canSave) return
    setBusy(true)
    setError(null)
    try {
      const updated =
        method === 'pin'
          ? await setLearnerSignIn(learner.id, { method: 'pin', pin })
          : await setLearnerSignIn(learner.id, { method: 'picture', picture })
      onSaved(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save sign-in')
    } finally {
      setBusy(false)
    }
  }

  async function turnOff() {
    if (!learner) return
    setBusy(true)
    setError(null)
    try {
      onSaved(await disableLearnerSignIn(learner.id))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to turn off sign-in')
    } finally {
      setBusy(false)
    }
  }

  const name = learner?.displayName ?? ''
  return (
    <Dialog
      open={learner !== null}
      onClose={() => {
        if (!busy) onClose()
      }}
      title={enabled ? `Reset sign-in for ${name}` : `Set up sign-in for ${name}`}
      description={`${name} can sign in on a family device from the "Who's learning?" screen without your password.`}
      closeLabel="Close"
      footer={
        <div className="flex w-full flex-wrap items-center justify-between gap-2">
          {enabled ? (
            <Button type="button" variant="danger" size="sm" disabled={busy} onClick={() => void turnOff()}>
              Turn off sign-in
            </Button>
          ) : (
            <span />
          )}
          <div className="flex gap-2">
            <Button type="button" variant="secondary" disabled={busy} onClick={onClose}>
              Cancel
            </Button>
            <Button type="button" disabled={!canSave} loading={busy} onClick={() => void save()}>
              Save
            </Button>
          </div>
        </div>
      }
    >
      <div className="space-y-4">
        {learner?.signInLockedUntil ? (
          <InlineAlert tone="warning">
            Sign-in is locked after too many tries. Saving a new PIN or picture unlocks it now.
          </InlineAlert>
        ) : null}
        {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
        <SegmentedControl<Method>
          label="Sign-in type"
          value={method}
          onChange={(m) => {
            setMethod(m)
            setError(null)
          }}
          options={[
            { value: 'pin', label: 'PIN' },
            { value: 'picture', label: 'Picture password' },
          ]}
        />
        {method === 'pin' ? (
          <div className="grid gap-3 sm:grid-cols-2">
            <PinPad id="learner-pin" label={`New PIN (${PIN_MIN}–${PIN_MAX} digits)`} value={pin} onChange={setPin} disabled={busy} />
            <PinPad id="learner-pin-confirm" label="Confirm PIN" value={confirmPin} onChange={setConfirmPin} disabled={busy} />
            {confirmPin.length > 0 && pin !== confirmPin ? (
              <p className="text-caption text-danger-fg sm:col-span-2">PINs don&apos;t match.</p>
            ) : null}
          </div>
        ) : (
          <div className="space-y-2">
            <p className="text-body-sm text-fg-muted">
              Pick {PICTURE_MIN}–{PICTURE_MAX} pictures in order. Great for younger learners who can&apos;t read numbers
              yet.
            </p>
            <PictureSequenceInput label="Picture password" value={picture} onChange={setPicture} disabled={busy} />
          </div>
        )}
      </div>
    </Dialog>
  )
}
