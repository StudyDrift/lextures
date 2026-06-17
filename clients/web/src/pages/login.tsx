import { type FormEvent, useEffect, useState } from 'react'
import { Link, Navigate, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { BrandLogo } from '../components/brand-logo'
import { OidcSignInButtons } from '../components/oidc-sign-in-buttons'
import { getAccessToken } from '../lib/auth'
import { applyAuthTokenResponse } from '../lib/session-tokens'
import { pickPostAuthPath } from '../lib/post-auth-redirect'
import { apiUrl } from '../lib/api'
import { readApiErrorMessage } from '../lib/errors'
import { applyUiTheme, parseUiTheme } from '../lib/ui-theme'
import { markPostLoginShortcutTip } from '../lib/post-login-shortcut-tip'
import { setMfaFlow } from '../lib/mfa-flow-storage'
import { syncUserLocale } from '../lib/sync-user-locale'
import { normalizeOrgSlug } from '../lib/org-slug'
import {
  authCardClass,
  authFieldClass,
  authMutedLinkClass,
  authOutlineButtonClass,
  authPrimaryButtonClass,
} from '../components/auth/auth-field-classes'
import { MagicLinkRequestForm } from '../components/auth/magic-link-request-form'
import { PublicAuthShell } from '../components/auth/public-auth-shell'

type LocationState = { from?: string; orgSlug?: string }

export default function Login() {
  const { t } = useTranslation('auth')
  const navigate = useNavigate()
  const location = useLocation()
  const params = useParams()
  const state = location.state as LocationState | undefined
  let from = state?.from ?? '/'
  if (
    from === '/login' ||
    from === '/signup' ||
    from === '/forgot-password' ||
    from === '/reset-password' ||
    from.startsWith('/login/mfa') ||
    from.startsWith('/login/magic-link')
  ) {
    from = '/'
  }

  const orgSlug = normalizeOrgSlug(params.orgSlug ?? state?.orgSlug ?? '')
  const [orgName, setOrgName] = useState<string | null>(null)
  const [orgLookupError, setOrgLookupError] = useState<string | null>(null)

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [status, setStatus] = useState<'idle' | 'loading' | 'error'>('idle')
  const [message, setMessage] = useState<string | null>(null)
  const [saml, setSaml] = useState<{
    enabled: boolean
    idp?: { id: string; label: string; forceSaml: boolean }
  } | null>(null)

  useEffect(() => {
    if (!orgSlug) {
      setOrgName(null)
      setOrgLookupError(null)
      return
    }
    let alive = true
    ;(async () => {
      try {
        const res = await fetch(apiUrl(`/api/v1/public/orgs/by-slug/${encodeURIComponent(orgSlug)}`))
        const raw: unknown = await res.json().catch(() => ({}))
        if (!alive) return
        if (!res.ok) {
          setOrgName(null)
          setOrgLookupError(t('auth.login.orgNotFound'))
          return
        }
        const data = raw as { name?: string }
        setOrgName(data.name?.trim() || orgSlug)
        setOrgLookupError(null)
      } catch {
        if (alive) {
          setOrgName(null)
          setOrgLookupError(t('auth.login.serverUnreachable'))
        }
      }
    })()
    return () => {
      alive = false
    }
  }, [orgSlug, t])

  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const res = await fetch(apiUrl('/api/v1/auth/saml/status'))
        const raw: unknown = await res.json().catch(() => ({}))
        if (!alive) return
        const o = raw as {
          enabled?: boolean
          idp?: { id: string; label: string; forceSaml: boolean }
        }
        if (o.enabled && o.idp) {
          setSaml({ enabled: true, idp: o.idp })
        } else if (o.enabled) {
          setSaml({ enabled: true })
        } else {
          setSaml({ enabled: false })
        }
      } catch {
        if (alive) setSaml({ enabled: false })
      }
    })()
    return () => {
      alive = false
    }
  }, [])

  if (getAccessToken()) {
    return <Navigate to="/" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (orgSlug && orgLookupError) return
    setStatus('loading')
    setMessage(null)
    try {
      const body: Record<string, string> = { email, password }
      if (orgSlug) body.org_slug = orgSlug
      const res = await fetch(apiUrl('/api/v1/auth/login'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      let raw: unknown
      try {
        raw = await res.json()
      } catch {
        raw = {}
      }
      if (!res.ok) {
        setStatus('error')
        setMessage(readApiErrorMessage(raw))
        return
      }
      const data = raw as {
        access_token?: string
        mfa_pending_token?: string
        requires_mfa?: boolean
        mfa_setup_required?: boolean
        user?: { email?: string; uiTheme?: string | null; locale?: string | null; accountType?: string }
      }
      const mfaState: LocationState = { from, orgSlug: orgSlug || undefined }
      if (data.requires_mfa && data.mfa_pending_token) {
        setMfaFlow({ token: data.mfa_pending_token, mode: 'challenge' })
        navigate('/login/mfa', { replace: true, state: mfaState })
        return
      }
      if (data.mfa_setup_required && data.mfa_pending_token) {
        setMfaFlow({ token: data.mfa_pending_token, mode: 'setup' })
        navigate('/login/mfa', { replace: true, state: mfaState })
        return
      }
      if (!data.access_token) {
        setStatus('error')
        setMessage(t('auth.login.unexpectedResponse'))
        return
      }
      applyAuthTokenResponse(data)
      applyUiTheme(parseUiTheme(data.user?.uiTheme))
      await syncUserLocale(data.user?.locale)
      markPostLoginShortcutTip()
      navigate(pickPostAuthPath(from), { replace: true })
    } catch {
      setStatus('error')
      setMessage(t('auth.login.serverUnreachable'))
    }
  }

  const subtitle = orgName ? t('auth.login.orgSubtitle', { name: orgName }) : t('auth.login.subtitle')

  return (
    <PublicAuthShell>
      <header className="mb-8 text-center">
        <div className="mb-5 flex justify-center px-2">
          <BrandLogo className="mx-auto h-14 w-auto max-w-[min(100%,240px)] object-contain" />
        </div>
        <h1 className="lex-auth-display text-[1.7rem] leading-snug text-stone-900 dark:text-neutral-50">
          {t('auth.login.title')}
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-stone-600 dark:text-neutral-400">{subtitle}</p>
        {orgSlug && (
          <p className="mt-2 text-xs text-stone-500 dark:text-neutral-500">
            {t('auth.login.orgSlugLabel')}:{' '}
            <code className="rounded bg-stone-100 px-1.5 py-0.5 font-mono dark:bg-neutral-800">{orgSlug}</code>
          </p>
        )}
      </header>

      <div className={authCardClass}>
        {orgLookupError ? (
          <p className="text-sm text-rose-600 dark:text-rose-400" role="alert">
            {orgLookupError}
          </p>
        ) : (
          <>
            <OidcSignInButtons nextPath={from} />
            {saml?.enabled && saml.idp && (
              <div className="mb-6">
                <a
                  className={authOutlineButtonClass}
                  href={apiUrl(
                    `/auth/saml/login?idpId=${encodeURIComponent(saml.idp.id)}&RelayState=${encodeURIComponent(from)}`,
                  )}
                  aria-label={t('auth.login.ssoAria')}
                >
                  {t('auth.login.ssoButton', { label: saml.idp.label })}
                </a>
              </div>
            )}
            {saml?.enabled && saml.idp?.forceSaml && (
              <p className="mb-4 text-center text-sm text-stone-600 dark:text-neutral-400">
                {t('auth.login.ssoRequired')}
              </p>
            )}
            {!saml?.idp?.forceSaml && (
              <form className="space-y-5" onSubmit={onSubmit}>
                <div>
                  <label htmlFor="email" className="mb-1.5 block text-sm font-medium text-stone-800 dark:text-neutral-200">
                    {t('auth.login.email')}
                  </label>
                  <input
                    id="email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    autoFocus
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className={authFieldClass}
                    placeholder={t('auth.login.emailPlaceholder')}
                  />
                </div>
                <div>
                  <label
                    htmlFor="password"
                    className="mb-1.5 block text-sm font-medium text-stone-800 dark:text-neutral-200"
                  >
                    {t('auth.login.password')}
                  </label>
                  <input
                    id="password"
                    name="password"
                    type="password"
                    autoComplete="current-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={authFieldClass}
                    placeholder="••••••••"
                  />
                  <div className="mt-2 text-end">
                    <Link to="/forgot-password" className={`text-sm ${authMutedLinkClass}`}>
                      {t('auth.login.forgotPassword')}
                    </Link>
                  </div>
                </div>

                {message && (
                  <p className="text-sm text-rose-600 dark:text-rose-400" role="status">
                    {message}
                  </p>
                )}

                <button type="submit" disabled={status === 'loading'} className={authPrimaryButtonClass}>
                  {status === 'loading' ? t('auth.login.submitting') : t('auth.login.submit')}
                </button>
              </form>
            )}

            {!saml?.idp?.forceSaml && <MagicLinkRequestForm redirectTo={from} defaultEmail={email} />}

            {!saml?.idp?.forceSaml && (
              <p className="mt-6 text-center text-sm text-stone-600 dark:text-neutral-400">
                {t('auth.login.newHere')}{' '}
                <Link to="/signup" className={authMutedLinkClass}>
                  {t('auth.login.createAccount')}
                </Link>
              </p>
            )}
          </>
        )}
      </div>
    </PublicAuthShell>
  )
}