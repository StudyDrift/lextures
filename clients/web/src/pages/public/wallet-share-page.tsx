import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { fetchPublicWalletShare, type PublicWalletShare } from '../../lib/wallet-api'
import { formatDate } from '../../lib/format'

export default function WalletSharePage() {
  const { t } = useTranslation('common')
  const { token } = useParams<{ token: string }>()
  const [data, setData] = useState<PublicWalletShare | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!token) {
      setError(t('wallet.public.missingToken'))
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    void fetchPublicWalletShare(token)
      .then((share) => {
        if (!cancelled) {
          setData(share)
          document.title = share.name || t('wallet.public.pageTitle')
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : t('wallet.public.loadError'))
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [token, t])

  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-100 to-slate-200 px-4 py-16 text-slate-900 dark:from-neutral-950 dark:to-neutral-900 dark:text-neutral-50">
      <div className="mx-auto max-w-lg">
        <p className="text-sm font-semibold tracking-wide text-emerald-700 dark:text-emerald-300">Lextures</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">
          {data?.name || t('wallet.public.heading')}
        </h1>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">{t('wallet.public.help')}</p>

        {loading ? <p className="mt-8 text-sm text-slate-500">{t('common.loading')}</p> : null}
        {error ? (
          <p
            className="mt-8 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-100"
            role="alert"
          >
            {error}
          </p>
        ) : null}

        {data && !loading ? (
          <section className="mt-8 space-y-3" aria-labelledby="wallet-share-items">
            <h2 id="wallet-share-items" className="sr-only">
              {t('wallet.public.itemsHeading')}
            </h2>
            <p className="text-xs uppercase tracking-wide text-slate-500">
              {t('wallet.public.disclosure', { level: data.disclosure })}
            </p>
            <ul className="space-y-3">
              {data.items.map((item, idx) => (
                <li
                  key={`${item.kind}-${idx}`}
                  className="rounded-lg border border-slate-200 bg-white/80 p-4 dark:border-neutral-700 dark:bg-neutral-900/80"
                >
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{item.kind}</p>
                  {item.title ? (
                    <p className="mt-1 text-base font-semibold">{item.title}</p>
                  ) : (
                    <p className="mt-1 text-base font-semibold">{t('wallet.public.credential')}</p>
                  )}
                  <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
                    {item.issuer ? `${item.issuer}` : null}
                    {item.issuedAt ? ` · ${formatDate(item.issuedAt)}` : null}
                    {item.revoked
                      ? ` · ${t('wallet.status.revoked')}`
                      : item.valid
                        ? ` · ${t('wallet.status.verified')}`
                        : ''}
                  </p>
                  {item.verifyUrl ? (
                    <a
                      href={item.verifyUrl}
                      className="mt-2 inline-block text-sm text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300"
                      target="_blank"
                      rel="noreferrer"
                    >
                      {t('wallet.verify')}
                    </a>
                  ) : null}
                </li>
              ))}
            </ul>
            <p className="pt-4 text-sm">
              <Link to="/verify" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">
                {t('wallet.public.verifyPortal')}
              </Link>
            </p>
          </section>
        ) : null}
      </div>
    </main>
  )
}
