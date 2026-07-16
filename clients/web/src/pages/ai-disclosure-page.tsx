import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { aiDisclosureI18n } from '../lib/ai-disclosure-i18n'
import { MARKETING_LEGAL_URLS } from '../lib/marketing-site'

type DisclosureDoc = {
  version: string
  provider: string
  providers?: string[]
  models: Array<{
    id: string
    name: string
    provider: string
    purposes: string[]
    dataSent: string
    retentionDays: number
    dpaStatus: string
    optOutPath: string
  }>
  features: Array<{ key: string; label: string; description: string }>
}

export default function AiDisclosurePage() {
  const [doc, setDoc] = useState<DisclosureDoc | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await fetch('/api/v1/public/ai-disclosure')
        if (!res.ok) {
          throw new Error('Failed to load disclosure')
        }
        const data = (await res.json()) as DisclosureDoc
        if (!cancelled) setDoc(data)
      } catch {
        if (!cancelled) setError('Could not load AI disclosure document.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <main className="mx-auto max-w-3xl px-4 py-10 text-slate-800 dark:text-neutral-100">
      <nav className="text-sm text-slate-500 dark:text-neutral-400">
        <a href={MARKETING_LEGAL_URLS.privacy} className="underline-offset-2 hover:underline">
          Privacy
        </a>
        <span aria-hidden="true"> / </span>
        <span>{aiDisclosureI18n.pageTitle}</span>
      </nav>
      <h1 className="mt-4 text-2xl font-semibold">{aiDisclosureI18n.pageTitle}</h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-300">{aiDisclosureI18n.pageIntro}</p>
      {error && (
        <p className="mt-6 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">
          {error}
        </p>
      )}
      {doc && (
        <div className="mt-8 space-y-8">
          <p className="text-xs text-slate-500">Version {doc.version}</p>
          {(doc.providers?.length ?? 0) > 0 && (
            <p className="text-sm text-slate-600 dark:text-neutral-300">
              Configured providers: {doc.providers!.join(', ')}
            </p>
          )}
          {doc.models.map((m) => (
            <article
              key={m.id}
              className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-600 dark:bg-neutral-900"
            >
              <h2 className="text-lg font-semibold">{m.name}</h2>
              <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{m.provider}</p>
              <dl className="mt-4 grid gap-2 text-sm">
                <div>
                  <dt className="font-medium">Purposes</dt>
                  <dd>{m.purposes.join(', ')}</dd>
                </div>
                <div>
                  <dt className="font-medium">Data sent</dt>
                  <dd>{m.dataSent}</dd>
                </div>
                <div>
                  <dt className="font-medium">Retention</dt>
                  <dd>{m.retentionDays} days (provider policy)</dd>
                </div>
                <div>
                  <dt className="font-medium">Opt out</dt>
                  <dd>
                    <Link to="/settings/account" className="text-indigo-700 underline dark:text-indigo-300">
                      Account AI settings
                    </Link>
                  </dd>
                </div>
              </dl>
            </article>
          ))}
        </div>
      )}
    </main>
  )
}
