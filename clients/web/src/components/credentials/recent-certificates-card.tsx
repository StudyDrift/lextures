import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Award, ArrowRight } from 'lucide-react'
import { fetchMyCredentials, type IssuedCredentialSummary } from '../../lib/credentials-api'

export function RecentCertificatesCard() {
  const [items, setItems] = useState<IssuedCredentialSummary[]>([])

  useEffect(() => {
    void fetchMyCredentials()
      .then((data) => setItems(data.credentials.slice(0, 3)))
      .catch(() => setItems([]))
  }, [])

  if (items.length === 0) return null

  return (
    <section aria-label="Recent certificates">
      <div className="rounded-2xl border border-emerald-100 bg-emerald-50/80 px-5 py-4 dark:border-emerald-900/40 dark:bg-emerald-950/30">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Recent certificates</p>
            <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
              Download PDFs or share verification links with employers.
            </p>
          </div>
          <Link
            to="/me/credentials"
            className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-700"
          >
            My credentials
            <ArrowRight className="h-4 w-4" aria-hidden />
          </Link>
        </div>
        <ul className="mt-4 space-y-2">
          {items.map((item) => (
            <li key={item.id}>
              <Link
                to="/me/credentials"
                className="flex items-center gap-2 rounded-lg bg-white/80 px-3 py-2 text-sm text-slate-800 hover:bg-white dark:bg-neutral-900/60 dark:text-neutral-100 dark:hover:bg-neutral-900"
              >
                <Award className="h-4 w-4 shrink-0 text-emerald-600" aria-hidden />
                <span className="truncate font-medium">{item.title}</span>
                <span className="ml-auto shrink-0 text-xs text-slate-500">
                  {new Date(item.issuedAt).toLocaleDateString()}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}