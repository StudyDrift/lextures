import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { getImpersonationToken } from '../lib/auth'
import { exitImpersonation, fetchMeProfile, type MeProfile } from '../lib/impersonation'

export function ImpersonationBanner() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const bannerId = useId()
  const [profile, setProfile] = useState<MeProfile | null>(null)
  const [exiting, setExiting] = useState(false)

  const load = useCallback(async () => {
    if (!getImpersonationToken()) {
      setProfile(null)
      return
    }
    const me = await fetchMeProfile()
    setProfile(me?.impersonating ? me : null)
  }, [])

  useEffect(() => {
    void load()
    function onAuthChange() {
      void load()
    }
    window.addEventListener('studydrift-auth-token', onAuthChange)
    return () => window.removeEventListener('studydrift-auth-token', onAuthChange)
  }, [load])

  async function handleExit() {
    setExiting(true)
    try {
      await exitImpersonation()
      navigate('/org-admin/users', { replace: true })
    } finally {
      setExiting(false)
    }
  }

  if (!profile?.impersonating) {
    return null
  }

  const displayName = profile.displayName?.trim() || profile.email

  return (
    <div
      id={bannerId}
      role="status"
      aria-live="polite"
      className="fixed inset-x-0 top-0 z-[9999] flex items-center justify-center gap-3 border-b border-amber-700 bg-amber-500 px-4 py-2 text-sm font-medium text-amber-950 shadow-md"
    >
      <span>
        {t('impersonation.banner.viewingAs', { name: displayName, defaultValue: 'You are viewing as {{name}}.' })}
      </span>
      <button
        type="button"
        disabled={exiting}
        onClick={() => void handleExit()}
        className="rounded border border-amber-900/30 bg-amber-100 px-3 py-1 text-sm font-semibold text-amber-950 hover:bg-amber-50 disabled:opacity-60"
      >
        {exiting
          ? t('impersonation.banner.exiting', { defaultValue: 'Exiting…' })
          : t('impersonation.banner.exit', { defaultValue: 'Exit impersonation' })}
      </button>
    </div>
  )
}
