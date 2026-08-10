import {
  type ChangeEvent,
  type ComponentProps,
  type FormEvent,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useState,
} from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
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
import { Button, ErrorSummary, Field, Input } from '../ui'
import { apiUrl, authorizedFetch } from '../../lib/api'
import { readApiErrorMessage, readApiFieldErrors } from '../../lib/errors'
import { useForm, useFormField, useFormState, useUnsavedChanges } from '../../lib/form'
import { passwordStrengthEnglish, passwordStrengthKey, type PasswordStrengthKey } from '../../lib/password-strength'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { clearSessionTokens, getRefreshToken } from '../../lib/session-tokens'
import { applyUiTheme, parseUiTheme, type UiTheme } from '../../lib/ui-theme'
import {
  applyUiSurfaceTint,
  readStoredUiSurfaceTint,
  UI_SURFACE_TINT_OPTIONS,
  type UiSurfaceTint,
} from '../../lib/ui-surface-tint'
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

const inputClass =
  'w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2.5 text-sm text-fg-default outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-border-default dark:bg-surface-raised dark:text-fg-default'

/** UX.6 pilot schema for account profile text fields. */
const accountProfileSchema = z.object({
  firstName: z.string().max(80),
  lastName: z.string().max(80),
  phoneNumber: z
    .string()
    .max(30)
    .refine((v) => v.trim() === '' || /^[+0-9().\-\s]{7,30}$/.test(v.trim()), {
      message: 'Enter a valid phone number, including country code if needed.',
    }),
  avatarUrl: z
    .string()
    .max(2048)
    .refine((v) => v.trim() === '' || /^https?:\/\//i.test(v.trim()), {
      message: 'Enter a valid URL, including https://.',
    }),
})

type AccountProfileFormValues = z.infer<typeof accountProfileSchema>

function ProfileTextField({
  formApi,
  name,
  label,
  description,
  ...inputProps
}: {
  formApi: ReturnType<typeof useForm<AccountProfileFormValues>>
  name: keyof AccountProfileFormValues & string
  label: string
  description?: string
} & Omit<ComponentProps<typeof Input>, 'name' | 'value' | 'onBlur' | 'invalid'>) {
  const field = useFormField(formApi, name)
  const { onChange: controlOnChange, ...controlRest } = field.controlProps
  return (
    <Field label={label} description={description} error={field.error} htmlFor={field.id}>
      <Input
        {...controlRest}
        {...inputProps}
        onChange={(e) => {
          controlOnChange(e)
          inputProps.onChange?.(e)
        }}
      />
    </Field>
  )
}

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
  const [deleteAccountBusy, setDeleteAccountBusy] = useState(false)
  const [accountMessage, setAccountMessage] = useState<string | null>(null)
  const [accountError, setAccountError] = useState<string | null>(null)
  const [email, setEmail] = useState('')
  const [avatarPreviewUrl, setAvatarPreviewUrl] = useState<string | null>(null)

  const profileForm = useForm<AccountProfileFormValues>({
    formId: 'account-profile',
    schema: accountProfileSchema,
    defaultValues: { firstName: '', lastName: '', phoneNumber: '', avatarUrl: '' },
    labels: {
      firstName: 'First name',
      lastName: 'Last name',
      phoneNumber: 'Phone number',
      avatarUrl: 'Image URL',
    },
    onSubmit: async (values, { setServerErrors, setFormError, reset }) => {
      setAccountMessage(null)
      setAccountError(null)
      try {
        const res = await authorizedFetch('/api/v1/settings/account', {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            firstName: values.firstName,
            lastName: values.lastName,
            phoneNumber: values.phoneNumber.trim() || null,
            avatarUrl: values.avatarUrl.trim() || null,
            uiTheme,
            showHelpPopover,
          }),
        })
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok) {
          const fieldErrors = readApiFieldErrors(raw)
          if (fieldErrors.length > 0) {
            setServerErrors(fieldErrors)
          } else {
            const msg = readApiErrorMessage(raw)
            setFormError(msg)
            setAccountError(msg)
          }
          return
        }
        const data = raw as AccountProfile
        const nextAvatar = data.avatarUrl ?? ''
        reset({
          firstName: data.firstName ?? '',
          lastName: data.lastName ?? '',
          phoneNumber: data.phoneNumber ?? '',
          avatarUrl: nextAvatar,
        })
        setAvatarPreviewUrl(nextAvatar || null)
        setStudentId(data.sid?.trim() ? data.sid.trim() : null)
        setSessionManagementUiEnabled(data.sessionManagementUiEnabled === true)
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
        const msg = t('common.form.retrySave', {
          defaultValue: 'Could not save. Your entries are still here — try again.',
        })
        setFormError(msg)
        setAccountError(msg)
        toastMutationError(msg)
      }
    },
  })
  const profileFormState = useFormState(profileForm)
  useUnsavedChanges(
    profileFormState.isDirty,
    t('common.form.unsavedLeaveConfirm', {
      defaultValue: 'You have unsaved changes. Leave this page and discard them?',
    }),
  )
  const [avatarModalOpen, setAvatarModalOpen] = useState(false)
  const [avatarPrompt, setAvatarPrompt] = useState('')
  const [avatarGenStatus, setAvatarGenStatus] = useState<'idle' | 'loading' | 'error'>('idle')
  const [avatarGenMessage, setAvatarGenMessage] = useState<string | null>(null)
  const [uiTheme, setUiTheme] = useState<UiTheme>('light')
  const [surfaceTint, setSurfaceTint] = useState<UiSurfaceTint>(() => readStoredUiSurfaceTint())
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
      const currentAvatar = data.avatarUrl ?? ''
      profileForm.reset({
        firstName: names.firstName,
        lastName: names.lastName,
        phoneNumber: data.phoneNumber ?? '',
        avatarUrl: currentAvatar,
      })
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
      profileForm.setValue('avatarUrl', result)
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
    const { firstName, lastName } = profileForm.getValues()
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
        profileForm.setValue('avatarUrl', data.imageUrl)
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
      const { firstName, lastName, avatarUrl } = profileForm.getValues()
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
      const { firstName, lastName, avatarUrl } = profileForm.getValues()
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

  if (accountLoading) {
    return <p className="text-sm text-fg-muted">Loading…</p>
  }

  return (
    <div className="space-y-6">
      <header>
        <h2 className="text-base font-semibold text-fg-default">Account</h2>
        <p className="mt-1 text-sm text-fg-muted">
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
        <form
          id={accountFormId}
          className="space-y-5"
          noValidate
          onSubmit={(e) => {
            void profileForm.handleSubmit(e)
          }}
        >
          <p className="text-xs text-fg-muted">
            {t('common.validation.requiredLegend', {
              defaultValue: 'Required fields are marked with an asterisk (*).',
            })}
          </p>
          <ErrorSummary
            title={t('common.validation.errorSummaryTitle', {
              defaultValue: 'Please fix the following errors',
            })}
            errors={profileFormState.summary.map((s) => ({
              id: s.id,
              label: s.label,
              message: s.message,
            }))}
          />
          {profileFormState.formError ? (
            <p className="rounded-xl border border-danger-fg/40 bg-danger-surface px-4 py-3 text-sm text-danger-fg">
              {profileFormState.formError}
            </p>
          ) : null}
          <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
            <div className="flex shrink-0 flex-col items-center gap-3">
              <div className="flex h-24 w-24 items-center justify-center overflow-hidden rounded-full border border-border-default bg-surface-sunken dark:border-border-default dark:bg-surface-overlay">
                {avatarPreviewUrl ? (
                  <img src={avatarPreviewUrl} alt="" className="h-full w-full object-cover" />
                ) : (
                  <ImageIcon className="h-7 w-7 text-fg-subtle" aria-hidden />
                )}
              </div>
              <div className="flex flex-wrap justify-center gap-2">
                <Button type="button" variant="secondary" size="sm" onClick={openGenerateAvatarModal}>
                  <ImageIcon className="h-3.5 w-3.5" aria-hidden />
                  Generate
                </Button>
                <label className="inline-flex min-h-6 cursor-pointer items-center gap-1.5 rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-xs font-medium text-fg-muted hover:border-border-strong dark:border-border-default dark:bg-surface-overlay dark:text-fg-default">
                  <Upload className="h-3.5 w-3.5" aria-hidden />
                  Upload
                  <input type="file" accept="image/*" className="hidden" onChange={onAvatarUpload} />
                </label>
              </div>
            </div>

            <div className="min-w-0 flex-1 space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <ProfileTextField
                  formApi={profileForm}
                  name="firstName"
                  label="First name"
                  form={accountFormId}
                  autoComplete="given-name"
                  maxLength={80}
                />
                <ProfileTextField
                  formApi={profileForm}
                  name="lastName"
                  label="Last name"
                  form={accountFormId}
                  autoComplete="family-name"
                  maxLength={80}
                />
              </div>

              <Field label="Email" htmlFor="account-email">
                <Input id="account-email" type="email" value={email} disabled autoComplete="email" />
              </Field>

              <ProfileTextField
                formApi={profileForm}
                name="phoneNumber"
                label="Phone number"
                description="Used for SMS notifications when enabled on the Notifications page."
                form={accountFormId}
                type="tel"
                autoComplete="tel"
                maxLength={30}
                placeholder="+1 (555) 555-0100"
                inputMode="tel"
              />

              {studentId ? (
                <Field
                  label="Student ID"
                  htmlFor="account-sid"
                  description="Assigned by your institution. Contact an administrator if this should be updated."
                >
                  <Input id="account-sid" type="text" value={studentId} disabled />
                </Field>
              ) : null}

              <details className="group">
                <summary className="cursor-pointer text-sm font-medium text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default">
                  Advanced: image URL
                </summary>
                <div className="mt-3">
                  <ProfileTextField
                    formApi={profileForm}
                    name="avatarUrl"
                    label="Image URL"
                    form={accountFormId}
                    type="url"
                    autoComplete="off"
                    placeholder="https://example.com/avatar.png"
                    onChange={(e) => {
                      setAvatarPreviewUrl(e.target.value.trim() || null)
                    }}
                  />
                </div>
              </details>
            </div>
          </div>

          {accountMessage ? (
            <p className="text-sm text-success-fg" role="status">
              {accountMessage}
            </p>
          ) : null}

          <Button type="submit" disabled={profileFormState.isSubmitting} className="gap-2">
            <Save className="h-4 w-4" aria-hidden />
            {profileFormState.isSubmitting ? 'Saving…' : 'Save profile'}
          </Button>
        </form>
      </SettingsSection>

      <SettingsSection
        id="security"
        title="Security"
        description="Password, two-factor authentication, and active sign-in sessions."
      >
        <div className="space-y-8">
          <div>
            <h4 className="text-sm font-medium text-fg-default">Password</h4>
            <p className="mt-1 text-xs text-fg-muted">
              Use a unique password you do not reuse on other sites.
            </p>
            <form className="mt-4 space-y-4" onSubmit={onChangePassword}>
              <ul
                id="account-password-requirements"
                className="list-inside list-disc text-xs text-fg-muted"
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
                <span className="mb-1.5 block text-sm font-medium text-fg-default">
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
                <span className="mb-1.5 block text-sm font-medium text-fg-default">
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
                <span className="text-xs font-medium text-fg-muted">Strength:</span>
                <span className="text-xs font-semibold text-fg-default">{cpStrengthLabel}</span>
                <div className="h-1.5 flex-1 rounded-full bg-slate-200 dark:bg-neutral-700" aria-hidden>
                  <div
                    className={`h-full rounded-full ${ cpStrengthKey === 'password.strength.weak' ? 'w-1/3 bg-rose-500' : cpStrengthKey === 'password.strength.fair' ? 'w-2/3 bg-amber-500' : 'w-full bg-emerald-600' }`}
                  />
                </div>
              </div>
              <label className="block">
                <span className="mb-1.5 block text-sm font-medium text-fg-default">
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
                className="rounded-xl border border-border-default bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:border-indigo-200 hover:bg-indigo-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-indigo-400 dark:hover:bg-surface-overlay"
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
                  <h4 className="flex items-center gap-2 text-sm font-medium text-fg-default">
                    <Monitor className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
                    Active sessions
                  </h4>
                  <p className="mt-1 text-xs text-fg-muted">
                    Where you are signed in. Location is approximate when shown.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => void revokeAllOtherSessions()}
                  disabled={sessionsLoading || sessions.filter((s) => !s.isCurrent).length === 0}
                  className="shrink-0 rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm font-medium text-fg-default shadow-sm transition-[background-color,color,border-color] hover:border-rose-200 hover:bg-rose-50 hover:text-rose-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-rose-500/50 dark:hover:bg-rose-950/40"
                >
                  Sign out everywhere else
                </button>
              </div>
              {sessionsError ? (
                <p className="mt-3 text-sm text-rose-600 dark:text-rose-400" role="alert">
                  {sessionsError}
                </p>
              ) : null}
              {sessionsLoading ? <p className="mt-4 text-sm text-fg-muted">Loading sessions…</p> : null}
              {!sessionsLoading && sessions.length === 0 ? (
                <p className="mt-4 text-sm text-fg-muted">No active sessions found.</p>
              ) : null}
              {!sessionsLoading && sessions.length > 0 ? (
                <div className="mt-4 overflow-x-auto rounded-xl border border-border-default">
                  <table
                    className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-600"
                    aria-label="Active sessions"
                  >
                    <thead className="bg-surface-sunken/80">
                      <tr>
                        <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                          Device
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                          Location
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                          Signed in
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                          Last active
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                          Method
                        </th>
                        <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                          Action
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 bg-surface-raised dark:divide-neutral-600 dark:bg-surface-raised">
                      {sessions.map((s) => (
                        <tr
                          key={s.id}
                          className={
                            s.isCurrent
                              ? 'bg-indigo-50/60 dark:bg-indigo-950/25'
                              : 'hover:bg-surface-base dark:hover:bg-neutral-800/60'
                          }
                        >
                          <th
                            scope="row"
                            className="whitespace-nowrap px-3 py-2.5 font-normal text-fg-default"
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
                          <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">
                            {s.location}
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">
                            {formatDateTime(s.createdAt)}
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">
                            {formatDateTime(s.lastUsedAt)}
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">
                            {s.authMethod}
                          </td>
                          <td className="px-3 py-2.5">
                            {s.isCurrent ? (
                              <span className="text-xs text-fg-subtle">—</span>
                            ) : (
                              <button
                                type="button"
                                onClick={() => void revokeSession(s.id)}
                                className="rounded-lg border border-border-default bg-surface-raised px-2.5 py-1.5 text-xs font-medium text-fg-default hover:border-rose-200 hover:bg-rose-50 hover:text-rose-900 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-rose-500/50"
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
            <p className="text-sm font-medium text-fg-default">Theme</p>
            <p className="mt-1 text-sm text-fg-muted">
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
            <p className="text-sm font-medium text-fg-default">Background colour</p>
            <p className="mt-1 text-sm text-fg-muted">
              Tints the app chrome in light and dark mode. Light mode uses soft pastels; dark mode
              uses deep near-black hues. Stored on this device.
            </p>
            <div
              className="mt-3 flex flex-wrap gap-2"
              role="radiogroup"
              aria-label="Background colour"
            >
              {UI_SURFACE_TINT_OPTIONS.map((opt) => {
                const selected = surfaceTint === opt.value
                const swatch = uiTheme === 'dark' ? opt.darkSwatch : opt.lightSwatch
                return (
                  <button
                    key={opt.value}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    aria-label={opt.label}
                    title={opt.label}
                    onClick={() => {
                      setSurfaceTint(opt.value)
                      applyUiSurfaceTint(opt.value)
                    }}
                    className={
                      selected
                        ? 'flex flex-col items-center gap-1.5 rounded-xl border-2 border-accent-solid bg-surface-raised p-2 shadow-sm outline-none ring-2 ring-accent-solid/25'
                        : 'flex flex-col items-center gap-1.5 rounded-xl border border-border-default bg-surface-raised p-2 hover:border-border-strong focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-solid/30'
                    }
                  >
                    <span
                      className="h-8 w-10 rounded-md border border-border-subtle shadow-inner"
                      style={{ backgroundColor: swatch }}
                      aria-hidden
                    />
                    <span className="text-[11px] font-medium text-fg-muted">{opt.label}</span>
                  </button>
                )
              })}
            </div>
          </div>

          <div>
            <p className="text-sm font-medium text-fg-default">Layout density</p>
            <p className="mt-1 text-sm text-fg-muted">
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
            disabled={profileFormState.isSubmitting}
            embedded
          />

          <div>
            <p className="text-sm font-medium text-fg-default">Help button</p>
            <p className="mt-1 text-sm text-fg-muted">
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
          <p className="text-sm text-fg-default">{t('account.delete.warning')}</p>
          <button
            type="button"
            disabled={deleteAccountBusy || accountLoading}
            onClick={() => void onDeleteAccount()}
            className="mt-4 inline-flex items-center gap-2 rounded-xl border border-red-300 bg-surface-raised px-4 py-2.5 text-sm font-semibold text-danger-fg shadow-sm transition-[background-color,color,border-color] hover:border-red-400 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-red-800 dark:bg-surface-raised dark:text-red-300 dark:hover:bg-red-950/50"
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
          <div className="w-full max-w-2xl overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-xl dark:border-border-default dark:bg-surface-raised">
            <div className="flex items-center justify-between border-b border-border-default px-4 py-3 dark:border-border-default">
              <h3 id="generate-avatar-title" className="text-sm font-semibold text-fg-default">
                Generate avatar
              </h3>
              <button
                type="button"
                onClick={() => setAvatarModalOpen(false)}
                className="rounded-lg p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-default dark:hover:bg-surface-overlay dark:hover:text-fg-default"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={onGenerateAvatar} className="grid gap-4 p-4 md:grid-cols-[1fr,240px]">
              <div>
                <label htmlFor="avatar-prompt" className="text-xs font-medium text-fg-muted">
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
                    className="rounded-xl px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-sunken dark:text-fg-muted dark:hover:bg-surface-overlay"
                  >
                    Close
                  </button>
                  <button
                    type="submit"
                    disabled={avatarGenStatus === 'loading' || !avatarPrompt.trim()}
                    className="rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-surface-raised dark:shadow-none"
                  >
                    {avatarGenStatus === 'loading' ? 'Generating…' : 'Generate'}
                  </button>
                </div>
              </div>
              <div>
                <span className="text-xs font-medium text-fg-muted">Preview</span>
                <div className="mt-1 flex h-60 items-center justify-center overflow-hidden rounded-xl border border-dashed border-border-default bg-surface-base dark:border-border-default/50">
                  {avatarGenStatus === 'loading' ? (
                    <span className="text-sm text-fg-muted">Generating…</span>
                  ) : avatarPreviewUrl ? (
                    <img src={avatarPreviewUrl} alt="" className="h-full w-full object-contain" />
                  ) : (
                    <span className="text-sm text-fg-subtle">Generated image will appear here</span>
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