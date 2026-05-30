import { useEffect, useState } from 'react'
import {
  fetchBehaviorDashboard,
  listBehaviorCategories,
  seedDefaultBehaviorCategories,
  type BehaviorDashboardResponse,
  type BehaviorCategory,
} from '../../lib/behavior-api'
import { authorizedFetch } from '../../lib/api'

function thisMonday(): string {
  const d = new Date()
  const day = d.getDay() || 7
  d.setDate(d.getDate() - (day - 1))
  return d.toISOString().slice(0, 10)
}

export default function BehaviorDashboard() {
  const [orgId, setOrgId] = useState<string>('')
  const [weekStart, setWeekStart] = useState<string>(thisMonday())
  const [dashboard, setDashboard] = useState<BehaviorDashboardResponse | null>(null)
  const [categories, setCategories] = useState<BehaviorCategory[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [seedMsg, setSeedMsg] = useState<string | null>(null)

  useEffect(() => {
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/me')
        if (!res.ok) return
        const me = (await res.json()) as { orgId?: string }
        if (me.orgId) setOrgId(me.orgId)
      } catch {
        // ignore
      }
    })()
  }, [])

  useEffect(() => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    void (async () => {
      try {
        const [dash, cats] = await Promise.all([
          fetchBehaviorDashboard(orgId, weekStart),
          listBehaviorCategories(orgId),
        ])
        setDashboard(dash)
        setCategories(cats.categories)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load dashboard.')
      } finally {
        setLoading(false)
      }
    })()
  }, [orgId, weekStart])

  async function handleSeedCategories() {
    if (!orgId) return
    setSeedMsg(null)
    try {
      const data = await seedDefaultBehaviorCategories(orgId)
      setCategories(data.categories)
      setSeedMsg(`Seeded ${data.categories.length} default categories.`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to seed categories.')
    }
  }

  const weekEnd = weekStart
    ? new Date(new Date(weekStart).getTime() + 6 * 86400000).toISOString().slice(0, 10)
    : ''

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <h1 className="text-2xl font-semibold mb-6">PBIS Behavior Dashboard</h1>

      <div className="flex flex-wrap gap-3 mb-6 items-end">
        <div>
          <label htmlFor="week-start" className="block text-sm font-medium mb-1">
            Week Starting (Monday)
          </label>
          <input
            id="week-start"
            type="date"
            value={weekStart}
            onChange={(e) => setWeekStart(e.target.value)}
            className="border rounded px-3 py-2 text-sm"
          />
        </div>
        {weekEnd && (
          <span className="text-sm text-gray-500 self-center">through {weekEnd}</span>
        )}
      </div>

      {error && (
        <p role="alert" className="text-red-600 text-sm mb-4">
          {error}
        </p>
      )}

      {loading && <p className="text-gray-500 text-sm">Loading…</p>}

      {!loading && dashboard && (
        <>
          <div className="grid grid-cols-2 gap-4 mb-8">
            <div className="border rounded p-4 bg-green-50">
              <div className="text-3xl font-bold text-green-700">{dashboard.totalPoints}</div>
              <div className="text-sm text-green-600 mt-1">Total Points Awarded</div>
            </div>
            <div className="border rounded p-4 bg-red-50">
              <div className="text-3xl font-bold text-red-700">{dashboard.totalReferrals}</div>
              <div className="text-sm text-red-600 mt-1">Total Referrals Filed</div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            <section aria-label="Points by category">
              <h2 className="text-base font-semibold mb-3">Top Point Categories</h2>
              {dashboard.pointsByCategory.length === 0 ? (
                <p className="text-gray-500 text-sm">No points awarded this week.</p>
              ) : (
                <ul className="divide-y border rounded">
                  {dashboard.pointsByCategory.map((p) => (
                    <li key={p.categoryId} className="flex justify-between px-3 py-2 text-sm">
                      <span>{p.categoryName}</span>
                      <span className="font-semibold text-green-700">{p.points} pts</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section aria-label="Referrals by category">
              <h2 className="text-base font-semibold mb-3">Referrals by Category</h2>
              {dashboard.referralsByCategory.length === 0 ? (
                <p className="text-gray-500 text-sm">No referrals filed this week.</p>
              ) : (
                <ul className="divide-y border rounded">
                  {dashboard.referralsByCategory.map((r) => (
                    <li key={r.categoryId} className="flex justify-between px-3 py-2 text-sm">
                      <span>{r.categoryName}</span>
                      <span className="font-semibold text-red-700">{r.count}</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </div>
        </>
      )}

      <section aria-label="Behavior categories management" className="border-t pt-6">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-base font-semibold">
            Behavior Categories ({categories.length})
          </h2>
          <button
            onClick={handleSeedCategories}
            className="text-sm px-3 py-1.5 border rounded hover:bg-gray-50 cursor-pointer"
          >
            Seed Defaults
          </button>
        </div>

        {seedMsg && (
          <p className="text-green-700 text-sm mb-3">{seedMsg}</p>
        )}

        {categories.length === 0 ? (
          <p className="text-gray-500 text-sm">
            No categories yet. Click "Seed Defaults" to add the standard PBIS categories.
          </p>
        ) : (
          <ul className="divide-y border rounded">
            {categories.map((c) => (
              <li key={c.id} className="flex items-center gap-3 px-3 py-2 text-sm">
                {c.color && (
                  <span
                    className="w-3 h-3 rounded-full flex-shrink-0"
                    style={{ backgroundColor: c.color }}
                    aria-hidden="true"
                  />
                )}
                <span className="flex-1">{c.name}</span>
                <span
                  className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                    c.type === 'positive'
                      ? 'bg-green-100 text-green-700'
                      : 'bg-red-100 text-red-700'
                  }`}
                >
                  {c.type}
                </span>
                {!c.active && (
                  <span className="text-xs text-gray-400">(inactive)</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
