import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Plus, Trash2 } from 'lucide-react'
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

type ServiceToken = {
  id: string
  label: string
  tokenMask: string
  scopes: string[]
  serviceAccountName?: string | null
  isServiceToken?: boolean
  expiresAt?: string | null
  lastUsedAt?: string | null
  revokedAt?: string | null
  createdAt: string
  unusedDays?: number | null
}

export function AdminServiceTokensPanel() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [createdKey, setCreatedKey] = useState<CreateAccessKeyResult | null>(null)
  const [tokens, setTokens] = useState<ServiceToken[]>([])
  const [scopes, setScopes] = useState<ScopeDef[]>([])
  const [error, setError] = useState<string | null>(null)
  const [serviceAccountName, setServiceAccountName] = useState('')
  const [label, setLabel] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['enrollments:read'])
  const [creating, setCreating] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [scopeRes, tokenRes] = await Promise.all([
        authorizedFetch('/api/v1/me/access-keys/scopes'),
        authorizedFetch('/api/v1/admin/tokens'),
      ])
      const scopeRaw: unknown = await scopeRes.json().catch(() => ({}))
      const tokenRaw: unknown = await tokenRes.json().catch(() => ({}))
      if (scopeRes.status === 403 || tokenRes.status === 403) {
        setError(null)
        setTokens([])
        return
      }
      if (!scopeRes.ok) throw new Error(readApiErrorMessage(scopeRaw))
      if (!tokenRes.ok) throw new Error(readApiErrorMessage(tokenRaw))
      setScopes((scopeRaw as { scopes?: ScopeDef[] }).scopes ?? [])
      const all = (tokenRaw as { tokens?: ServiceToken[] }).tokens ?? []
      setTokens(all.filter((t) => t.isServiceToken))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load service tokens.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function createServiceToken() {
    if (!serviceAccountName.trim() || selectedScopes.length === 0) return
    setCreating(true)
    setError(null)
    try {
      const res = await authorizedFetch('/api/v1/admin/tokens', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          serviceAccountName: serviceAccountName.trim(),
          label: label.trim() || serviceAccountName.trim(),
          scopes: selectedScopes,
        }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setError(readApiErrorMessage(raw))
        return
      }
      const created = raw as { token?: string; label?: string }
      if (!created.token) {
        setError('Token was created but the secret was missing from the response.')
        return
      }
      setCreatedKey({ token: created.token, label: created.label ?? label })
      setCreateOpen(false)
      setServiceAccountName('')
      setLabel('')
      toastSaveOk('Service token created.')
      await load()
    } catch {
      toastMutationError('Could not create service token.')
    } finally {
      setCreating(false)
    }
  }

  async function revoke(id: string) {
    if (!(await confirm({ title: t('serviceTokens.revoke.title'), variant: 'danger' }))) return
    try {
      const res = await authorizedFetch(`/api/v1/admin/tokens/${encodeURIComponent(id)}`, { method: 'DELETE' })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        toastMutationError(readApiErrorMessage(raw))
        return
      }
      toastSaveOk('Service token revoked.')
      await load()
    } catch {
      toastMutationError('Could not revoke service token.')
    }
  }

  const active = tokens.filter((t) => !t.revokedAt)
  if (!loading && !error && active.length === 0 && !createOpen) {
    return null
  }

  return (
    <section className="mt-10">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-neutral-100">
            <KeyRound className="h-4 w-4" aria-hidden />
            Institutional service tokens
          </h3>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            Org-scoped credentials for SIS sync, webhooks, and other institutional automation. Org admins only.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setCreateOpen(true)}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          <Plus className="h-4 w-4" aria-hidden />
          New service token
        </button>
      </div>

      {error && (
        <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
          {error}
        </p>
      )}

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading service tokens…</p>
      ) : active.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No service tokens yet.</p>
      ) : (
        <ul className="mt-4 divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-600">
          {active.map((t) => (
            <li key={t.id} className="flex flex-wrap items-start justify-between gap-3 px-4 py-3">
              <div>
                <p className="font-medium text-slate-900 dark:text-neutral-100">{t.label}</p>
                <p className="text-xs text-slate-500 dark:text-neutral-400">{t.serviceAccountName}</p>
                <p className="mt-0.5 font-mono text-xs text-slate-500 dark:text-neutral-400">{t.tokenMask}</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                  {t.scopes.join(', ')} · Created {formatDateTime(t.createdAt)}
                  {t.lastUsedAt ? ` · Last used ${formatDateTime(t.lastUsedAt)}` : ''}
                </p>
                {t.unusedDays != null && t.unusedDays >= 90 && (
                  <p className="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300">
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

      {createOpen && (
        <div className="fixed inset-0 z-[400] flex items-center justify-center p-4" role="presentation">
          <button
            type="button"
            aria-label="Close dialog"
            className="absolute inset-0 bg-black/45"
            onClick={() => setCreateOpen(false)}
          />
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="service-token-title"
            className="relative z-10 w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
          >
            <h2 id="service-token-title" className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
              Create service token
            </h2>
            <div className="mt-4 space-y-3">
              <input
                value={serviceAccountName}
                onChange={(e) => setServiceAccountName(e.target.value)}
                placeholder="Service account name (e.g. SIS roster sync)"
                className="w-full rounded-xl border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
              />
              <input
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="Optional display label"
                className="w-full rounded-xl border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
              />
              <fieldset>
                <legend className="text-xs font-semibold uppercase tracking-wide text-slate-500">Scopes</legend>
                <ul className="mt-2 max-h-40 space-y-1 overflow-y-auto">
                  {scopes.map((s) => (
                    <li key={s.id}>
                      <label className="flex gap-2 text-sm">
                        <input
                          type="checkbox"
                          checked={selectedScopes.includes(s.id)}
                          onChange={() =>
                            setSelectedScopes((prev) =>
                              prev.includes(s.id) ? prev.filter((x) => x !== s.id) : [...prev, s.id],
                            )
                          }
                        />
                        {s.label}
                      </label>
                    </li>
                  ))}
                </ul>
              </fieldset>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button type="button" onClick={() => setCreateOpen(false)} className="rounded-xl border px-4 py-2 text-sm">
                Cancel
              </button>
              <button
                type="button"
                disabled={creating || !serviceAccountName.trim() || selectedScopes.length === 0}
                onClick={() => void createServiceToken()}
                className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60"
              >
                {creating ? 'Creating…' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      <CreateAccessKeyModal open={false} scopes={scopes} onClose={() => {}} onCreated={() => {}} />
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
