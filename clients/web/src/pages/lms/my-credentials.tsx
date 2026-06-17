import { useCallback, useEffect, useState } from 'react'
import { Award, Copy, Download, ExternalLink } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  downloadCredentialPDF,
  fetchMyCredentials,
  type IssuedCredential,
} from '../../lib/credentials-api'

function CredentialCard({ item }: { item: IssuedCredential }) {
  const [busy, setBusy] = useState(false)
  const [copied, setCopied] = useState(false)

  const onDownload = useCallback(async () => {
    setBusy(true)
    try {
      const blob = await downloadCredentialPDF(item.id)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `certificate-${item.id}.pdf`
      a.click()
      URL.revokeObjectURL(url)
    } finally {
      setBusy(false)
    }
  }, [item.id])

  const onCopy = useCallback(async () => {
    await navigator.clipboard.writeText(item.verificationUrl)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }, [item.verificationUrl])

  return (
    <article
      aria-label={`Certificate: ${item.title}`}
      className="flex flex-col rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900"
    >
      <div className="flex items-start gap-3">
        <Award className="mt-0.5 h-8 w-8 shrink-0 text-emerald-600" aria-hidden />
        <div className="min-w-0 flex-1">
          <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">{item.title}</h2>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Issued {new Date(item.issuedAt).toLocaleDateString()}
            {item.revoked ? ' · Revoked' : ''}
          </p>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          disabled={busy || item.revoked}
          onClick={() => void onDownload()}
          className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          <Download className="h-3.5 w-3.5" aria-hidden />
          Download PDF
        </button>
        <a
          href={item.verificationUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
        >
          <ExternalLink className="h-3.5 w-3.5" aria-hidden />
          Verify
        </a>
        <button
          type="button"
          onClick={() => void onCopy()}
          className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
        >
          <Copy className="h-3.5 w-3.5" aria-hidden />
          {copied ? 'Copied' : 'Copy link'}
        </button>
      </div>
    </article>
  )
}

export default function MyCredentialsPage() {
  const { ffCompletionCredentials, loading: featuresLoading } = usePlatformFeatures()
  const [items, setItems] = useState<IssuedCredential[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (featuresLoading || !ffCompletionCredentials) {
      setLoading(false)
      return
    }
    void fetchMyCredentials()
      .then((data) => setItems(data.credentials))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load credentials.'))
      .finally(() => setLoading(false))
  }, [ffCompletionCredentials, featuresLoading])

  if (!ffCompletionCredentials && !featuresLoading) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10">
        <p className="text-sm text-slate-600 dark:text-neutral-400">Completion certificates are not enabled on this platform.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">My credentials</h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Verifiable Open Badges certificates you have earned on Lextures.
      </p>

      {loading ? <p className="mt-8 text-sm text-slate-600">Loading…</p> : null}
      {error ? (
        <p role="alert" className="mt-8 text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}

      {!loading && !error && items.length === 0 ? (
        <p className="mt-8 rounded-2xl border border-dashed border-slate-200 px-6 py-10 text-center text-sm text-slate-600 dark:border-neutral-700 dark:text-neutral-400">
          Complete a self-paced course or learning path to earn your first certificate.
        </p>
      ) : null}

      <div className="mt-8 grid gap-4 sm:grid-cols-2">
        {items.map((item) => (
          <CredentialCard key={item.id} item={item} />
        ))}
      </div>
    </div>
  )
}