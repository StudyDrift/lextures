import { type FormEvent, useEffect, useState } from 'react'
import { Link, Navigate, useSearchParams } from 'react-router-dom'
import {
  authCardClass,
  authFieldClass,
  authMutedLinkClass,
  authPrimaryButtonClass,
} from '../components/auth/auth-field-classes'
import { PublicAuthShell } from '../components/auth/public-auth-shell'
import { BrandLogo } from '../components/brand-logo'
import { getAccessToken } from '../lib/auth'
import { apiUrl } from '../lib/api'
import { consumeParentInvite } from '../lib/parent-assign-api'
import { passwordStrengthEnglish, passwordStrengthKey, type PasswordStrengthKey } from '../lib/password-strength'

export default function ActivateParentPage() {
  const [searchParams] = useSearchParams()
  const tokenFromUrl = searchParams.get('token') ?? ''

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [status, setStatus] = useState<'idle' | 'loading' | 'error' | 'done'>('idle')
  const [message, setMessage] = useState<string | null>(null)
  const [policy, setPolicy] = useState<{
    minLength: number
  } | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await fetch(apiUrl('/api/v1/auth/password-policy'))
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok || cancelled) return
        const p = raw as { minLength?: number }
        setPolicy({
          minLength: typeof p.minLength === 'number' ? p.minLength : 8,
        })
      } catch {
        /* ignore */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const minLen = policy?.minLength ?? 8
  const strengthKey: PasswordStrengthKey = passwordStrengthKey(password)
  const strengthLabel = passwordStrengthEnglish(strengthKey)

  if (getAccessToken()) {
    return <Navigate to="/parent" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setMessage(null)
    if (!tokenFromUrl.trim()) {
      setStatus('error')
      setMessage('This page needs a valid activate link from your invite email.')
      return
    }
    if (password !== confirm) {
      setStatus('error')
      setMessage('Passwords do not match.')
      return
    }
    if (password.length < minLen) {
      setStatus('error')
      setMessage(`Password must be at least ${minLen} characters.`)
      return
    }

    setStatus('loading')
    try {
      const data = await consumeParentInvite(tokenFromUrl, password)
      setStatus('done')
      setMessage(data.message)
    } catch (err) {
      setStatus('error')
      setMessage(err instanceof Error ? err.message : 'Could not activate your account.')
    }
  }

  return (
    <PublicAuthShell>
      <header className="mb-8 text-center">
        <div className="mb-5 flex justify-center px-2">
          <BrandLogo className="mx-auto h-14 w-auto max-w-[min(100%,240px)] object-contain" />
        </div>
        <h1 className="lex-auth-display text-[1.7rem] leading-snug text-stone-900 dark:text-neutral-50">
          Activate parent account
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-stone-600 dark:text-neutral-400">
          Set a password to connect to your child&apos;s Family dashboard.
        </p>
      </header>

      <div className={authCardClass}>
        {status === 'done' ? (
          <div className="space-y-4 text-center">
            <p className="text-sm text-stone-700 dark:text-neutral-300" role="status">
              {message}
            </p>
            <Link to="/login" state={{ from: '/parent' }} className={`inline-block text-sm ${authMutedLinkClass}`}>
              Sign in to Family dashboard
            </Link>
          </div>
        ) : (
          <form className="space-y-5" onSubmit={onSubmit}>
            {!tokenFromUrl.trim() ? (
              <p className="text-sm text-amber-700" role="status">
                Missing token. Open the link from your invite email, or ask school staff to resend it.
              </p>
            ) : null}
            <div>
              <label
                htmlFor="activate-password"
                className="mb-1.5 block text-sm font-medium text-stone-800 dark:text-neutral-200"
              >
                New password
              </label>
              <input
                id="activate-password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={authFieldClass}
                required
              />
              {password ? (
                <p className="mt-1 text-xs text-stone-500">Strength: {strengthLabel}</p>
              ) : null}
            </div>
            <div>
              <label
                htmlFor="activate-confirm"
                className="mb-1.5 block text-sm font-medium text-stone-800 dark:text-neutral-200"
              >
                Confirm password
              </label>
              <input
                id="activate-confirm"
                type="password"
                autoComplete="new-password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                className={authFieldClass}
                required
              />
            </div>
            {status === 'error' && message ? (
              <p className="text-sm text-red-700 dark:text-red-300" role="alert">
                {message}
              </p>
            ) : null}
            <button type="submit" className={authPrimaryButtonClass} disabled={status === 'loading'}>
              {status === 'loading' ? 'Activating…' : 'Activate account'}
            </button>
          </form>
        )}
      </div>
    </PublicAuthShell>
  )
}
