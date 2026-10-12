import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Avatar, Button, InlineAlert, LinkButton, Spinner } from '../../components/ui'
import { PictureSequenceInput } from '../../components/family-sign-in/picture-sequence-input'
import { PinPad } from '../../components/family-sign-in/pin-pad'
import { PICTURE_MIN, PIN_MIN, isValidPin } from '../../components/family-sign-in/picture-choices'
import { isFamilySessionActive } from '../../lib/family-device'
import {
  FamilySignInError,
  fetchFamilyPicker,
  familySignIn,
  signOutFamilySession,
  type FamilyPicker,
  type FamilyPickerLearner,
} from '../../lib/family-sign-in-api'

const PAGE_TITLE = "Who's learning? · Lextures"

function initialsOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  return (parts[0]?.[0] ?? '?').toUpperCase() + (parts[1]?.[0] ?? '').toUpperCase()
}

function lockedMessage(until?: string | null): string {
  if (!until) return 'Too many tries. Ask a grown-up for help.'
  const t = new Date(until)
  return `Too many tries. Try again after ${t.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}, or ask a grown-up.`
}

/** "Who's learning?" — managed learners sign in on a family device (#771). */
export default function FamilySignInPage() {
  const [picker, setPicker] = useState<FamilyPicker | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selected, setSelected] = useState<FamilyPickerLearner | null>(null)
  const [pin, setPin] = useState('')
  const [picture, setPicture] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [sessionActive, setSessionActive] = useState(() => isFamilySessionActive())

  const load = useCallback(async () => {
    setLoading(true)
    setLoadError(null)
    try {
      setPicker(await fetchFamilyPicker())
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : "Failed to load who's learning")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    document.title = PAGE_TITLE
    void load()
  }, [load])

  function choose(learner: FamilyPickerLearner | null) {
    setSelected(learner)
    setPin('')
    setPicture([])
    setError(null)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!selected) return
    setBusy(true)
    setError(null)
    try {
      await familySignIn(selected.id, selected.signInMethod === 'picture' ? { picture } : { pin })
      window.location.assign('/')
    } catch (err) {
      setPin('')
      setPicture([])
      if (err instanceof FamilySignInError && err.status === 429) {
        setSelected({ ...selected, lockedUntil: err.lockedUntil ?? null })
        setError(lockedMessage(err.lockedUntil))
      } else if (err instanceof FamilySignInError && typeof err.attemptsRemaining === 'number') {
        setError(
          `That didn't match. ${err.attemptsRemaining} ${err.attemptsRemaining === 1 ? 'try' : 'tries'} left.`,
        )
      } else {
        setError(err instanceof Error ? err.message : 'Sign-in failed')
      }
      setBusy(false)
    }
  }

  async function switchLearner() {
    await signOutFamilySession()
    setSessionActive(false)
  }

  const locked = Boolean(selected?.lockedUntil && new Date(selected.lockedUntil) > new Date())
  const canSubmit =
    !busy &&
    !locked &&
    (selected?.signInMethod === 'picture' ? picture.length >= PICTURE_MIN : isValidPin(pin) && pin.length >= PIN_MIN)

  return (
    <main className="flex min-h-dvh items-start justify-center bg-surface-base px-4 py-10 sm:items-center">
      <div className="w-full max-w-xl space-y-6 rounded-3xl border border-border-default bg-surface-raised p-6 shadow-sm sm:p-8">
        {sessionActive ? (
          <div className="space-y-3 text-center">
            <h1 className="text-title-lg font-semibold text-fg-default">Someone is already signed in</h1>
            <p className="text-body-sm text-fg-muted">Keep learning, or sign out so someone else can sign in.</p>
            <div className="flex justify-center gap-2">
              <LinkButton to="/">Keep learning</LinkButton>
              <Button type="button" variant="secondary" onClick={() => void switchLearner()}>
                Sign out
              </Button>
            </div>
          </div>
        ) : loading ? (
          <div className="flex justify-center py-10">
            <Spinner label="Loading" />
          </div>
        ) : loadError ? (
          <div className="space-y-3">
            <InlineAlert tone="danger">{loadError}</InlineAlert>
            <Button type="button" variant="secondary" onClick={() => void load()}>
              Try again
            </Button>
          </div>
        ) : !picker ? (
          <div className="space-y-3 text-center">
            <h1 className="text-title-lg font-semibold text-fg-default">Who&apos;s learning?</h1>
            <p className="text-body-sm text-fg-muted">
              This device isn&apos;t set up for learner sign-in yet. A parent can turn it on from the Learners page.
            </p>
            <LinkButton to="/login?next=%2Flearners" variant="secondary">
              Parent sign in
            </LinkButton>
          </div>
        ) : !selected ? (
          <div className="space-y-5">
            <h1 className="text-center text-title-lg font-semibold text-fg-default">Who&apos;s learning?</h1>
            {picker.learners.length === 0 ? (
              <p className="text-center text-body-sm text-fg-muted">No learners yet. A parent can add them on the Learners page.</p>
            ) : (
              <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {picker.learners.map((l) => (
                  <li key={l.id}>
                    <Button
                      type="button"
                      variant="secondary"
                      onClick={() => choose(l)}
                      className="h-auto w-full flex-col gap-2 py-4"
                    >
                      <Avatar alt="" initials={initialsOf(l.displayName)} size="lg" />
                      <span className="max-w-full truncate text-body font-semibold">{l.displayName}</span>
                    </Button>
                  </li>
                ))}
              </ul>
            )}
            <p className="text-center text-caption text-fg-muted">
              Parent?{' '}
              <a href="/login" className="font-medium text-accent-fg hover:underline">
                Sign in to your account
              </a>
            </p>
          </div>
        ) : (
          <form onSubmit={(e) => void onSubmit(e)} className="space-y-5">
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => choose(null)}
                aria-label="Back to Who's learning"
                className="min-h-6 min-w-6"
              >
                <ArrowLeft className="h-4 w-4" aria-hidden />
              </Button>
              <Avatar alt="" initials={initialsOf(selected.displayName)} size="lg" />
              <h1 className="min-w-0 truncate text-title-lg font-semibold text-fg-default">Hi, {selected.displayName}!</h1>
            </div>
            {selected.signInMethod === 'none' ? (
              <InlineAlert tone="info">Ask a parent to set up your PIN or picture password on the Learners page.</InlineAlert>
            ) : locked ? (
              <InlineAlert tone="warning">{lockedMessage(selected.lockedUntil)}</InlineAlert>
            ) : (
              <>
                {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
                {selected.signInMethod === 'picture' ? (
                  <PictureSequenceInput
                    label="Your picture password"
                    value={picture}
                    onChange={setPicture}
                    choices={picker.pictureChoices.length ? picker.pictureChoices : undefined}
                    disabled={busy}
                    masked
                  />
                ) : (
                  <PinPad id="family-pin" label="Your PIN" value={pin} onChange={setPin} disabled={busy} keypad autoFocus />
                )}
                <Button type="submit" className="w-full" disabled={!canSubmit} loading={busy}>
                  Start learning
                </Button>
              </>
            )}
          </form>
        )}
      </div>
    </main>
  )
}
