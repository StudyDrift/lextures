import { useCallback, useEffect, useId, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Award, CheckCircle2, Download, Loader2, Share2, Wallet } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { authorizedFetch } from '../../lib/api'
import { formatDate } from '../../lib/format'
import {
  createWalletCollection,
  deleteWalletCollection,
  downloadWalletExport,
  fetchWallet,
  fetchWalletCollectionAccess,
  fetchWalletCollections,
  fetchWalletExport,
  revokeWalletCollection,
  startWalletExport,
  type WalletAccessEvent,
  type WalletCollection,
  type WalletDisclosure,
  type WalletItem,
  type WalletKind,
} from '../../lib/wallet-api'
import { LmsPage } from './lms-page'

function kindLabel(kind: WalletKind, t: (k: string) => string): string {
  switch (kind) {
    case 'transcript':
      return t('wallet.kind.transcript')
    case 'clr':
      return t('wallet.kind.clr')
    case 'badge':
      return t('wallet.kind.badge')
    case 'certificate':
      return t('wallet.kind.certificate')
    case 'diploma':
      return t('wallet.kind.diploma')
    case 'ce_record':
      return t('wallet.kind.ceRecord')
    default: {
      const _exhaustive: never = kind
      return _exhaustive
    }
  }
}

function useWalletEnabled(): boolean {
  const {
    ffTranscripts,
    ffCoCurricularTranscript,
    ffCompetencyBadges,
    ffCompletionCredentials,
    ffCeuTracking,
    ffDiplomas,
  } = usePlatformFeatures()
  return (
    ffTranscripts ||
    ffCoCurricularTranscript ||
    ffCompetencyBadges ||
    ffCompletionCredentials ||
    ffCeuTracking ||
    ffDiplomas
  )
}

export default function CredentialWalletPage() {
  const { t } = useTranslation('common')
  const titleId = useId()
  const shareDialogTitleId = useId()
  const { loading: featuresLoading } = usePlatformFeatures()
  const enabled = useWalletEnabled()

  const [items, setItems] = useState<WalletItem[]>([])
  const [collections, setCollections] = useState<WalletCollection[]>([])
  const [alumniNote, setAlumniNote] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [shareOpen, setShareOpen] = useState(false)
  const [shareName, setShareName] = useState('')
  const [disclosure, setDisclosure] = useState<WalletDisclosure>('validity')
  const [sharing, setSharing] = useState(false)
  const [shareResult, setShareResult] = useState<WalletCollection | null>(null)

  const [accessFor, setAccessFor] = useState<string | null>(null)
  const [accessEvents, setAccessEvents] = useState<WalletAccessEvent[]>([])

  const [exporting, setExporting] = useState(false)
  const [exportStatus, setExportStatus] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [wallet, cols] = await Promise.all([fetchWallet(), fetchWalletCollections()])
      setItems(wallet.items)
      setAlumniNote(wallet.alumniNote ?? null)
      setCollections(cols)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.loadError'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (featuresLoading || !enabled) return
    void load()
  }, [featuresLoading, enabled, load])

  const grouped = useMemo(() => {
    const map = new Map<WalletKind, WalletItem[]>()
    for (const item of items) {
      const list = map.get(item.kind) ?? []
      list.push(item)
      map.set(item.kind, list)
    }
    return map
  }, [items])

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function onCreateShare() {
    if (selected.size === 0) return
    setSharing(true)
    setError(null)
    try {
      const col = await createWalletCollection({
        name: shareName.trim() || t('wallet.defaultCollectionName'),
        disclosure,
        itemIds: [...selected],
        share: true,
      })
      setShareResult(col)
      setCollections((prev) => [col, ...prev])
      setSelected(new Set())
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.shareError'))
    } finally {
      setSharing(false)
    }
  }

  async function onRevoke(id: string) {
    try {
      const updated = await revokeWalletCollection(id)
      setCollections((prev) => prev.map((c) => (c.id === id ? updated : c)))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.revokeError'))
    }
  }

  async function onDelete(id: string) {
    try {
      await deleteWalletCollection(id)
      setCollections((prev) => prev.filter((c) => c.id !== id))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.deleteError'))
    }
  }

  async function onShowAccess(id: string) {
    setAccessFor(id)
    try {
      setAccessEvents(await fetchWalletCollectionAccess(id))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.accessError'))
    }
  }

  async function onDownloadPath(path: string, filename: string) {
    try {
      const res = await authorizedFetch(path)
      if (!res.ok) throw new Error(t('wallet.downloadError'))
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.downloadError'))
    }
  }

  async function onExport() {
    setExporting(true)
    setExportStatus(t('wallet.export.starting'))
    setError(null)
    try {
      let status = await startWalletExport()
      for (let i = 0; i < 40 && status.status === 'pending'; i++) {
        setExportStatus(t('wallet.export.generating'))
        await new Promise((r) => setTimeout(r, 500))
        status = await fetchWalletExport(status.id)
      }
      if (status.status === 'ready') {
        setExportStatus(t('wallet.export.ready'))
        await downloadWalletExport(status.id)
      } else if (status.status === 'failed') {
        throw new Error(status.error || t('wallet.export.failed'))
      } else {
        throw new Error(t('wallet.export.timeout'))
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t('wallet.export.failed'))
      setExportStatus(null)
    } finally {
      setExporting(false)
    }
  }

  if (!enabled && !featuresLoading) {
    return (
      <LmsPage title={t('wallet.title')}>
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('wallet.disabled')}</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title={t('wallet.title')}>
      <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1
            id={titleId}
            className="flex items-center gap-2 text-2xl font-semibold text-slate-900 dark:text-white"
          >
            <Wallet className="h-7 w-7 text-emerald-600" aria-hidden />
            {t('wallet.title')}
          </h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">{t('wallet.help')}</p>
          {alumniNote ? (
            <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">{t('wallet.alumniNote')}</p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
            disabled={selected.size === 0}
            onClick={() => {
              setShareResult(null)
              setShareName('')
              setDisclosure('validity')
              setShareOpen(true)
            }}
          >
            <Share2 className="h-4 w-4" aria-hidden />
            {t('wallet.shareSelected')}
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-800 disabled:opacity-50 dark:border-slate-600 dark:text-slate-100"
            disabled={exporting || items.length === 0}
            onClick={() => void onExport()}
          >
            {exporting ? <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden /> : <Download className="h-4 w-4" aria-hidden />}
            {t('wallet.export.downloadAll')}
          </button>
        </div>
      </header>

      {exportStatus ? (
        <p className="mb-4 text-sm text-slate-600 dark:text-slate-300" aria-live="polite">
          {exportStatus}
        </p>
      ) : null}

      {loading ? (
        <p className="inline-flex items-center gap-2 text-sm text-slate-600">
          <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
          {t('common.loading')}
        </p>
      ) : null}

      {error ? (
        <p role="alert" className="mb-4 text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}

      {!loading && items.length === 0 ? (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('wallet.empty')}</p>
      ) : null}

      <div className="space-y-8">
        {[...grouped.entries()].map(([kind, kindItems]) => (
          <section key={kind} aria-labelledby={`wallet-kind-${kind}`}>
            <h2 id={`wallet-kind-${kind}`} className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
              {kindLabel(kind, t)}
            </h2>
            <ul className="grid gap-3 md:grid-cols-2">
              {kindItems.map((item) => {
                const downloadPath =
                  typeof item.metadata?.downloadPath === 'string' ? item.metadata.downloadPath : null
                return (
                  <li
                    key={item.id}
                    className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800"
                  >
                    <div className="flex items-start gap-3">
                      <input
                        type="checkbox"
                        className="mt-1"
                        checked={selected.has(item.id)}
                        onChange={() => toggleSelect(item.id)}
                        aria-label={t('wallet.selectItem', { title: item.title })}
                      />
                      <div className="min-w-0 flex-1">
                        <h3 className="text-base font-semibold text-slate-900 dark:text-white">{item.title}</h3>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                          {item.issuer ? `${item.issuer} · ` : ''}
                          {item.issuedAt ? formatDate(item.issuedAt) : t('wallet.noDate')}
                          {item.revoked ? ` · ${t('wallet.status.revoked')}` : ''}
                        </p>
                        <p className="mt-2 inline-flex items-center gap-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                          {item.verifyStatus === 'verified' ? <CheckCircle2 className="h-3.5 w-3.5" aria-hidden /> : null}
                          {t(`wallet.status.${item.verifyStatus}`)}
                        </p>
                        <div className="mt-3 flex flex-wrap gap-2 text-sm">
                          {downloadPath ? (
                            <button
                              type="button"
                              className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300"
                              onClick={() => void onDownloadPath(downloadPath, `${item.kind}.bin`)}
                            >
                              {t('wallet.download')}
                            </button>
                          ) : null}
                          {item.verifyUrl ? (
                            <a
                              href={item.verifyUrl}
                              className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300"
                              target="_blank"
                              rel="noreferrer"
                            >
                              {t('wallet.verify')}
                            </a>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  </li>
                )
              })}
            </ul>
          </section>
        ))}
      </div>

      <section className="mt-10" aria-labelledby="wallet-collections-heading">
        <h2 id="wallet-collections-heading" className="mb-3 flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white">
          <Award className="h-5 w-5" aria-hidden />
          {t('wallet.collectionsHeading')}
        </h2>
        {collections.length === 0 ? (
          <p className="text-sm text-slate-600 dark:text-slate-300">{t('wallet.collectionsEmpty')}</p>
        ) : (
          <ul className="space-y-3">
            {collections.map((col) => (
              <li
                key={col.id}
                className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800"
              >
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <h3 className="font-semibold text-slate-900 dark:text-white">{col.name}</h3>
                    <p className="text-xs text-slate-500">
                      {t('wallet.collectionMeta', {
                        count: col.itemIds.length,
                        disclosure: col.disclosure,
                      })}
                      {col.revoked ? ` · ${t('wallet.shareRevoked')}` : ''}
                    </p>
                    {col.shareUrl && !col.revoked ? (
                      <p className="mt-2 break-all text-sm text-indigo-700 dark:text-indigo-300">{col.shareUrl}</p>
                    ) : null}
                  </div>
                  <div className="flex flex-wrap gap-2 text-sm">
                    <button type="button" className="underline" onClick={() => void onShowAccess(col.id)}>
                      {t('wallet.accessHistory')}
                    </button>
                    {!col.revoked && col.shareUrl ? (
                      <button type="button" className="underline" onClick={() => void onRevoke(col.id)}>
                        {t('wallet.revoke')}
                      </button>
                    ) : null}
                    <button type="button" className="underline text-red-700 dark:text-red-300" onClick={() => void onDelete(col.id)}>
                      {t('wallet.delete')}
                    </button>
                  </div>
                </div>
                {accessFor === col.id ? (
                  <ul className="mt-3 space-y-1 border-t border-slate-100 pt-3 text-xs text-slate-600 dark:border-slate-700 dark:text-slate-300">
                    {accessEvents.length === 0 ? <li>{t('wallet.accessEmpty')}</li> : null}
                    {accessEvents.map((ev) => (
                      <li key={ev.id}>
                        {formatDate(ev.createdAt)} — {ev.result}
                        {ev.requesterIp ? ` (${ev.requesterIp})` : ''}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </section>

      <p className="mt-8 text-sm text-slate-500">
        <Link to="/transcripts" className="underline-offset-2 hover:underline">
          {t('wallet.linkTranscripts')}
        </Link>
        {' · '}
        <Link to="/me/ccr" className="underline-offset-2 hover:underline">
          {t('wallet.linkCcr')}
        </Link>
        {' · '}
        <Link to="/me/credentials" className="underline-offset-2 hover:underline">
          {t('wallet.linkCredentials')}
        </Link>
      </p>

      {shareOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={shareDialogTitleId}
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
        >
          <div className="w-full max-w-md rounded-xl bg-white p-5 shadow-lg dark:bg-slate-900">
            <h2 id={shareDialogTitleId} className="text-lg font-semibold text-slate-900 dark:text-white">
              {t('wallet.shareDialogTitle')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              {t('wallet.shareDialogHelp', { count: selected.size || shareResult?.itemIds.length || 0 })}
            </p>
            {shareResult?.shareUrl ? (
              <div className="mt-4 space-y-2">
                <p className="text-sm font-medium text-emerald-700 dark:text-emerald-300">{t('wallet.shareReady')}</p>
                <p className="break-all text-sm text-indigo-700 dark:text-indigo-300">{shareResult.shareUrl}</p>
                <button
                  type="button"
                  className="rounded-md bg-slate-900 px-3 py-2 text-sm text-white dark:bg-slate-100 dark:text-slate-900"
                  onClick={() => {
                    void navigator.clipboard.writeText(shareResult.shareUrl ?? '')
                  }}
                >
                  {t('wallet.copyLink')}
                </button>
              </div>
            ) : (
              <div className="mt-4 space-y-3">
                <label className="block text-sm">
                  <span className="text-slate-700 dark:text-slate-200">{t('wallet.collectionName')}</span>
                  <input
                    className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 dark:border-slate-600 dark:bg-slate-800"
                    value={shareName}
                    onChange={(e) => setShareName(e.target.value)}
                  />
                </label>
                <fieldset>
                  <legend className="text-sm text-slate-700 dark:text-slate-200">{t('wallet.disclosure')}</legend>
                  <p className="mt-1 text-xs text-slate-500">{t('wallet.disclosureHelp')}</p>
                  <div className="mt-2 space-y-1 text-sm">
                    {(['validity', 'summary', 'full'] as const).map((level) => (
                      <label key={level} className="flex items-center gap-2">
                        <input
                          type="radio"
                          name="disclosure"
                          checked={disclosure === level}
                          onChange={() => setDisclosure(level)}
                        />
                        {t(`wallet.disclosure.${level}`)}
                      </label>
                    ))}
                  </div>
                </fieldset>
              </div>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <button
                type="button"
                className="rounded-md px-3 py-2 text-sm text-slate-700 dark:text-slate-200"
                onClick={() => setShareOpen(false)}
              >
                {t('wallet.close')}
              </button>
              {!shareResult ? (
                <button
                  type="button"
                  className="rounded-md bg-slate-900 px-3 py-2 text-sm text-white disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
                  disabled={sharing || selected.size === 0}
                  onClick={() => void onCreateShare()}
                >
                  {sharing ? t('wallet.sharing') : t('wallet.createShare')}
                </button>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}
    </LmsPage>
  )
}
