import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Route } from 'lucide-react'
import {
  fetchCatalogPaths,
  formatCents,
  formatDurationMinutes,
  type LearningPathSummary,
} from '../../lib/learning-paths-api'

export default function PathsCatalogPage() {
  const [paths, setPaths] = useState<LearningPathSummary[]>([])
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    setLoading(true)
    fetchCatalogPaths(q)
      .then((rows) => active && setPaths(rows))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [q])

  return (
    <main className="mx-auto max-w-3xl px-4 py-10">
      <header className="mb-8">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <Route className="size-6" aria-hidden />
          Learning paths
        </h1>
        <p className="mt-2 text-muted-foreground">Curated course sequences to build job-ready skills.</p>
        <label className="mt-4 block max-w-md text-sm">
          <span className="font-medium">Search</span>
          <input
            className="mt-1 w-full rounded-md border bg-background px-3 py-2"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Machine learning, Python…"
          />
        </label>
      </header>
      {loading ? (
        <div className="h-32 motion-safe:animate-pulse rounded-lg border bg-card" aria-hidden />
      ) : paths.length === 0 ? (
        <p className="text-sm text-muted-foreground">No public learning paths yet.</p>
      ) : (
        <ul className="space-y-4">
          {paths.map((path) => (
            <li key={path.id} className="rounded-lg border bg-card p-4">
              <Link to={`/paths/${path.slug}`} className="font-semibold hover:underline">
                {path.title}
              </Link>
              <p className="mt-1 text-sm text-muted-foreground line-clamp-2">{path.description}</p>
              <p className="mt-2 text-sm text-muted-foreground">
                {path.courseCount} courses · {formatDurationMinutes(path.totalDurationMinutes)}
                {path.bundlePriceCents != null ? ` · ${formatCents(path.bundlePriceCents)}` : ' · Free'}
              </p>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}
