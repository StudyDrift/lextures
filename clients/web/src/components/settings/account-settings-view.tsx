import {
  type ChangeEvent,
  type FormEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useState,
} from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ImageIcon, Monitor, Save, Trash2, Upload, X } from 'lucide-react'
import { useConfirm } from '../use-confirm'
import { OidcConnectedAccountsPanel } from '../oidc-connected-accounts-panel'
import { BotConnectedAccountsPanel } from '../bot-connected-accounts-panel'
import { MfaFactorsPanel } from './mfa-factors-panel'
import { StudyRemindersSettingsPanel } from './study-reminders-settings-panel'
import { BadgeProfileSettingsPanel } from './badge-profile-settings-panel'
import { AiProcessingSettingsPanel } from './ai-processing-settings-panel'
import { LocaleFormatSettingsPanel } from './locale-format-settings-panel'
import { LocaleSwitcher } from './locale-switcher'
import { SettingsSection } from './settings-section'
import { SegmentedControl } from './segmented-control'
import { apiUrl, authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { passwordStrengthEnglish, passwordStrengthKey, type PasswordStrengthKey } from '../../lib/password-strength'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { clearSessionTokens, getRefreshToken } from '../../lib/session-tokens'
import { applyUiTheme, parseUiTheme, type UiTheme } from '../../lib/ui-theme'
import { useUiDensityControls } from '../../context/ui-density-context'
import { useLocaleFormatContext } from '../../context/locale-format-context'
import { detectBrowserLocale, detectBrowserTimeZone, formatDateTime } from '../../lib/format'
import { syncUserLocale } from '../../lib/sync-user-locale'
import { nameFieldsFromProfile } from '../layout/top-bar-utils'

type AccountProfile = {
  email: string
  displayName?: string | null
  firstName?: string | null
  lastName?: string | null
  phoneNumber?: string | null
  avatarUrl?: string | null
  uiTheme?: string | null
  showHelpPopover?: boolean
  locale?: string | null
  rtlEnabled?: boolean
  sid?: string | null
  sessionManagementUiEnabled?: boolean
  timezone?: string | null
}

type ActiveSessionRow = {
  id: string
  createdAt: string
  lastUsedAt: string
  deviceLabel: string
  location: string
  authMethod: string
  isCurrent: boolean
}

function defaultAvatarPrompt(firstName: string, lastName: string): string {
  const name = [firstName.trim(), lastName.trim()].filter(Boolean).join(' ').trim()
  return name
    ? `Create a friendly profile avatar illustration for ${name}. Clean background, centered portrait framing, modern style.`
    : 'Create a friendly profile avatar illustration with clean background, centered portrait framing, and modern style.'
}

function SettingsField({
  label,
  htmlFor,
  hint,
  children,
}: {
  label: string
  htmlFor?: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div>
      <label htmlFor={htmlFor} className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-200">
        {label}
      </label>
      {children}
      {hint ? <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">{hint}</p> : null}
    </div>
  )
}

const inputClass =
  'w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100'
const disabledInputClass =
  'w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm text-slate-500 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-400'

export function AccountSettingsView() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { confirm, ConfirmDialogHost } = useConfirm()
  const accountFormId = useId()
  const { density, setDensity } = useUiDensityControls()
  const { setProfile: setLocaleProfile } = useLocaleFormatContext()
  const [displayLocale, setDisplayLocale] = useState(detectBrowserLocale())
  const [displayTimezone, setDisplayTimezone] = useState(detectBrowserTimeZone())

  const [accountLoading, setAccountLoading] = useState(false)
  const [accountSaving, setAccountSaving] = useState(false)
  const [deleteAccountBusy, setDeleteAccountBusy] = useState(false)
  const [accountMessage, setAccountMessage] = useState<string | null>(null)
  const [accountError, setAccountError] = useState<string | null>(null)
  const [email, setEmail] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [phoneNumber, setPhoneNumber] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [avatarPreviewUrl, setAvatarPreviewUrl] = useState<string | null>(null)
  const [avatarModalOpen, setAvatarModalOpen] = useState(false)
  const [avatarPrompt, setAvatarPrompt] = useState('')
  const [avatarGenStatus, setAvatarGenStatus] = useState<'idle' | 'loading' | 'error'>('idle')
  const [avatarGenMessage, setAvatarGenMessage] = useState<string | null>(null)
  const [uiTheme, setUiTheme] = useState<UiTheme>('light')
  const [showHelpPopover, setShowHelpPopover] = useState(true)
  const [localeTag, setLocaleTag] = useState('en')
  const [studentId, setStudentId] = useState<string | null>(null)
  const [sessionManagementUiEnabled, setSessionManagementUiEnabled] = useState(false)
  const [sessions, setSessions] = useState<ActiveSessionRow[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [sessionsError, setSessionsError] = useState<string | null>(null)

  const [pwPolicy, setPwPolicy] = useState<{
    minLength: number
    requireUpper: boolean
    requireLower: boolean
    requireDigit: boolean
    requireSpecial: boolean
    checkHibp: boolean
  } | null>(null)
  const [cpCurrent, setCpCurrent] = useState('')
  const [cpNew, setCpNew] = useState('')
  const [cpConfirm, setCpConfirm] = useState('')
  const [cpBusy, setCpBusy] = useState(false)
  const [cpErr, setCpErr] = useState<string | null>(null)
  const [cpOk, setCpOk] = useState<string | null>(null)

  const pwMinLen = pwPolicy?.minLength ?? 8
  const cpStrengthKey: PasswordStrengthKey = passwordStrengthKey(cpNew)
  const cpStrengthLabel = useMemo(() => passwordStrengthEnglish(cpStrengthKey), [cpStrengthKey])

  const loadAccount = useCallback(async () => {
    setAccountLoading(true)
    setAccountError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/account')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setAccountError(readApiErrorMessage(raw))
        return
      }
      const data = raw as AccountProfile
      setEmail(data.email ?? '')
      const names = nameFieldsFromProfile(data)
      setFirstName(names.firstName)
      setLastName(names.lastName)
      setPhoneNumber(data.phoneNumber ?? '')
      const currentAvatar = data.avatarUrl ?? ''
      setAvatarUrl(currentAvatar)
      setAvatarPreviewUrl(currentAvatar || null)
      setUiTheme(parseUiTheme(data.uiTheme))
      setStudentId(data.sid?.trim() ? data.sid.trim() : null)
      setSessionManagementUiEnabled(data.sessionManagementUiEnabled === true)
      if (data.showHelpPopover !== undefined) {
        setShowHelpPopover(data.showHelpPopover)
      }
      if (data.rtlEnabled !== undefined) {
        try {
          window.localStorage.setItem('lextures.rtlEnabled', data.rtlEnabled ? '1' : '0')
        } catch {
          /* ignore */
        }
      }
      if (data.locale?.trim()) {
        const loc = data.locale.trim()
        setLocaleTag(loc)
        setDisplayLocale(loc)
        void syncUserLocale(loc)
      } else {
        setDisplayLocale(detectBrowserLocale())
      }
      const tz = data.timezone?.trim() || detectBrowserTimeZone()
      setDisplayTimezone(tz)
      setLocaleProfile({
        locale: data.locale?.trim() ?? null,
        timezone: data.timezone ?? null,
      })
    } catch {
      setAccountError('Could not load account settings.')
    } finally {
      setAccountLoading(false)
    }
  }, [setLocaleProfile])

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true)
    setSessionsError(null)
    try {
      const res = await authorizedFetch('/api/v1/me/sessions')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setSessionsError(readApiErrorMessage(raw))
        return
      }
      const data = raw as { sessions?: ActiveSessionRow[] }
      setSessions(data.sessions ?? [])
    } catch {
      setSessionsError('Could not load active sessions.')
    } finally {
      setSessionsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadAccount()
  }, [loadAccount])

  useEffect(() => {
    if (!sessionManagementUiEnabled) {
      setSessions([])
      return
    }
    void loadSessions()
  }, [sessionManagementUiEnabled, loadSessions])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await fetch(apiUrl('/api/v1/auth/password-policy'))
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok || cancelled) return
        const p = raw as {
          minLength?: number
          requireUpper?: boolean
          requireLower?: boolean
          requireDigit?: boolean
          requireSpecial?: boolean
          checkHibp?: boolean
        }
        setPwPolicy({
          minLength: typeof p.minLength === 'number' ? p.minLength : 8,
          requireUpper: !!p.requireUpper,
          requireLower: !!p.requireLower,
          requireDigit: !!p.requireDigit,
          requireSpecial: !!p.requireSpecial,
          checkHibp: p.checkHibp !== false,
        })
      } catch {
        /* ignore */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!avatarModalOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (avatarGenStatus === 'loading') return
      e.preventDefault()
      setAvatarModalOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [avatarModalOpen, avatarGenStatus])

  async function onAvatarUpload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    if (!file.type.startsWith('image/')) {
      setAccountError('Choose an image file.')
      return
    }
    if (file.size > 3 * 1024 * 1024) {
      setAccountError('Image file must be 3MB or smaller.')
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      const result = typeof reader.result === 'string' ? reader.result : ''
      setAvatarUrl(result)
      setAvatarPreviewUrl(result || null)
      setAccountError(null)
      setAccountMessage('Image selected. Save to apply it.')
    }
    reader.onerror = () => {
      setAccountError('Could not read that image file.')
    }
    reader.readAsDataURL(file)
    e.target.value = ''
  }

  function openGenerateAvatarModal() {
    setAvatarPrompt(defaultAvatarPrompt(firstName, lastName))
    setAvatarGenStatus('idle')
    setAvatarGenMessage(null)
    setAvatarModalOpen(true)
  }

  async function onGenerateAvatar(e: FormEvent) {
    e.preventDefault()
    if (!avatarPrompt.trim()) return
    setAvatarGenStatus('loading')
    setAvatarGenMessage(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/account/generate-avatar', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt: avatarPrompt.trim() }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setAvatarGenStatus('error')
        setAvatarGenMessage(readApiErrorMessage(raw))
        return
      }
      const data = raw as { imageUrl?: string }
      if (data.imageUrl) {
        setAvatarUrl(data.imageUrl)
        setAvatarPreviewUrl(data.imageUrl)
      }
      setAvatarGenStatus('idle')
      setAvatarGenMessage('Avatar generated. Save account to apply it.')
    } catch {
      setAvatarGenStatus('error')
      setAvatarGenMessage('Could not reach the server.')
    }
  }

  async function persistUiTheme(next: UiTheme) {
    const prev = uiTheme
    setUiTheme(next)
    applyUiTheme(next)
    setAccountError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/account', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          firstName,
          lastName,
          avatarUrl: avatarUrl.trim() || null,
          uiTheme: next,
        }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setUiTheme(prev)
        applyUiTheme(prev)
        setAccountError(readApiErrorMessage(raw))
        return
      }
      window.dispatchEvent(new Event('studydrift-profile-updated'))
    } catch {
      setUiTheme(prev)
      applyUiTheme(prev)
      setAccountError('Could not save appearance.')
    }
  }

  async function persistDisplayTimezone(next: string) {
    const prev = displayTimezone
    setDisplayTimezone(next)
    setAccountError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/account', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ timezone: next }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setDisplayTimezone(prev)
        setAccountError(readApiErrorMessage(raw))
        return
      }
      const data = raw as AccountProfile
      setLocaleProfile({
        locale: data.locale?.trim() ?? displayLocale,
        timezone: data.timezone ?? null,
      })
      window.dispatchEvent(new Event('studydrift-profile-updated'))
    } catch {
      setDisplayTimezone(prev)
      setAccountError('Could not save time zone.')
    }
  }

  async function persistShowHelpPopover(next: boolean) {
    const prev = showHelpPopover
    setShowHelpPopover(next)
    setAccountError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/account', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          firstName,
          lastName,
          avatarUrl: avatarUrl.trim() || null,
          uiTheme,
          showHelpPopover: next,
        }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setShowHelpPopover(prev)
        setAccountError(readApiErrorMessage(raw))
        return
      }
      window.dispatchEvent(new Event('studydrift-profile-updated'))
    } catch {
      setShowHelpPopover(prev)
      setAccountError('Could not save appearance.')
    }
  }

  async function onChangePassword(e: FormEvent) {
    e.preventDefault()
    setCpErr(null)
    setCpOk(null)
    if (cpNew !== cpConfirm) {
      setCpErr('New passwords do not match.')
      return
    }
    if (cpNew.length < pwMinLen) {
      setCpErr(`New password must be at least ${pwMinLen} characters.`)
      return
    }
    setCpBusy(true)
    try {
      const res = await authorizedFetch('/api/v1/auth/change-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: cpCurrent, new_password: cpNew }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setCpErr(readApiErrorMessage(raw))
        return
      }
      setCpCurrent('')
      setCpNew('')
      setCpConfirm('')
      setCpOk('Password updated.')
      toastSaveOk('Password updated')
    } catch {
      setCpErr('Could not update password.')
      toastMutationError('Could not update password.')
    } finally {
      setCpBusy(false)
    }
  }

  async function revokeSession(id: string) {
    if (!(await confirm({ title: t('account.signOutSession.title') }))) {
      return
    }
    setSessionsError(null)
    try {
      const res = await authorizedFetch(`/api/v1/me/sessions/${encodeURIComponent(id)}`, { method: 'DELETE' })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setSessionsError(readApiErrorMessage(raw))
        toastMutationError(readApiErrorMessage(raw))
        return
      }
      toastSaveOk('Session signed out')
      await loadSessions()
    } catch {
      setSessionsError('Could not revoke session.')
      toastMutationError('Could not revoke session.')
    }
  }

  async function onDeleteAccount() {
    const ok = await confirm({
      title: t('account.delete.title'),
      description: t('account.delete.description'),
      confirmLabel: t('account.delete.confirm'),
      variant: 'danger',
      requireTypedPhrase: t('account.delete.phrase'),
    })
    if (!ok) return
    setDeleteAccountBusy(true)
    try {
      const res = await authorizedFetch('/api/v1/settings/account', { method: 'DELETE' })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        toastMutationError(readApiErrorMessage(raw) || t('account.delete.error'))
        return
      }
      const rt = getRefreshToken()
      if (rt) {
        try {
          await fetch(apiUrl('/api/v1/auth/logout'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: rt }),
          })
        } catch {
          /* ignore — local session is cleared below */
        }
      }
      clearSessionTokens()
      applyUiTheme('light')
      navigate('/login', { replace: true })
    } catch {
      toastMutationError(t('account.delete.error'))
    } finally {
      setDeleteAccountBusy(false)
    }
  }

  async function revokeAllOtherSessions() {
    if (!(await confirm({ title: t('account.signOutAll.title') }))) {
      return
    }
    setSessionsError(null)
    try {
      const res = await authorizedFetch('/api/v1/me/sessions', { method: 'DELETE' })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setSessionsError(readApiErrorMessage(raw))
        toastMutationError(readApiErrorMessage(raw))
        return
      }
      toastSaveOk('Other sessions signed out')
      await loadSessions()
    } catch {
      setSessionsError('Could not sign out other sessions.')
      toastMutationError('Could not sign out other sessions.')
    }
  }

  async function onSaveAccount(e: FormEvent) {
    e.preventDefault()
    setAccountSaving(true)
    setAccountMessage(null)
    setAccountError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/account', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          firstName,
          lastName,
          phoneNumber: phoneNumber.trim() || null,
          avatarUrl: avatarUrl.trim() || null,
          uiTheme,
          showHelpPopover,
        }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setAccountError(readApiErrorMessage(raw))
        return
      }
      const data = raw as AccountProfile
      setFirstName(data.firstName ?? '')
      setLastName(data.lastName ?? '')
      setPhoneNumber(data.phoneNumber ?? '')
      setStudentId(data.sid?.trim() ? data.sid.trim() : null)
      setSessionManagementUiEnabled(data.sessionManagementUiEnabled === true)
      const nextAvatar = data.avatarUrl ?? ''
      setAvatarUrl(nextAvatar)
      setAvatarPreviewUrl(nextAvatar || null)
      if (data.showHelpPopover !== undefined) {
        setShowHelpPopover(data.showHelpPopover)
      }
      const loc = data.locale?.trim() || detectBrowserLocale()
      const tz = data.timezone?.trim() || detectBrowserTimeZone()
      setDisplayLocale(loc)
      setDisplayTimezone(tz)
      setLocaleProfile({ locale: data.locale ?? null, timezone: data.timezone ?? null })
      setAccountMessage('Saved.')
      toastSaveOk('Account saved')
      window.dispatchEvent(new Event('studydrift-profile-updated'))
    } catch {
      setAccountError('Could not save account settings.')
      toastMutationError('Could not save account settings.')
    } finally {
      setAccountSaving(false)
    }
  }

  if (accountLoading) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
  }

  return (
    <div className="space-y-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Account</h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
          Manage your profile, security, and personal preferences.
        </p>
      </header>

      {accountError ? (
        <p className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
          {accountError}
        </p>
      ) : null}

      <SettingsSection
        id="profile"
        title="Profile"
        description="Your name and photo appear in the app header and across courses."
      >
        <form id={accountFormId} className="space-y-5" onSubmit={onSaveAccount}>
          <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
            <div className="flex shrink-0 flex-col items-center gap-3">
              <div className="flex h-24 w-24 items-center justify-center overflow-hidden rounded-full border border-slate-200 bg-slate-100 dark:border-neutral-600 dark:bg-neutral-800">
                {avatarPreviewUrl ? (
                  <img src={avatarPreviewUrl} alt="" className="h-full w-full object-cover" />
                ) : (
                  <ImageIcon className="h-7 w-7 text-slate-400" aria-hidden />
                )}
              </div>
              <div className="flex flex-wrap justify-center gap-2">
                <button
                  type="button"
                  onClick={openGenerateAvatarModal}
                  className="inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-medium text-slate-700 hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-900 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:border-indigo-400 dark:hover:bg-neutral-700"
                >
                  <ImageIcon className="h-3.5 w-3.5" aria-hidden />
                  Generate
                </button>
                <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-medium text-slate-700 hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-900 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:border-indigo-400 dark:hover:bg-neutral-700">
                  <Upload className="h-3.5 w-3.5" aria-hidden />
                  Upload
                  <input type="file" accept="image/*" className="hidden" onChange={onAvatarUpload} />
                </label>
              </div>
            </div>

            <div className="min-w-0 flex-1 space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <SettingsField label="First name">
                  <input
                    form={accountFormId}
                    type="text"
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    maxLength={80}
                    className={inputClass}
                  />
                </SettingsField>
                <SettingsField label="Last name">
                  <input
                    form={accountFormId}
                    type="text"
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    maxLength={80}
                    className={inputClass}
                  />
                </SettingsField>
              </div>

              <SettingsField label="Email" htmlFor="account-email">
                <input id="account-email" type="text" value={email} disabled className={disabledInputClass} />
              </SettingsField>

              <SettingsField
                label="Phone number"
                htmlFor="account-phone"
                hint="Used for SMS notifications when enabled on the Notifications page."
              >
                <input
                  id="account-phone"
                  form={accountFormId}
                  type="tel"
                  autoComplete="tel"
                  value={phoneNumber}
                  onChange={(e) => setPhoneNumber(e.target.value)}
                  maxLength={30}
                  placeholder="+1 (555) 555-0100"
                  className={inputClass}
                />
              </SettingsField>

              {studentId ? (
                <SettingsField
                  label="Student ID"
                  htmlFor="account-sid"
                  hint="Assigned by your institution. Contact an administrator if this should be updated."
                >
                  <input id="account-sid" type="text" value={studentId} disabled className={disabledInputClass} />
                </SettingsField>
              ) : null}

              <details className="group">
                <summary className="cursor-pointer text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-200">
                  Advanced: image URL
                </summary>
                <div className="mt-3">
                  <input
                    form={accountFormId}
                    type="url"
                    value={avatarUrl}
                    onChange={(e) => {
                      setAvatarUrl(e.target.value)
                      setAvatarPreviewUrl(e.target.value.trim() || null)
                    }}
                    placeholder="https://example.com/avatar.png"
                    className={inputClass}
                  />
                </div>
              </details>
            </div>
          </div>

          {accountMessage ? (
            <p className="text-sm text-emerald-700 dark:text-emerald-400" role="status">
              {accountMessage}
            </p>
          ) : null}

          <button
            type="submit"
            disabled={accountSaving}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:shadow-none"
          >
            <Save className="h-4 w-4" aria-hidden />
            {accountSaving ? 'Saving…' : 'Save profile'}
          </button>
        </form>
      </SettingsSection>

      <SettingsSection
        id="security"
        title="Security"
        description="Password, two-factor authentication, and active sign-in sessions."
      >
        <div className="space-y-8">
          <div>
            <h4 className="text-sm font-medium text-slate-800 dark:text-neutral-200">Password</h4>
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              Use a unique password you do not reuse on other sites.
            </p>
            <form className="mt-4 space-y-4" onSubmit={onChangePassword}>
              <ul
                id="account-password-requirements"
                className="list-inside list-disc text-xs text-slate-600 dark:text-neutral-400"
              >
                <li>At least {pwMinLen} characters</li>
                {pwPolicy?.requireUpper ? <li>One uppercase letter</li> : null}
                {pwPolicy?.requireLower ? <li>One lowercase letter</li> : null}
                {pwPolicy?.requireDigit ? <li>One digit</li> : null}
                {pwPolicy?.requireSpecial ? <li>One symbol or punctuation character</li> : null}
                {pwPolicy == null || pwPolicy.checkHibp ? (
                  <li>Must not appear in known public breach lists (checked securely)</li>
                ) : null}
              </ul>
              <label className="block">
                <span className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-200">
                  Current password
                </span>
                <input
                  type="password"
                  autoComplete="current-password"
                  value={cpCurrent}
                  onChange={(e) => setCpCurrent(e.target.value)}
                  aria-invalid={cpErr != null}
                  aria-describedby="account-password-requirements account-password-strength"
                  className={inputClass}
                />
              </label>
              <label className="block">
                <span className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-200">
                  New password
                </span>
                <input
                  type="password"
                  autoComplete="new-password"
                  value={cpNew}
                  minLength={pwMinLen}
                  onChange={(e) => setCpNew(e.target.value)}
                  aria-describedby="account-password-requirements account-password-strength"
                  className={inputClass}
                />
              </label>
              <div id="account-password-strength" className="flex items-center gap-2" aria-live="polite">
                <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">Strength:</span>
                <span className="text-xs font-semibold text-slate-800 dark:text-neutral-100">{cpStrengthLabel}</span>
                <div className="h-1.5 flex-1 rounded-full bg-slate-200 dark:bg-neutral-700" aria-hidden>
                  <div
                    className={`h-full rounded-full ${
                      cpStrengthKey === 'password.strength.weak'
                        ? 'w-1/3 bg-rose-500'
                        : cpStrengthKey === 'password.strength.fair'
                          ? 'w-2/3 bg-amber-500'
                          : 'w-full bg-emerald-600'
                    }`}
                  />
                </div>
              </div>
              <label className="block">
                <span className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-200">
                  Confirm new password
                </span>
                <input
                  type="password"
                  autoComplete="new-password"
                  value={cpConfirm}
                  minLength={pwMinLen}
                  onChange={(e) => setCpConfirm(e.target.value)}
                  className={inputClass}
                />
              </label>
              {cpErr ? (
                <p className="text-sm text-rose-600 dark:text-rose-400" role="status">
                  {cpErr}
                </p>
              ) : null}
              {cpOk ? (
                <p className="text-sm text-emerald-700 dark:text-emerald-400" role="status">
                  {cpOk}
                </p>
              ) : null}
              <button
                type="submit"
                disabled={cpBusy}
                className="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-indigo-200 hover:bg-indigo-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-indigo-400 dark:hover:bg-neutral-800"
              >
                {cpBusy ? 'Updating…' : 'Update password'}
              </button>
            </form>
          </div>

          <MfaFactorsPanel embedded />

          {sessionManagementUiEnabled ? (
            <div>
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <h4 className="flex items-center gap-2 text-sm font-medium text-slate-800 dark:text-neutral-200">
                    <Monitor className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
                    Active sessions
                  </h4>
                  <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                    Where you are signed in. Location is approximate when shown.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => void revokeAllOtherSessions()}
                  disabled={sessionsLoading || sessions.filter((s) => !s.isCurrent).length === 0}
                  className="shrink-0 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-rose-200 hover:bg-rose-50 hover:text-rose-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-rose-500/50 dark:hover:bg-rose-950/40"
                >
                  Sign out everywhere else
                </button>
              </div>
              {sessionsError ? (
                <p className="mt-3 text-sm text-rose-600 dark:text-rose-400" role="alert">
                  {sessionsError}
                </p>
              ) : null}
              {sessionsLoading ? <p className="mt-4 text-sm text-slate-500">Loading sessions…</p> : null}
              {!sessionsLoading && sessions.length === 0 ? (
                <p className="mt-4 text-sm text-slate-500">No active sessions found.</p>
              ) : null}
              {!sessionsLoading && sessions.length > 0 ? (
                <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-600">
                  <table
                    className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-600"
                    aria-label="Active sessions"
                  >
                    <thead className="bg-slate-50 dark:bg-neutral-800/80">
                      <tr>
                        <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                          Device
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                          Location
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                          Signed in
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                          Last active
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                          Method
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                          Action
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 bg-white dark:divide-neutral-600 dark:bg-neutral-900">
                      {sessions.map((s) => (
                        <tr
                          key={s.id}
                          className={
                            s.isCurrent
                              ? 'bg-indigo-50/60 dark:bg-indigo-950/25'
                              : 'hover:bg-slate-50 dark:hover:bg-neutral-800/60'
                          }
                        >
                          <th
                            scope="row"
                            className="whitespace-nowrap px-3 py-2.5 font-normal text-slate-900 dark:text-neutral-100"
                          >
                            <span className="flex flex-wrap items-center gap-2">
                              {s.deviceLabel}
                              {s.isCurrent ? (
                                <span className="rounded-md bg-indigo-100 px-1.5 py-0.5 text-xs font-semibold text-indigo-900 dark:bg-indigo-900/60 dark:text-indigo-100">
                                  This device
                                </span>
                              ) : null}
                            </span>
                          </th>
                          <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">
                            {s.location}
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">
                            {formatDateTime(s.createdAt)}
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">
                            {formatDateTime(s.lastUsedAt)}
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">
                            {s.authMethod}
                          </td>
                          <td className="px-3 py-2.5">
                            {s.isCurrent ? (
                              <span className="text-xs text-slate-400 dark:text-neutral-500">—</span>
                            ) : (
                              <button
                                type="button"
                                onClick={() => void revokeSession(s.id)}
                                className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-800 hover:border-rose-200 hover:bg-rose-50 hover:text-rose-900 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-rose-500/50"
                                aria-label={`Sign out session on ${s.deviceLabel}`}
                              >
                                Sign out
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </SettingsSection>

      <SettingsSection
        id="connected-accounts"
        title="Connected accounts"
        description="Link sign-in providers and messaging apps to this account."
      >
        <div className="space-y-6">
          <OidcConnectedAccountsPanel embedded />
          <BotConnectedAccountsPanel embedded />
        </div>
      </SettingsSection>

      <SettingsSection
        id="preferences"
        title="Preferences"
        description="Appearance, language, and regional settings for your account."
      >
        <div className="space-y-6">
          <div>
            <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">Theme</p>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Saved to your account and applied when you sign in.
            </p>
            <div className="mt-3">
              <SegmentedControl
                aria-label="Theme"
                value={uiTheme}
                options={[
                  { value: 'light', label: 'Light' },
                  { value: 'dark', label: 'Dark' },
                ]}
                onChange={(v) => void persistUiTheme(v)}
              />
            </div>
          </div>

          <div>
            <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">Layout density</p>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Compact tightens tables and navigation. Stored on this device only.
            </p>
            <div className="mt-3">
              <SegmentedControl
                aria-label="Layout density"
                value={density}
                options={[
                  { value: 'comfortable', label: 'Comfortable' },
                  { value: 'compact', label: 'Compact' },
                ]}
                onChange={setDensity}
              />
            </div>
          </div>

          <LocaleSwitcher
            initialLocale={localeTag}
            onLocaleChange={(tag) => {
              setLocaleTag(tag)
              setDisplayLocale(tag)
              setLocaleProfile({ locale: tag, timezone: displayTimezone })
            }}
            embedded
          />

          <LocaleFormatSettingsPanel
            timezone={displayTimezone}
            onTimezoneChange={(v) => void persistDisplayTimezone(v)}
            disabled={accountSaving}
            embedded
          />

          <div>
            <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">Help button</p>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Show or hide the help button in the top menu bar.
            </p>
            <div className="mt-3">
              <SegmentedControl
                aria-label="Help button visibility"
                value={showHelpPopover ? 'show' : 'hide'}
                options={[
                  { value: 'show', label: 'Show' },
                  { value: 'hide', label: 'Hide' },
                ]}
                onChange={(v) => void persistShowHelpPopover(v === 'show')}
              />
            </div>
          </div>

          <AiProcessingSettingsPanel embedded />
        </div>
      </SettingsSection>

      <StudyRemindersSettingsPanel embedded />
      <BadgeProfileSettingsPanel />

      <SettingsSection
        id="danger-zone"
        title={t('account.delete.sectionTitle')}
        description={t('account.delete.sectionDescription')}
      >
        <div className="rounded-xl border border-red-200 bg-red-50/60 p-4 dark:border-red-900/50 dark:bg-red-950/30">
          <p className="text-sm text-slate-700 dark:text-neutral-200">{t('account.delete.warning')}</p>
          <button
            type="button"
            disabled={deleteAccountBusy || accountLoading}
            onClick={() => void onDeleteAccount()}
            className="mt-4 inline-flex items-center gap-2 rounded-xl border border-red-300 bg-white px-4 py-2.5 text-sm font-semibold text-red-700 shadow-sm transition-[background-color,color,border-color] hover:border-red-400 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-red-800 dark:bg-neutral-900 dark:text-red-300 dark:hover:bg-red-950/50"
          >
            <Trash2 className="h-4 w-4" aria-hidden />
            {deleteAccountBusy ? t('account.delete.deleting') : t('account.delete.button')}
          </button>
        </div>
      </SettingsSection>

      {avatarModalOpen ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
          role="dialog"
          aria-modal="true"
          aria-labelledby="generate-avatar-title"
          onClick={(e) => {
            if (e.target === e.currentTarget) setAvatarModalOpen(false)
          }}
        >
          <div className="w-full max-w-2xl overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
            <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-600">
              <h3 id="generate-avatar-title" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                Generate avatar
              </h3>
              <button
                type="button"
                onClick={() => setAvatarModalOpen(false)}
                className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={onGenerateAvatar} className="grid gap-4 p-4 md:grid-cols-[1fr,240px]">
              <div>
                <label htmlFor="avatar-prompt" className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                  Prompt
                </label>
                <textarea
                  id="avatar-prompt"
                  rows={6}
                  value={avatarPrompt}
                  onChange={(e) => setAvatarPrompt(e.target.value)}
                  className={`mt-1 ${inputClass}`}
                />
                {avatarGenMessage ? (
                  <p
                    className={
                      avatarGenStatus === 'error'
                        ? 'mt-2 text-sm text-rose-700 dark:text-rose-400'
                        : 'mt-2 text-sm text-emerald-700 dark:text-emerald-400'
                    }
                    role="status"
                  >
                    {avatarGenMessage}
                  </p>
                ) : null}
                <div className="mt-3 flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setAvatarModalOpen(false)}
                    className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
                  >
                    Close
                  </button>
                  <button
                    type="submit"
                    disabled={avatarGenStatus === 'loading' || !avatarPrompt.trim()}
                    className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:shadow-none"
                  >
                    {avatarGenStatus === 'loading' ? 'Generating…' : 'Generate'}
                  </button>
                </div>
              </div>
              <div>
                <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">Preview</span>
                <div className="mt-1 flex h-60 items-center justify-center overflow-hidden rounded-xl border border-dashed border-slate-200 bg-slate-50 dark:border-neutral-600 dark:bg-neutral-800/50">
                  {avatarGenStatus === 'loading' ? (
                    <span className="text-sm text-slate-500">Generating…</span>
                  ) : avatarPreviewUrl ? (
                    <img src={avatarPreviewUrl} alt="" className="h-full w-full object-contain" />
                  ) : (
                    <span className="text-sm text-slate-400">Generated image will appear here</span>
                  )}
                </div>
              </div>
            </form>
          </div>
        </div>
      ) : null}
      {ConfirmDialogHost}
    </div>
  )
}