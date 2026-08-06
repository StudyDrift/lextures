import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useConfirm } from '../use-confirm'
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
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
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

  async function rotate(id: string) {
    if (!(await confirm({ title: t('accessKeys.rotate.title'), variant: 'danger' }))) return
    setError(null)
    try {
      const res = await authorizedFetch(`/api/v1/me/access-keys/${encodeURIComponent(id)}/rotate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ overlapHours: 24 }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        toastMutationError(readApiErrorMessage(raw))
        return
      }
      const created = raw as { token?: string; label?: string }
      if (!created.token) {
        toastMutationError('Key was rotated but the secret was missing from the response.')
        return
      }
      setCreatedKey({ token: created.token, label: created.label ?? 'Rotated key' })
      toastSaveOk('Access key rotated.')
      await load()
    } catch {
      toastMutationError('Could not rotate access key.')
    }
  }

  async function revoke(id: string) {
    if (!(await confirm({ title: t('accessKeys.revoke.title'), variant: 'danger' }))) return
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
          <h3 className="flex items-center gap-2 text-sm font-semibold text-fg-default">
            <KeyRound className="h-4 w-4" aria-hidden />
            Access keys
          </h3>
          <p className="mt-1 text-sm text-fg-muted">
            Long-lived credentials for API tools and MCP agents. Each key starts with{' '}
            <code className="font-mono text-xs">ltk_</code>.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setCreateOpen(true)}
          className="inline-flex items-center gap-2 rounded-xl bg-accent-solid px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-500 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-surface-raised"
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
        <p className="mt-4 text-sm text-fg-muted">Loading access keys…</p>
      ) : activeKeys.length === 0 && revokedKeys.length === 0 ? (
        <div className="mt-4 rounded-xl border border-dashed border-border-default bg-slate-50/50 px-4 py-8 text-center dark:border-border-default/30">
          <p className="text-sm text-fg-muted">No access keys yet.</p>
          <p className="mt-1 text-xs text-fg-muted">
            Create one to connect scripts, automation, or an AI agent via MCP.
          </p>
          <button
            type="button"
            onClick={() => setCreateOpen(true)}
            className="mt-4 inline-flex items-center gap-2 rounded-xl bg-accent-solid px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-500 dark:bg-neutral-100 dark:text-neutral-950"
          >
            <Plus className="h-4 w-4" aria-hidden />
            Create your first key
          </button>
        </div>
      ) : (
        <div className="mt-4 space-y-6">
          {activeKeys.length > 0 && (
            <ul className="divide-y divide-slate-200 rounded-xl border border-border-default dark:divide-neutral-700 dark:border-border-default">
              {activeKeys.map((t) => (
                <li key={t.id} className="flex flex-wrap items-start justify-between gap-3 px-4 py-3">
                  <div className="min-w-0">
                    <p className="font-medium text-fg-default">{t.label}</p>
                    <p className="mt-0.5 font-mono text-xs text-fg-muted">{t.tokenMask}</p>
                    <dl className="mt-2 grid gap-1 text-xs text-fg-muted sm:grid-cols-2 sm:gap-x-4">
                      <div>
                        <dt className="inline font-medium text-fg-muted">Permissions: </dt>
                        <dd className="inline">{t.scopes.join(', ')}</dd>
                      </div>
                      <div>
                        <dt className="inline font-medium text-fg-muted">Courses: </dt>
                        <dd className="inline">{courseSummary(t)}</dd>
                      </div>
                      <div className="sm:col-span-2">
                        <dt className="inline font-medium text-fg-muted">Created: </dt>
                        <dd className="inline">
                          {formatDateTime(t.createdAt)}
                          {t.lastUsedAt ? ` · Last used ${formatDateTime(t.lastUsedAt)}` : ' · Never used'}
                          {t.expiresAt ? ` · Expires ${formatDateTime(t.expiresAt)}` : ''}
                        </dd>
                      </div>
                    </dl>
                    {t.unusedDays != null && t.unusedDays >= 90 && (
                      <p className="mt-2 text-xs font-medium text-warning-fg">
                        Unused {t.unusedDays} days — consider revoking
                      </p>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => void rotate(t.id)}
                      className="inline-flex items-center gap-1 rounded-lg border border-border-default px-3 py-1.5 text-sm text-fg-muted hover:bg-surface-base dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
                    >
                      <RefreshCw className="h-4 w-4" aria-hidden />
                      Rotate
                    </button>
                    <button
                      type="button"
                      onClick={() => void revoke(t.id)}
                      className="inline-flex items-center gap-1 rounded-lg border border-rose-200 px-3 py-1.5 text-sm text-rose-700 hover:bg-rose-50 dark:border-rose-900/50 dark:text-rose-300 dark:hover:bg-rose-950/40"
                    >
                      <Trash2 className="h-4 w-4" aria-hidden />
                      Revoke
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}

          {revokedKeys.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                Revoked
              </h4>
              <ul className="mt-2 divide-y divide-slate-200 rounded-xl border border-border-default opacity-75 dark:divide-neutral-700 dark:border-border-default">
                {revokedKeys.map((t) => (
                  <li key={t.id} className="px-4 py-3">
                    <p className="text-sm text-fg-muted line-through dark:text-fg-muted">{t.label}</p>
                    <p className="mt-0.5 font-mono text-xs text-fg-subtle">{t.tokenMask}</p>
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
      {ConfirmDialogHost}
    </section>
  )
}
