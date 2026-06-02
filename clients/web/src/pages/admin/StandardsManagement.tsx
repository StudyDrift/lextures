import { useCallback, useEffect, useRef, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import {
  listStandardDomains,
  createStandardDomain,
  getMasteryScale,
  putMasteryScale,
  importStandardsCSV,
  type StandardDomain,
  type MasteryScaleEntry,
  type CSVImportResult,
} from '../../lib/sbg-api'

export default function StandardsManagement() {
  const [orgId, setOrgId] = useState<string | null>(null)
  const [domains, setDomains] = useState<StandardDomain[]>([])
  const [scale, setScale] = useState<MasteryScaleEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  // New domain form
  const [newCode, setNewCode] = useState('')
  const [newName, setNewName] = useState('')
  const [newGrade, setNewGrade] = useState('')
  const [domainSaving, setDomainSaving] = useState(false)

  // Mastery scale editor
  const [editingScale, setEditingScale] = useState<Array<{ label: string; value: number; color: string }>>([])
  const [scaleSaving, setScaleSaving] = useState(false)

  // CSV import
  const [csvFile, setCsvFile] = useState<File | null>(null)
  const [importResult, setImportResult] = useState<CSVImportResult | null>(null)
  const [importing, setImporting] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const announceRef = useRef<HTMLDivElement>(null)

  function announce(msg: string) {
    if (announceRef.current) announceRef.current.textContent = msg
  }

  // Load org ID
  useEffect(() => {
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/me')
        if (!res.ok) return
        const me = (await res.json()) as { orgId?: string }
        if (me.orgId) setOrgId(me.orgId)
      } catch { /* ignore */ }
    })()
  }, [])

  const loadData = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const [ds, sc] = await Promise.all([
        listStandardDomains(orgId),
        getMasteryScale(orgId),
      ])
      setDomains(ds)
      setScale(sc)
      setEditingScale(sc.map((s) => ({ label: s.label, value: s.value, color: s.color ?? '#6b7280' })))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load data.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void loadData()
  }, [loadData])

  async function handleCreateDomain(e: React.FormEvent) {
    e.preventDefault()
    if (!orgId || !newCode.trim() || !newName.trim()) return
    setDomainSaving(true)
    setError(null)
    try {
      await createStandardDomain(orgId, newCode.trim(), newName.trim(), newGrade.trim() || undefined)
      setNewCode('')
      setNewName('')
      setNewGrade('')
      announce('Domain created.')
      setSuccessMsg('Domain created successfully.')
      await loadData()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create domain.')
    } finally {
      setDomainSaving(false)
    }
  }

  async function handleSaveScale(e: React.FormEvent) {
    e.preventDefault()
    if (!orgId || editingScale.length === 0) return
    setScaleSaving(true)
    setError(null)
    try {
      const saved = await putMasteryScale(
        orgId,
        editingScale.map((s) => ({ label: s.label, value: s.value, color: s.color })),
      )
      setScale(saved)
      announce('Mastery scale saved.')
      setSuccessMsg('Mastery scale saved.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save mastery scale.')
    } finally {
      setScaleSaving(false)
    }
  }

  function updateScaleEntry(idx: number, field: 'label' | 'value' | 'color', val: string) {
    setEditingScale((prev) =>
      prev.map((s, i) =>
        i === idx
          ? { ...s, [field]: field === 'value' ? parseInt(val, 10) || s.value : val }
          : s,
      ),
    )
  }

  function addScaleLevel() {
    const maxVal = editingScale.reduce((m, s) => Math.max(m, s.value), 0)
    setEditingScale((prev) => [...prev, { label: 'New Level', value: maxVal + 1, color: '#6b7280' }])
  }

  function removeScaleLevel(idx: number) {
    setEditingScale((prev) => prev.filter((_, i) => i !== idx))
  }

  async function handleImportCSV(e: React.FormEvent) {
    e.preventDefault()
    if (!orgId || !csvFile) return
    setImporting(true)
    setError(null)
    setImportResult(null)
    try {
      const result = await importStandardsCSV(orgId, csvFile)
      setImportResult(result)
      if (result.errors.length === 0) {
        announce(`Import complete: ${result.standardsImported} standards imported.`)
        setSuccessMsg(`Imported ${result.standardsImported} standards across ${result.domainsCreated} domains.`)
      } else {
        setError(`Import completed with ${result.errors.length} error(s). See details below.`)
      }
      setCsvFile(null)
      if (fileInputRef.current) fileInputRef.current.value = ''
      await loadData()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Import failed.')
    } finally {
      setImporting(false)
    }
  }

  return (
    <div style={{ padding: '24px', maxWidth: '960px', margin: '0 auto' }}>
      <div ref={announceRef} role="status" aria-live="polite" style={{ position: 'absolute', left: '-9999px' }} />

      <h1 style={{ fontSize: '1.5rem', fontWeight: 700, marginBottom: '4px' }}>
        Standards Management
      </h1>
      <p style={{ color: '#6b7280', marginBottom: '24px', fontSize: '0.875rem' }}>
        Manage learning standards and mastery scales for standards-based grading (SBG).
      </p>

      {error && (
        <p role="alert" style={{ color: '#dc2626', background: '#fef2f2', padding: '10px', borderRadius: '6px', marginBottom: '16px' }}>
          {error}
        </p>
      )}
      {successMsg && (
        <p role="status" style={{ color: '#15803d', background: '#f0fdf4', padding: '10px', borderRadius: '6px', marginBottom: '16px' }}>
          {successMsg}
        </p>
      )}

      {/* ── Mastery Scale ──────────────────────────────────────────────────── */}
      <section aria-labelledby="scale-heading" style={{ marginBottom: '32px' }}>
        <h2 id="scale-heading" style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '12px' }}>
          Mastery Scale
        </h2>
        <p style={{ color: '#6b7280', fontSize: '0.85rem', marginBottom: '12px' }}>
          Define the mastery levels used on report cards. The default is a 4-level scale (Exceeds→Below).
        </p>
        <form onSubmit={(e) => { void handleSaveScale(e) }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '12px' }}>
            <thead>
              <tr style={{ background: '#f9fafb' }}>
                {['Value', 'Label', 'Color', ''].map((h) => (
                  <th key={h} style={{ padding: '8px 12px', textAlign: 'left', border: '1px solid #e5e7eb', fontWeight: 600, fontSize: '0.85rem' }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {editingScale.map((s, idx) => (
                <tr key={idx}>
                  <td style={{ padding: '6px 12px', border: '1px solid #e5e7eb' }}>
                    <input
                      type="number"
                      min={1}
                      aria-label="Score value"
                      value={s.value}
                      onChange={(e) => updateScaleEntry(idx, 'value', e.target.value)}
                      style={{ width: '60px', border: '1px solid #d1d5db', borderRadius: '4px', padding: '4px 8px' }}
                    />
                  </td>
                  <td style={{ padding: '6px 12px', border: '1px solid #e5e7eb' }}>
                    <input
                      type="text"
                      aria-label="Level label"
                      value={s.label}
                      onChange={(e) => updateScaleEntry(idx, 'label', e.target.value)}
                      style={{ width: '200px', border: '1px solid #d1d5db', borderRadius: '4px', padding: '4px 8px' }}
                    />
                  </td>
                  <td style={{ padding: '6px 12px', border: '1px solid #e5e7eb' }}>
                    <input
                      type="color"
                      aria-label="Level color"
                      value={s.color}
                      onChange={(e) => updateScaleEntry(idx, 'color', e.target.value)}
                      style={{ width: '48px', height: '32px', border: 'none', cursor: 'pointer' }}
                    />
                  </td>
                  <td style={{ padding: '6px 12px', border: '1px solid #e5e7eb' }}>
                    <button
                      type="button"
                      aria-label={`Remove level ${s.label}`}
                      onClick={() => removeScaleLevel(idx)}
                      style={{ color: '#dc2626', background: 'none', border: 'none', cursor: 'pointer', fontSize: '0.85rem' }}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              type="button"
              onClick={addScaleLevel}
              style={{ padding: '6px 14px', border: '1px solid #d1d5db', borderRadius: '6px', cursor: 'pointer', background: '#fff' }}
            >
              + Add Level
            </button>
            <button
              type="submit"
              disabled={scaleSaving}
              style={{ padding: '6px 16px', background: '#2563eb', color: '#fff', border: 'none', borderRadius: '6px', cursor: 'pointer' }}
            >
              {scaleSaving ? 'Saving…' : 'Save Scale'}
            </button>
          </div>
        </form>
      </section>

      {/* ── CSV Import ─────────────────────────────────────────────────────── */}
      <section aria-labelledby="import-heading" style={{ marginBottom: '32px' }}>
        <h2 id="import-heading" style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '8px' }}>
          Import Standards from CSV
        </h2>
        <p style={{ color: '#6b7280', fontSize: '0.85rem', marginBottom: '12px' }}>
          Upload a CSV file with columns: <code>code</code>, <code>description</code>,{' '}
          <code>domain_code</code>, <code>domain_name</code>, <code>grade_level</code> (optional).
        </p>
        <form onSubmit={(e) => { void handleImportCSV(e) }} style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,text/csv"
            aria-label="CSV file to import"
            onChange={(e) => setCsvFile(e.target.files?.[0] ?? null)}
            style={{ border: '1px solid #d1d5db', borderRadius: '6px', padding: '4px 8px' }}
          />
          <button
            type="submit"
            disabled={!csvFile || importing}
            style={{ padding: '6px 16px', background: '#059669', color: '#fff', border: 'none', borderRadius: '6px', cursor: 'pointer' }}
          >
            {importing ? 'Importing…' : 'Import'}
          </button>
        </form>
        {importResult && (
          <div style={{ marginTop: '12px', padding: '12px', background: '#f9fafb', borderRadius: '6px', border: '1px solid #e5e7eb' }}>
            <p><strong>Domains created/updated:</strong> {importResult.domainsCreated}</p>
            <p><strong>Standards imported:</strong> {importResult.standardsImported}</p>
            {importResult.errors.length > 0 && (
              <details style={{ marginTop: '8px' }}>
                <summary style={{ cursor: 'pointer', color: '#dc2626' }}>
                  {importResult.errors.length} error(s)
                </summary>
                <ul style={{ marginTop: '4px', fontSize: '0.85rem', color: '#dc2626' }}>
                  {importResult.errors.map((e, i) => <li key={i}>{e}</li>)}
                </ul>
              </details>
            )}
          </div>
        )}
      </section>

      {/* ── Standard Domains ───────────────────────────────────────────────── */}
      <section aria-labelledby="domains-heading" style={{ marginBottom: '32px' }}>
        <h2 id="domains-heading" style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '12px' }}>
          Standard Domains
        </h2>
        {loading && <p>Loading…</p>}
        {!loading && domains.length === 0 && (
          <p style={{ color: '#6b7280' }}>No domains yet. Import a CSV or create one below.</p>
        )}
        {domains.length > 0 && (
          <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '16px', fontSize: '0.875rem' }}>
            <thead>
              <tr style={{ background: '#f9fafb' }}>
                {['Code', 'Name', 'Grade Level'].map((h) => (
                  <th key={h} style={{ padding: '8px 12px', textAlign: 'left', border: '1px solid #e5e7eb', fontWeight: 600 }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {domains.map((d) => (
                <tr key={d.id}>
                  <td style={{ padding: '8px 12px', border: '1px solid #e5e7eb', fontFamily: 'monospace' }}>{d.code}</td>
                  <td style={{ padding: '8px 12px', border: '1px solid #e5e7eb' }}>{d.name}</td>
                  <td style={{ padding: '8px 12px', border: '1px solid #e5e7eb', color: '#6b7280' }}>{d.gradeLevel ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {/* Create domain form */}
        <details>
          <summary style={{ cursor: 'pointer', fontWeight: 500, color: '#2563eb' }}>
            + Create a domain manually
          </summary>
          <form onSubmit={(e) => { void handleCreateDomain(e) }} style={{ marginTop: '12px', display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'flex-end' }}>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.85rem' }}>
              Code *
              <input
                type="text"
                value={newCode}
                onChange={(e) => setNewCode(e.target.value)}
                placeholder="e.g. OA"
                required
                style={{ border: '1px solid #d1d5db', borderRadius: '6px', padding: '6px 10px' }}
              />
            </label>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.85rem' }}>
              Name *
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="e.g. Operations and Algebraic Thinking"
                required
                style={{ border: '1px solid #d1d5db', borderRadius: '6px', padding: '6px 10px', minWidth: '280px' }}
              />
            </label>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.85rem' }}>
              Grade Level
              <input
                type="text"
                value={newGrade}
                onChange={(e) => setNewGrade(e.target.value)}
                placeholder="e.g. 3"
                style={{ border: '1px solid #d1d5db', borderRadius: '6px', padding: '6px 10px', width: '80px' }}
              />
            </label>
            <button
              type="submit"
              disabled={domainSaving || !newCode.trim() || !newName.trim()}
              style={{ padding: '6px 16px', background: '#2563eb', color: '#fff', border: 'none', borderRadius: '6px', cursor: 'pointer', alignSelf: 'flex-end', marginBottom: '1px' }}
            >
              {domainSaving ? 'Saving…' : 'Create Domain'}
            </button>
          </form>
        </details>
      </section>

      {/* Mastery scale preview */}
      {scale.length > 0 && (
        <section aria-labelledby="scale-preview-heading">
          <h2 id="scale-preview-heading" style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '8px' }}>
            Current Mastery Scale Preview
          </h2>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            {[...scale].sort((a, b) => b.value - a.value).map((s) => (
              <span
                key={s.value}
                style={{
                  padding: '6px 14px',
                  borderRadius: '9999px',
                  background: s.color ?? '#6b7280',
                  color: '#fff',
                  fontWeight: 600,
                  fontSize: '0.85rem',
                }}
              >
                {s.value} – {s.label}
              </span>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
