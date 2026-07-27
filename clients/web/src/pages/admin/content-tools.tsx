import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AdminToolTelemetry } from '../../components/content-tools/analytics/admin-tool-telemetry'
import {
  createContentToolMigration,
  fetchContentToolQuarantine,
  fetchContentToolVersions,
  patchContentToolVersion,
  type ContentToolMigrationJob,
  type ContentToolQuarantineItem,
  type ContentToolVersionRow,
} from '../../lib/content-tools-admin-api'

export default function ContentToolsAdminPage() {
  const { t } = useTranslation('contentTools')
  const titleId = useId()
  const [versions, setVersions] = useState<ContentToolVersionRow[]>([])
  const [sandboxMode, setSandboxMode] = useState('optin')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [migration, setMigration] = useState<ContentToolMigrationJob | null>(null)
  const [quarantine, setQuarantine] = useState<ContentToolQuarantineItem[]>([])

  const refresh = useCallback(async () => {
    const res = await fetchContentToolVersions()
    setVersions(res.versions)
    setSandboxMode(res.sandboxMode)
  }, [])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void refresh()
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [refresh])

  async function run(key: string, fn: () => Promise<void>) {
    setBusy(key)
    setError(null)
    try {
      await fn()
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Request failed')
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="mx-auto max-w-5xl p-6">
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        {t('contentTools.admin.title')}
      </h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        {t('contentTools.admin.help')} · sandbox mode: {sandboxMode}
      </p>
      {loading ? <p className="mt-4 text-sm">{t('contentTools.admin.loading')}</p> : null}
      {error ? (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      ) : null}
      {!loading && versions.length === 0 ? (
        <p className="mt-4 text-sm">{t('contentTools.admin.empty')}</p>
      ) : null}
      {versions.length > 0 ? (
        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full text-left text-sm" aria-labelledby={titleId}>
            <thead>
              <tr className="border-b border-slate-200 dark:border-neutral-700">
                <th className="px-2 py-2 font-medium">{t('contentTools.admin.colTool')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.admin.colVersion')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.admin.colStatus')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.admin.colSandbox')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.admin.colBreaker')}</th>
                <th className="px-2 py-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {versions.map((row) => {
                const key = `${row.toolId}@${row.version}`
                return (
                  <tr
                    key={key}
                    className="border-b border-slate-100 dark:border-neutral-800"
                    data-tool-version={key}
                  >
                    <td className="px-2 py-2">{row.toolId}</td>
                    <td className="px-2 py-2">{row.version}</td>
                    <td className="px-2 py-2">{row.status}</td>
                    <td className="px-2 py-2">{row.sandboxMode}</td>
                    <td className="px-2 py-2" data-breaker={row.breakerOpen ? 'open' : 'closed'}>
                      {row.breakerOpen ? 'open' : 'closed'}
                    </td>
                    <td className="px-2 py-2">
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={busy === key}
                          className="rounded border px-2 py-1 text-xs"
                          onClick={() =>
                            void run(key, async () => {
                              await patchContentToolVersion(row.toolId, row.version, {
                                resetBreaker: true,
                                status: 'active',
                              })
                            })
                          }
                        >
                          {t('contentTools.admin.resetBreaker')}
                        </button>
                        <button
                          type="button"
                          disabled={busy === key}
                          className="rounded border px-2 py-1 text-xs"
                          onClick={() =>
                            void run(key, async () => {
                              await patchContentToolVersion(row.toolId, row.version, {
                                status: row.status === 'deprecated' ? 'active' : 'deprecated',
                              })
                            })
                          }
                        >
                          {row.status === 'deprecated'
                            ? t('contentTools.admin.activate')
                            : t('contentTools.admin.deprecate')}
                        </button>
                        <button
                          type="button"
                          disabled={busy === key}
                          className="rounded border px-2 py-1 text-xs"
                          onClick={() =>
                            void run(key, async () => {
                              const job = await createContentToolMigration({
                                toolId: row.toolId,
                                fromVersion: 1,
                                toVersion: Math.max(row.stateSchemaVersion, 2),
                                dryRun: true,
                              })
                              setMigration(job)
                            })
                          }
                        >
                          {t('contentTools.admin.migrateDryRun')}
                        </button>
                        <button
                          type="button"
                          disabled={busy === key}
                          className="rounded border px-2 py-1 text-xs"
                          onClick={() =>
                            void run(key, async () => {
                              const q = await fetchContentToolQuarantine(row.toolId)
                              setQuarantine(q.items)
                            })
                          }
                        >
                          {t('contentTools.admin.quarantine')}
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      ) : null}
      {migration ? (
        <p className="mt-4 text-sm" data-testid="migration-result" role="status">
          {t('contentTools.admin.migrationResult', {
            migrated: migration.migratedDocs,
            failed: migration.failedDocs,
            total: migration.totalDocs,
          })}{' '}
          ({migration.status}
          {migration.dryRun ? ', dry-run' : ''})
        </p>
      ) : null}
      {quarantine.length > 0 ? (
        <ul className="mt-4 list-disc pl-5 text-sm" data-testid="quarantine-list">
          {quarantine.map((q) => (
            <li key={q.id}>
              {q.toolId}: v{q.fromVersion}→v{q.toVersion} — {q.error}
            </li>
          ))}
        </ul>
      ) : null}
      <div className="mt-8 border-t border-slate-200 pt-6 dark:border-neutral-700">
        <AdminToolTelemetry />
      </div>
    </div>
  )
}
