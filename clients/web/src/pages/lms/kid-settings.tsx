import { useEffect, useState, type ChangeEvent } from 'react'
import { BookOpen } from 'lucide-react'
import { Avatar, Button, Field, FileInput, Input, SegmentedControl, Switch } from '../../components/ui'
import { ReadingPreferencesPanel } from '../../components/a11y/ReadingPreferencesPanel'
import { LmsPage } from './lms-page'
import {
  fetchLearnerSettings,
  updateLearnerSettings,
  type LearnerSettings,
} from '../../lib/learner-settings-api'
import { applyHighContrast, applyUiTheme, readStoredHighContrast, type UiTheme } from '../../lib/ui-theme'
import {
  notificationSoundsEnabled,
  playNotificationChime,
  setNotificationSoundsEnabled,
} from '../../lib/notification-sounds'

/** Data-URL avatars must stay under the server's 2,000,000-character limit. */
const MAX_AVATAR_BYTES = 1_400_000

const COPY = {
  page: { title: 'Settings', description: 'Make Lextures look and sound the way you like.' },
  profile: { title: 'Your name and picture', description: 'This is how your family and teachers see you.' },
  appearance: { title: 'Look and reading', description: 'Colors, contrast, and reading help.' },
  sounds: { title: 'Sounds', description: 'Hear a short chime when a new message arrives.' },
} as const

function initialsOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  return name.trim().slice(0, 2).toUpperCase()
}

function SettingsCard({ id, title, description, children }: { id: string; title: string; description: string; children: React.ReactNode }) {
  return (
    <section aria-labelledby={id} className="space-y-4 rounded-xl border border-border-default bg-surface-raised p-5">
      <div>
        <h2 id={id} className="text-base font-semibold text-fg-default">
          {title}
        </h2>
        <p className="mt-1 text-sm text-fg-muted">{description}</p>
      </div>
      {children}
    </section>
  )
}

/**
 * Simplified Settings for a managed learner (#775): display name, avatar, theme and
 * accessibility, and notification sounds. No email, password, privacy, or AI controls, and
 * nothing here calls a parent-only API, so loading it never ends the Learn-as session.
 */
export default function KidSettingsPage() {
  const [settings, setSettings] = useState<LearnerSettings | null>(null)
  const [nameDraft, setNameDraft] = useState('')
  const [loadError, setLoadError] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [highContrast, setHighContrast] = useState(() => readStoredHighContrast())
  const [sounds, setSounds] = useState(() => notificationSoundsEnabled())
  const [readingOpen, setReadingOpen] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const s = await fetchLearnerSettings()
        if (cancelled) return
        setSettings(s)
        setNameDraft(s.displayName)
      } catch (e) {
        if (!cancelled) setLoadError(e instanceof Error ? e.message : 'Could not load your settings.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  async function save(patch: Parameters<typeof updateLearnerSettings>[0], okMessage: string) {
    setBusy(true)
    setError(null)
    setStatus(null)
    try {
      const next = await updateLearnerSettings(patch)
      setSettings(next)
      setNameDraft(next.displayName)
      setStatus(okMessage)
      return true
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save. Try again.')
      return false
    } finally {
      setBusy(false)
    }
  }

  function onAvatarFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (!file.type.startsWith('image/')) {
      setError('Choose a picture file.')
      return
    }
    if (file.size > MAX_AVATAR_BYTES) {
      setError('That picture is too big. Choose one smaller than 1 MB.')
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      const url = typeof reader.result === 'string' ? reader.result : ''
      if (url) void save({ avatarUrl: url }, 'Picture updated.')
    }
    reader.onerror = () => setError('Could not read that picture.')
    reader.readAsDataURL(file)
  }

  async function onTheme(next: UiTheme) {
    const prev = settings?.uiTheme ?? 'light'
    applyUiTheme(next)
    const ok = await save({ uiTheme: next }, 'Theme saved.')
    if (!ok) applyUiTheme(prev)
  }

  function onHighContrast(next: boolean) {
    setHighContrast(next)
    applyHighContrast(next)
  }

  function onSounds(next: boolean) {
    setSounds(next)
    setNotificationSoundsEnabled(next)
    if (next) playNotificationChime()
  }

  const displayName = settings?.displayName ?? ''
  const trimmedDraft = nameDraft.trim()

  return (
    <LmsPage title={COPY.page.title} description={COPY.page.description}>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-5">
        {loadError && (
          <p role="alert" className="rounded-lg border border-danger-border bg-danger-surface px-4 py-3 text-sm text-danger-fg">
            {loadError}
          </p>
        )}
        {error && (
          <p role="alert" className="text-sm text-danger-fg">
            {error}
          </p>
        )}
        {status && (
          <p role="status" className="text-sm text-fg-muted">
            {status}
          </p>
        )}

        <SettingsCard id="kid-settings-profile" title={COPY.profile.title} description={COPY.profile.description}>
          <form
            className="flex flex-wrap items-end gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              if (trimmedDraft && trimmedDraft !== displayName) void save({ displayName: trimmedDraft }, 'Name saved.')
            }}
          >
            <Field label="Display name" className="min-w-[14rem] flex-1">
              <Input value={nameDraft} maxLength={80} onChange={(e) => setNameDraft(e.target.value)} disabled={!settings} />
            </Field>
            <Button type="submit" variant="secondary" disabled={busy || !settings || !trimmedDraft || trimmedDraft === displayName}>
              Save name
            </Button>
          </form>
          <div className="flex flex-wrap items-center gap-4">
            <Avatar size="lg" src={settings?.avatarUrl ?? undefined} alt="" initials={initialsOf(displayName || '?')} />
            <FileInput accept="image/*" buttonLabel="Choose a picture" label="Picture" onChange={onAvatarFile} disabled={busy || !settings} />
            {settings?.avatarUrl ? (
              <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => void save({ avatarUrl: '' }, 'Picture removed.')}>
                Remove picture
              </Button>
            ) : null}
          </div>
        </SettingsCard>

        <SettingsCard id="kid-settings-appearance" title={COPY.appearance.title} description={COPY.appearance.description}>
          <SegmentedControl
            label="Theme"
            value={settings?.uiTheme ?? 'light'}
            options={[
              { value: 'light', label: 'Light' },
              { value: 'dark', label: 'Dark' },
            ]}
            onChange={(v) => void onTheme(v)}
          />
          <Switch
            checked={highContrast}
            onCheckedChange={onHighContrast}
            label="High contrast"
            description="Stronger colors that are easier to see. Saved on this device."
          />
          <div>
            <Button type="button" variant="secondary" onClick={() => setReadingOpen(true)}>
              <BookOpen size={16} aria-hidden />
              Reading and text options
            </Button>
          </div>
        </SettingsCard>

        <SettingsCard id="kid-settings-sounds" title={COPY.sounds.title} description={COPY.sounds.description}>
          <Switch
            checked={sounds}
            onCheckedChange={onSounds}
            label="Notification sounds"
            description="Saved on this device."
          />
        </SettingsCard>
      </div>
      <ReadingPreferencesPanel open={readingOpen} onClose={() => setReadingOpen(false)} />
    </LmsPage>
  )
}
