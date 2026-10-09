import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { getImpersonationToken } from '../lib/auth'
import { exitImpersonation, fetchMeProfile, type MeProfile } from '../lib/impersonation'
import { exitLearnAs } from '../lib/managed-learners-api'

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
    setProfile(me?.impersonating || me?.learningAs ? me : null)
  }, [])

  useEffect(() => {
    void load()
    function onAuthChange() {
      void load()
    }
    window.addEventListener('studydrift-auth-token', onAuthChange)
    return () => window.removeEventListener('studydrift-auth-token', onAuthChange)
  }, [load])

  const isLearnAs = Boolean(profile?.learningAs)
  const isImpersonation = Boolean(profile?.impersonating)

  async function handleExit() {
    setExiting(true)
    try {
      if (isLearnAs) {
        await exitLearnAs()
        navigate('/learners', { replace: true })
      } else {
        await exitImpersonation()
        navigate('/org-admin/users', { replace: true })
      }
    } finally {
      setExiting(false)
    }
  }

  if (!isImpersonation && !isLearnAs) {
    return null
  }

  const displayName = profile?.displayName?.trim() || profile?.email || 'learner'

  return (
    <div
      id={bannerId}
      role="status"
      aria-live="polite"
      className="relative z-50 flex shrink-0 flex-wrap items-center justify-center gap-x-3 gap-y-1 border-b border-amber-700 bg-amber-500 px-4 py-2 text-sm font-medium text-amber-950 shadow-md"
    >
      <span>
        {isLearnAs
          ? `Learning as ${displayName} — Exit to parent.`
          : t('impersonation.banner.viewingAs', { name: displayName, defaultValue: 'You are viewing as {{name}}.' })}
      </span>
      <button
        type="button"
        disabled={exiting}
        onClick={() => void handleExit()}
        className="rounded border border-amber-900/30 bg-amber-100 px-3 py-1 text-sm font-semibold text-amber-950 hover:bg-amber-50 disabled:opacity-60"
      >
        {exiting
          ? t('impersonation.banner.exiting', { defaultValue: 'Exiting…' })
          : isLearnAs
            ? 'Exit to parent'
            : t('impersonation.banner.exit', { defaultValue: 'Exit impersonation' })}
      </button>
    </div>
  )
}
