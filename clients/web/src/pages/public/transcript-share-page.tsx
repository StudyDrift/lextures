import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  downloadTranscriptShareLink,
  fetchTranscriptShareLink,
  type TranscriptShareLinkMeta,
} from '../../lib/transcripts-api'
import { formatDate } from '../../lib/format'

export default function TranscriptSharePage() {
  const { t } = useTranslation('common')
  const { token } = useParams<{ token: string }>()
  const [meta, setMeta] = useState<TranscriptShareLinkMeta | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [downloading, setDownloading] = useState(false)

  useEffect(() => {
    if (!token) {
      setError(t('transcripts.share.missingToken'))
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    void fetchTranscriptShareLink(token)
      .then((m) => {
        if (!cancelled) {
          setMeta(m)
          document.title = t('transcripts.share.pageTitle')
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : t('transcripts.share.loadError'))
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [token, t])

  async function onDownload() {
    if (!token || !meta || meta.expired || meta.exhausted) return
    setDownloading(true)
    setError(null)
    try {
      await downloadTranscriptShareLink(token)
      const refreshed = await fetchTranscriptShareLink(token)
      setMeta(refreshed)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('transcripts.share.downloadError'))
    } finally {
      setDownloading(false)
    }
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-100 to-slate-200 px-4 py-16 text-fg-default dark:from-neutral-950 dark:to-neutral-900">
      <div className="mx-auto max-w-lg">
        <p className="text-sm font-semibold tracking-wide text-accent-fg">Lextures</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">{t('transcripts.share.heading')}</h1>
        <p className="mt-2 text-sm text-fg-muted">{t('transcripts.share.help')}</p>

        {loading ? <p className="mt-8 text-sm text-fg-muted">{t('common.loading')}</p> : null}
        {error ? (
          <p className="mt-8 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-100" role="alert">
            {error}
          </p>
        ) : null}

        {meta && !loading ? (
          <section className="mt-8 space-y-4" aria-labelledby="share-meta-heading">
            <h2 id="share-meta-heading" className="sr-only">
              {t('transcripts.share.metaHeading')}
            </h2>
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between gap-4">
                <dt className="text-fg-muted">{t('transcripts.share.expires')}</dt>
                <dd>{formatDate(meta.expiresAt, { dateStyle: 'medium', timeStyle: 'short' })}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-fg-muted">{t('transcripts.share.downloadsLeft')}</dt>
                <dd>{meta.downloadsRemaining}</dd>
              </div>
            </dl>

            {meta.expired ? (
              <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100" role="status">
                {t('transcripts.share.expired')}
              </p>
            ) : null}
            {meta.exhausted && !meta.expired ? (
              <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100" role="status">
                {t('transcripts.share.exhausted')}
              </p>
            ) : null}

            <button
              type="button"
              onClick={() => void onDownload()}
              disabled={downloading || meta.expired || meta.exhausted}
              className="w-full rounded-md bg-accent-solid px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
            >
              {downloading ? t('transcripts.share.downloading') : t('transcripts.share.download')}
            </button>

            {meta.verificationUrl || meta.verifyToken ? (
              <a
                href={meta.verificationUrl ?? `/verify/${encodeURIComponent(meta.verifyToken!)}`}
                className="block w-full rounded-md border border-border-strong px-4 py-2.5 text-center text-sm font-semibold text-fg-default hover:bg-surface-base dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
              >
                {t('transcripts.share.verify')}
              </a>
            ) : null}
          </section>
        ) : null}
      </div>
    </main>
  )
}
