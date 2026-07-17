import { useCallback, useEffect, useId, useState, type FormEvent } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createDiplomaTemplate,
  fetchDiplomaBatch,
  fetchDiplomaTemplates,
  issueDiploma,
  issueDiplomaBatch,
  type DiplomaBatch,
  type DiplomaKind,
  type DiplomaTemplate,
  type IssuedDiploma,
} from '../../lib/diplomas-api'

export default function AdminDiplomas() {
  const titleId = useId()
  const { ffDiplomas, loading: featuresLoading } = usePlatformFeatures()
  const [templates, setTemplates] = useState<DiplomaTemplate[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const [kind, setKind] = useState<DiplomaKind>('diploma')
  const [name, setName] = useState('')
  const [title, setTitle] = useState('')
  const [program, setProgram] = useState('')

  const [templateId, setTemplateId] = useState('')
  const [userId, setUserId] = useState('')
  const [learnerName, setLearnerName] = useState('')
  const [honors, setHonors] = useState('')
  const [lastIssued, setLastIssued] = useState<IssuedDiploma | null>(null)

  const [batchUserIds, setBatchUserIds] = useState('')
  const [batch, setBatch] = useState<DiplomaBatch | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await fetchDiplomaTemplates()
      setTemplates(list)
      if (!templateId && list.length > 0) {
        setTemplateId(list[0].id)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load templates')
    } finally {
      setLoading(false)
    }
  }, [templateId])

  useEffect(() => {
    if (!featuresLoading && ffDiplomas) {
      void reload()
    }
  }, [featuresLoading, ffDiplomas, reload])

  useEffect(() => {
    if (!batch || batch.status === 'completed' || batch.status === 'failed') return
    const t = window.setInterval(() => {
      void fetchDiplomaBatch(batch.id)
        .then(setBatch)
        .catch(() => undefined)
    }, 2000)
    return () => window.clearInterval(t)
  }, [batch])

  if (featuresLoading) {
    return <p className="p-6 text-sm text-[var(--lx-muted)]">Loading…</p>
  }
  if (!ffDiplomas) {
    return (
      <main className="mx-auto max-w-3xl p-6" aria-labelledby={titleId}>
        <h1 id={titleId} className="text-2xl font-semibold text-[var(--lx-fg)]">
          Diplomas & certificates
        </h1>
        <p className="mt-2 text-sm text-[var(--lx-muted)]">
          Diploma issuance is not enabled. Turn on <code>ffDiplomas</code> in Settings → Global platform.
        </p>
      </main>
    )
  }

  async function onCreateTemplate(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setNotice(null)
    try {
      const tmpl = await createDiplomaTemplate({
        kind,
        name,
        title: title || name,
        program: program || undefined,
      })
      setNotice(`Created template “${tmpl.name}”.`)
      setName('')
      setTitle('')
      setProgram('')
      await reload()
      setTemplateId(tmpl.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    }
  }

  async function onIssue(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setNotice(null)
    try {
      const res = await issueDiploma({
        userId: userId.trim(),
        templateId,
        learnerName: learnerName.trim() || undefined,
        honors: honors.trim() || undefined,
        program: program.trim() || undefined,
      })
      setLastIssued(res.diploma)
      setNotice(res.skipped ? `Already issued (${res.reason ?? 'idempotent'}).` : 'Credential issued.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Issue failed')
    }
  }

  async function onBatch(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setNotice(null)
    const ids = batchUserIds
      .split(/[\s,]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    try {
      const b = await issueDiplomaBatch({
        templateId,
        userIds: ids,
        program: program.trim() || undefined,
        honors: honors.trim() || undefined,
      })
      setBatch(b)
      setNotice('Batch queued.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Batch failed')
    }
  }

  return (
    <main className="mx-auto max-w-4xl space-y-8 p-6" aria-labelledby={titleId}>
      <header>
        <h1 id={titleId} className="text-2xl font-semibold text-[var(--lx-fg)]">
          Diplomas & certificates
        </h1>
        <p className="mt-1 text-sm text-[var(--lx-muted)]">
          Define templates and issue verifiable diplomas or certificates into the learner wallet.
        </p>
      </header>

      {error ? (
        <p className="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800" role="alert">
          {error}
        </p>
      ) : null}
      {notice ? (
        <p className="rounded border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-900" role="status">
          {notice}
        </p>
      ) : null}

      <section aria-labelledby="tmpl-heading" className="space-y-3">
        <h2 id="tmpl-heading" className="text-lg font-medium text-[var(--lx-fg)]">
          Templates
        </h2>
        <form onSubmit={onCreateTemplate} className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm">
            Kind
            <select
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={kind}
              onChange={(e) => setKind(e.target.value as DiplomaKind)}
            >
              <option value="diploma">Diploma</option>
              <option value="certificate">Certificate</option>
            </select>
          </label>
          <label className="text-sm">
            Name
            <input
              required
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <label className="text-sm">
            Credential title
            <input
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Bachelor of Science"
            />
          </label>
          <label className="text-sm">
            Program
            <input
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={program}
              onChange={(e) => setProgram(e.target.value)}
            />
          </label>
          <div className="sm:col-span-2">
            <button
              type="submit"
              className="rounded bg-[var(--lx-accent)] px-3 py-1.5 text-sm font-medium text-white"
            >
              Create template
            </button>
          </div>
        </form>
        {loading ? <p className="text-sm text-[var(--lx-muted)]">Loading templates…</p> : null}
        <ul className="divide-y divide-[var(--lx-border)] rounded border border-[var(--lx-border)]">
          {templates.map((t) => (
            <li key={t.id} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm">
              <div>
                <span className="font-medium">{t.name}</span>
                <span className="ms-2 text-[var(--lx-muted)]">
                  {t.kind} · {t.title}
                  {!t.active ? ' · inactive' : ''}
                </span>
              </div>
              <button
                type="button"
                className="text-[var(--lx-accent)] underline"
                onClick={() => setTemplateId(t.id)}
              >
                Use for issue
              </button>
            </li>
          ))}
          {templates.length === 0 && !loading ? (
            <li className="px-3 py-4 text-sm text-[var(--lx-muted)]">No templates yet.</li>
          ) : null}
        </ul>
      </section>

      <section aria-labelledby="issue-heading" className="space-y-3">
        <h2 id="issue-heading" className="text-lg font-medium text-[var(--lx-fg)]">
          Issue to one learner
        </h2>
        <form onSubmit={onIssue} className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm sm:col-span-2">
            Template
            <select
              required
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
            >
              <option value="" disabled>
                Select template
              </option>
              {templates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} ({t.kind})
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            Learner user ID
            <input
              required
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5 font-mono text-xs"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
            />
          </label>
          <label className="text-sm">
            Learner display name
            <input
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={learnerName}
              onChange={(e) => setLearnerName(e.target.value)}
            />
          </label>
          <label className="text-sm">
            Honors
            <input
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5"
              value={honors}
              onChange={(e) => setHonors(e.target.value)}
            />
          </label>
          <div className="sm:col-span-2">
            <button
              type="submit"
              className="rounded bg-[var(--lx-accent)] px-3 py-1.5 text-sm font-medium text-white"
            >
              Issue
            </button>
          </div>
        </form>
        {lastIssued ? (
          <p className="text-sm text-[var(--lx-muted)]">
            Last: {lastIssued.credentialTitle} v{lastIssued.version}
            {lastIssued.verifyToken ? ` · verify token ${lastIssued.verifyToken}` : ''}
            {lastIssued.revokedAt ? ' · revoked' : ''}
          </p>
        ) : null}
      </section>

      <section aria-labelledby="batch-heading" className="space-y-3">
        <h2 id="batch-heading" className="text-lg font-medium text-[var(--lx-fg)]">
          Batch issue
        </h2>
        <form onSubmit={onBatch} className="space-y-3">
          <label className="block text-sm">
            Learner user IDs (comma or whitespace separated)
            <textarea
              required
              rows={3}
              className="mt-1 w-full rounded border border-[var(--lx-border)] bg-[var(--lx-bg)] px-2 py-1.5 font-mono text-xs"
              value={batchUserIds}
              onChange={(e) => setBatchUserIds(e.target.value)}
            />
          </label>
          <button
            type="submit"
            className="rounded bg-[var(--lx-accent)] px-3 py-1.5 text-sm font-medium text-white"
            disabled={!templateId}
          >
            Start batch
          </button>
        </form>
        {batch ? (
          <p className="text-sm text-[var(--lx-muted)]" role="status">
            Batch {batch.id}: {batch.status} — {batch.successCount} issued, {batch.skipCount} skipped,{' '}
            {batch.failCount} failed / {batch.totalCount}
            {batch.errorSummary ? ` · ${batch.errorSummary}` : ''}
          </p>
        ) : null}
      </section>
    </main>
  )
}
