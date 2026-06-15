import { useCallback, useEffect, useState } from 'react'
import { KeyRound, Plus, Trash2 } from 'lucide-react'
import {
  AccessKeyCreatedModal,
  CreateAccessKeyModal,
  type CreateAccessKeyResult,
} from './create-access-key-modal'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { formatDateTime } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type ScopeDef = {
  id: string
  label: string
  description: string
  group: string
}

type AccessKeyCourse = {
  id: string
  courseCode: string
  title: string
}

type AccessKey = {
  id: string
  label: string
  tokenMask: string
  scopes: string[]
  courseIds?: string[]
  courses?: AccessKeyCourse[]
  allCourses?: boolean
  expiresAt?: string | null
  lastUsedAt?: string | null
  revokedAt?: string | null
  createdAt: string
  unusedDays?: number | null
}

function courseSummary(key: AccessKey): string {
  if (key.allCourses !== false && (!key.courseIds || key.courseIds.length === 0)) {
    return 'All courses'
  }
  const codes = (key.courses ?? []).map((c) => c.courseCode)
  if (codes.length > 0) return codes.join(', ')
  if (key.courseIds?.length) return `${key.courseIds.length} course(s)`
  return 'None'
}

export function IntegrationsAccessKeysPanel() {
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [createdKey, setCreatedKey] = useState<CreateAccessKeyResult | null>(null)
  const [tokens, setTokens] = useState<AccessKey[]>([])
  const [scopes, setScopes] = useState<ScopeDef[]>([])
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [scopeRes, tokenRes] = await Promise.all([
        authorizedFetch('/api/v1/me/access-keys/scopes'),
        authorizedFetch('/api/v1/me/access-keys'),
      ])
      const scopeRaw: unknown = await scopeRes.json().catch(() => ({}))
      const tokenRaw: unknown = await tokenRes.json().catch(() => ({}))
      if (!scopeRes.ok) throw new Error(readApiErrorMessage(scopeRaw))
      if (!tokenRes.ok) throw new Error(readApiErrorMessage(tokenRaw))
      setScopes((scopeRaw as { scopes?: ScopeDef[] }).scopes ?? [])
      setTokens((tokenRaw as { tokens?: AccessKey[] }).tokens ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load access keys.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  function handleCreated(result: CreateAccessKeyResult) {
    setCreatedKey(result)
    toastSaveOk('Access key created.')
    void load()
  }

  async function revoke(id: string) {
    if (!globalThis.confirm('Revoke this access key? Tools using it will stop working immediately.')) return
    setError(null)
    try {
      const res = await authorizedFetch(`/api/v1/me/access-keys/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        toastMutationError(readApiErrorMessage(raw))
        return
      }
      toastSaveOk('Access key revoked.')
      await load()
    } catch {
      toastMutationError('Could not revoke access key.')
    }
  }

  const activeKeys = tokens.filter((t) => !t.revokedAt)
  const revokedKeys = tokens.filter((t) => t.revokedAt)

  return (
    <section className="mt-8">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-neutral-100">
            <KeyRound className="h-4 w-4" aria-hidden />
            Access keys
          </h3>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            Long-lived credentials for API tools and MCP agents. Each key starts with{' '}
            <code className="font-mono text-xs">ltk_</code>.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setCreateOpen(true)}
          className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-500 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white"
        >
          <Plus className="h-4 w-4" aria-hidden />
          New key
        </button>
      </div>

      {error && (
        <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
          {error}
        </p>
      )}

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading access keys…</p>
      ) : activeKeys.length === 0 && revokedKeys.length === 0 ? (
        <div className="mt-4 rounded-xl border border-dashed border-slate-200 bg-slate-50/50 px-4 py-8 text-center dark:border-neutral-600 dark:bg-neutral-800/30">
          <p className="text-sm text-slate-600 dark:text-neutral-300">No access keys yet.</p>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Create one to connect scripts, automation, or an AI agent via MCP.
          </p>
          <button
            type="button"
            onClick={() => setCreateOpen(true)}
            className="mt-4 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-500 dark:bg-neutral-100 dark:text-neutral-950"
          >
            <Plus className="h-4 w-4" aria-hidden />
            Create your first key
          </button>
        </div>
      ) : (
        <div className="mt-4 space-y-6">
          {activeKeys.length > 0 && (
            <ul className="divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-600">
              {activeKeys.map((t) => (
                <li key={t.id} className="flex flex-wrap items-start justify-between gap-3 px-4 py-3">
                  <div className="min-w-0">
                    <p className="font-medium text-slate-900 dark:text-neutral-100">{t.label}</p>
                    <p className="mt-0.5 font-mono text-xs text-slate-500 dark:text-neutral-400">{t.tokenMask}</p>
                    <dl className="mt-2 grid gap-1 text-xs text-slate-500 dark:text-neutral-400 sm:grid-cols-2 sm:gap-x-4">
                      <div>
                        <dt className="inline font-medium text-slate-600 dark:text-neutral-300">Permissions: </dt>
                        <dd className="inline">{t.scopes.join(', ')}</dd>
                      </div>
                      <div>
                        <dt className="inline font-medium text-slate-600 dark:text-neutral-300">Courses: </dt>
                        <dd className="inline">{courseSummary(t)}</dd>
                      </div>
                      <div className="sm:col-span-2">
                        <dt className="inline font-medium text-slate-600 dark:text-neutral-300">Created: </dt>
                        <dd className="inline">
                          {formatDateTime(t.createdAt)}
                          {t.lastUsedAt ? ` · Last used ${formatDateTime(t.lastUsedAt)}` : ' · Never used'}
                          {t.expiresAt ? ` · Expires ${formatDateTime(t.expiresAt)}` : ''}
                        </dd>
                      </div>
                    </dl>
                    {t.unusedDays != null && t.unusedDays >= 90 && (
                      <p className="mt-2 text-xs font-medium text-amber-700 dark:text-amber-300">
                        Unused {t.unusedDays} days — consider revoking
                      </p>
                    )}
                  </div>
                  <button
                    type="button"
                    onClick={() => void revoke(t.id)}
                    className="inline-flex items-center gap-1 rounded-lg border border-rose-200 px-3 py-1.5 text-sm text-rose-700 hover:bg-rose-50 dark:border-rose-900/50 dark:text-rose-300 dark:hover:bg-rose-950/40"
                  >
                    <Trash2 className="h-4 w-4" aria-hidden />
                    Revoke
                  </button>
                </li>
              ))}
            </ul>
          )}

          {revokedKeys.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Revoked
              </h4>
              <ul className="mt-2 divide-y divide-slate-200 rounded-xl border border-slate-200 opacity-75 dark:divide-neutral-700 dark:border-neutral-600">
                {revokedKeys.map((t) => (
                  <li key={t.id} className="px-4 py-3">
                    <p className="text-sm text-slate-600 line-through dark:text-neutral-400">{t.label}</p>
                    <p className="mt-0.5 font-mono text-xs text-slate-400 dark:text-neutral-500">{t.tokenMask}</p>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      <CreateAccessKeyModal
        open={createOpen}
        scopes={scopes}
        onClose={() => setCreateOpen(false)}
        onCreated={handleCreated}
      />

      <AccessKeyCreatedModal
        open={createdKey != null}
        token={createdKey?.token ?? null}
        label={createdKey?.label ?? null}
        onClose={() => setCreatedKey(null)}
      />
    </section>
  )
}
