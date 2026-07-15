import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  checkBadgeHandleAvailable,
  fetchBadgeProfile,
  patchBadgeProfile,
  type BadgeProfile,
} from '../../lib/badges-api'

export function BadgeProfileSettingsPanel() {
  const { ffCompetencyBadges, loading: featuresLoading } = usePlatformFeatures()
  const handleId = useId()
  const [profile, setProfile] = useState<BadgeProfile | null>(null)
  const [handle, setHandle] = useState('')
  const [pagePublic, setPagePublic] = useState(false)
  const [searchIndexable, setSearchIndexable] = useState(false)
  const [hideRealName, setHideRealName] = useState(false)
  const [availability, setAvailability] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [loaded, setLoaded] = useState(false)

  const load = useCallback(async () => {
    try {
      const p = await fetchBadgeProfile()
      setProfile(p)
      setHandle(p.handle)
      setPagePublic(p.pagePublic)
      setSearchIndexable(p.searchIndexable)
      setHideRealName(p.hideRealName)
      setLoaded(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load badge profile.')
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffCompetencyBadges) return
    void load()
  }, [featuresLoading, ffCompetencyBadges, load])

  useEffect(() => {
    if (!handle || handle === profile?.handle) {
      setAvailability(null)
      return
    }
    const t = window.setTimeout(() => {
      void checkBadgeHandleAvailable(handle)
        .then((r) => {
          if (!r.valid) setAvailability(r.reason ?? 'Invalid handle')
          else if (!r.available) setAvailability('Already taken')
          else setAvailability('Available')
        })
        .catch(() => setAvailability(null))
    }, 300)
    return () => window.clearTimeout(t)
  }, [handle, profile?.handle])

  if (!ffCompetencyBadges) return null

  async function onSave() {
    setSaving(true)
    setError(null)
    try {
      const body: {
        handle?: string
        pagePublic?: boolean
        searchIndexable?: boolean
        hideRealName?: boolean
      } = {
        pagePublic,
        searchIndexable,
        hideRealName,
      }
      if (handle && handle !== profile?.handle) {
        body.handle = handle
      }
      const p = await patchBadgeProfile(body)
      setProfile(p)
      setHandle(p.handle)
      setPagePublic(p.pagePublic)
      setSearchIndexable(p.searchIndexable)
      setHideRealName(p.hideRealName)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save.')
    } finally {
      setSaving(false)
    }
  }

  if (!loaded && !error) {
    return (
      <section className="rounded-xl border border-slate-200 p-4 dark:border-neutral-700">
        <h2 className="text-base font-semibold text-slate-900 dark:text-white">Public badge page</h2>
        <p className="mt-1 text-sm text-slate-500">Loading…</p>
      </section>
    )
  }

  return (
    <section className="rounded-xl border border-slate-200 p-4 dark:border-neutral-700">
      <h2 className="text-base font-semibold text-slate-900 dark:text-white">Public badge page</h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
        Choose a handle for your shareable badge backpack URL. Pages are private by default.
      </p>

      {error ? (
        <p role="alert" className="mt-3 text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}

      <div className="mt-4 space-y-4">
        <div>
          <label htmlFor={handleId} className="mb-1.5 block text-sm font-medium">
            Handle
          </label>
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-500">/badges/</span>
            <input
              id={handleId}
              value={handle}
              onChange={(e) => setHandle(e.target.value.toLowerCase())}
              className="w-full max-w-xs rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          {availability ? (
            <p className="mt-1 text-xs text-slate-500" aria-live="polite">
              {availability}
            </p>
          ) : null}
          {profile?.publicUrl ? (
            <p className="mt-2 text-xs text-slate-500">
              Public URL:{' '}
              <a href={profile.publicUrl} className="text-indigo-600 hover:underline dark:text-indigo-400">
                {profile.publicUrl}
              </a>
            </p>
          ) : null}
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={pagePublic}
            onChange={(e) => setPagePublic(e.target.checked)}
          />
          Make my badge page public
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={searchIndexable}
            onChange={(e) => setSearchIndexable(e.target.checked)}
            disabled={!pagePublic}
          />
          Allow search engines to index my page
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={hideRealName}
            onChange={(e) => setHideRealName(e.target.checked)}
          />
          Hide my real name (show handle only)
        </label>

        <button
          type="button"
          onClick={() => void onSave()}
          disabled={saving}
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          {saving ? 'Saving…' : 'Save badge page settings'}
        </button>
      </div>
    </section>
  )
}
