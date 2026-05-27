import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authorizedFetch } from '../lib/api'

const API = '/api/v1/compliance/iso'

interface Dashboard {
  program: {
    scopeStatement: string
    iso27001Status: string
    iso27701Status: string
    soa: { total: number; implemented: number; planned: number; excluded: number }
  }
  openFindings: number
  highRisks: number
  pendingSuppliers: number
  trainingYear: number
  trainingCount: number
}

interface AuditFinding {
  id: string
  auditCycle: string
  findingType: string
  isoClause: string
  description: string
  status: string
}

interface Risk {
  id: string
  riskTitle: string
  likelihood: number
  impact: number
  treatment: string
  residualScore: number
}

export default function IsoComplianceAdminPage() {
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
  const [findings, setFindings] = useState<AuditFinding[]>([])
  const [risks, setRisks] = useState<Risk[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [dashRes, findRes, riskRes] = await Promise.all([
        authorizedFetch(`${API}/dashboard`),
        authorizedFetch(`${API}/audit-findings`),
        authorizedFetch(`${API}/risk-register`),
      ])
      if (dashRes.status === 404) {
        setError('ISO ISMS module is not enabled on this environment.')
        return
      }
      if (dashRes.status === 403) {
        setError('You need Global Admin / compliance:iso:admin permission.')
        return
      }
      if (!dashRes.ok) {
        setError('Failed to load ISO compliance dashboard.')
        return
      }
      const dash = (await dashRes.json()) as Dashboard
      setDashboard(dash)
      if (findRes.ok) {
        const f = (await findRes.json()) as { findings: AuditFinding[] }
        setFindings(f.findings ?? [])
      }
      if (riskRes.ok) {
        const r = (await riskRes.json()) as { risks: Risk[] }
        setRisks(r.risks ?? [])
      }
    } catch {
      setError('Network error loading ISO compliance data.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    document.title = 'ISO ISMS — Lextures'
    void load()
  }, [load])

  async function addFinding(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const res = await authorizedFetch(`${API}/audit-findings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        auditCycle: form.get('auditCycle'),
        findingType: form.get('findingType'),
        isoClause: form.get('isoClause'),
        description: form.get('description'),
      }),
    })
    if (res.ok) {
      setMessage('Audit finding recorded.')
      e.currentTarget.reset()
      void load()
    } else {
      setMessage('Could not create finding.')
    }
  }

  async function addRisk(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const res = await authorizedFetch(`${API}/risk-register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        riskTitle: form.get('riskTitle'),
        likelihood: Number(form.get('likelihood')),
        impact: Number(form.get('impact')),
        treatment: form.get('treatment'),
      }),
    })
    if (res.ok) {
      setMessage('Risk entry added.')
      e.currentTarget.reset()
      void load()
    } else {
      setMessage('Could not add risk.')
    }
  }

  if (loading) {
    return (
      <div className="p-6 max-w-5xl mx-auto">
        <p className="text-slate-600 dark:text-neutral-400">Loading ISO ISMS dashboard…</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6 max-w-5xl mx-auto">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">ISO 27001 / 27701</h1>
        <p role="alert" className="mt-4 text-red-600 dark:text-red-400">{error}</p>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
          See <Link to="/trust" className="text-indigo-700 underline dark:text-indigo-300">Trust Center</Link> for public program status.
        </p>
      </div>
    )
  }

  const soa = dashboard?.program.soa

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-8">
      <header>
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">ISO 27001 / 27701 ISMS</h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Audit findings, risk register, and Statement of Applicability. Documentation in{' '}
          <a href="https://github.com/lextures/lextures/tree/main/docs/isms" className="text-indigo-700 underline dark:text-indigo-300">
            docs/isms
          </a>
          .
        </p>
      </header>

      {message ? (
        <p role="status" className="text-sm text-emerald-700 dark:text-emerald-300">{message}</p>
      ) : null}

      {dashboard ? (
        <section aria-labelledby="iso-metrics" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <h2 id="iso-metrics" className="sr-only">Program metrics</h2>
          <MetricCard label="Open findings" value={dashboard.openFindings} />
          <MetricCard label="High risks (≥15)" value={dashboard.highRisks} />
          <MetricCard label="Pending suppliers" value={dashboard.pendingSuppliers} />
          <MetricCard label={`Training (${dashboard.trainingYear})`} value={dashboard.trainingCount} />
        </section>
      ) : null}

      {soa ? (
        <section aria-labelledby="soa-heading" className="rounded-lg border border-slate-200 dark:border-neutral-800 p-4">
          <h2 id="soa-heading" className="font-semibold text-slate-900 dark:text-neutral-50">Statement of Applicability</h2>
          <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
            {soa.implemented} implemented · {soa.planned} planned · {soa.excluded} excluded · {soa.total} total (Annex A 2022)
          </p>
          <p className="mt-1 text-sm text-slate-500">
            ISO 27001: <span className="font-medium">{dashboard?.program.iso27001Status}</span>
            {' · '}
            ISO 27701: <span className="font-medium">{dashboard?.program.iso27701Status}</span>
          </p>
        </section>
      ) : null}

      <section aria-labelledby="findings-heading" className="space-y-3">
        <h2 id="findings-heading" className="font-semibold text-slate-900 dark:text-neutral-50">Audit findings</h2>
        <form onSubmit={addFinding} className="flex flex-wrap gap-2 text-sm">
          <input name="auditCycle" required placeholder="Audit cycle" className="rounded border px-2 py-1 dark:bg-neutral-900" />
          <select name="findingType" required className="rounded border px-2 py-1 dark:bg-neutral-900">
            <option value="observation">Observation</option>
            <option value="nonconformity">Nonconformity</option>
            <option value="opportunity">Opportunity</option>
          </select>
          <input name="isoClause" required placeholder="Clause (e.g. A.8.15)" className="rounded border px-2 py-1 dark:bg-neutral-900" />
          <input name="description" required placeholder="Description" className="min-w-[12rem] flex-1 rounded border px-2 py-1 dark:bg-neutral-900" />
          <button type="submit" className="rounded bg-indigo-600 px-3 py-1 text-white">Add</button>
        </form>
        <ul className="divide-y divide-slate-100 dark:divide-neutral-800 text-sm">
          {findings.length === 0 ? (
            <li className="py-2 text-slate-500">No findings yet.</li>
          ) : (
            findings.map((f) => (
              <li key={f.id} className="py-2">
                <span className="font-medium">{f.isoClause}</span> — {f.description}
                <span className="ml-2 text-slate-500">({f.status}, {f.auditCycle})</span>
              </li>
            ))
          )}
        </ul>
      </section>

      <section aria-labelledby="risks-heading" className="space-y-3">
        <h2 id="risks-heading" className="font-semibold text-slate-900 dark:text-neutral-50">Risk register</h2>
        <form onSubmit={addRisk} className="flex flex-wrap gap-2 text-sm">
          <input name="riskTitle" required placeholder="Risk title" className="min-w-[10rem] flex-1 rounded border px-2 py-1 dark:bg-neutral-900" />
          <input name="likelihood" type="number" min={1} max={5} defaultValue={3} required className="w-16 rounded border px-2 py-1 dark:bg-neutral-900" aria-label="Likelihood" />
          <input name="impact" type="number" min={1} max={5} defaultValue={3} required className="w-16 rounded border px-2 py-1 dark:bg-neutral-900" aria-label="Impact" />
          <select name="treatment" required className="rounded border px-2 py-1 dark:bg-neutral-900">
            <option value="mitigate">Mitigate</option>
            <option value="accept">Accept</option>
            <option value="transfer">Transfer</option>
            <option value="avoid">Avoid</option>
          </select>
          <button type="submit" className="rounded bg-indigo-600 px-3 py-1 text-white">Add</button>
        </form>
        <ul className="divide-y divide-slate-100 dark:divide-neutral-800 text-sm">
          {risks.length === 0 ? (
            <li className="py-2 text-slate-500">No risks recorded.</li>
          ) : (
            risks.map((r) => (
              <li key={r.id} className="py-2">
                {r.riskTitle}
                <span className="ml-2 text-slate-500">
                  score {r.residualScore} · {r.treatment}
                </span>
              </li>
            ))
          )}
        </ul>
      </section>
    </div>
  )
}

function MetricCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-slate-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-slate-900 dark:text-neutral-50">{value}</p>
    </div>
  )
}
