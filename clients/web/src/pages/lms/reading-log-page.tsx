import { useCallback, useEffect, useState } from 'react'
import { BookOpen, Plus } from 'lucide-react'
import {
  createReadingLogEntry,
  listReadingLogEntries,
  type CreateReadingLogPayload,
  type ReadingLogEntry,
} from '../../lib/library-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

const today = () => new Date().toISOString().slice(0, 10)

export default function ReadingLogPage() {
  const { ffLibrary } = usePlatformFeatures()
  const [entries, setEntries] = useState<ReadingLogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState<CreateReadingLogPayload>({ logDate: today() })
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await listReadingLogEntries(100)
      setEntries(result)
    } catch {
      setError('Failed to load reading log.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!ffLibrary) return
    void load()
  }, [ffLibrary, load])

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.bookTitle?.trim() && !form.bookId?.trim()) {
      setFormError('Please enter a book title or select a book.')
      return
    }
    if (!form.logDate) {
      setFormError('Date is required.')
      return
    }
    setSaving(true)
    setFormError(null)
    try {
      const entry = await createReadingLogEntry({
        ...form,
        bookTitle: form.bookTitle?.trim() || undefined,
        reflection: form.reflection?.trim() || undefined,
      })
      setEntries((prev) => [entry, ...prev])
      setShowAdd(false)
      setForm({ logDate: today() })
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save entry.')
    } finally {
      setSaving(false)
    }
  }

  const totalWeeklyPages = entries
    .filter((e) => {
      const d = new Date(e.logDate)
      const cutoff = new Date()
      cutoff.setDate(cutoff.getDate() - 6)
      return d >= cutoff
    })
    .reduce((sum, e) => sum + (e.pagesRead ?? 0), 0)

  if (!ffLibrary) {
    return (
      <LmsPage title="Reading Log">
        <p className="text-muted-foreground">The library feature is not enabled.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Reading Log">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="h-5 w-5" aria-hidden />
            <h1 className="text-xl font-semibold">My Reading Log</h1>
          </div>
          <button
            onClick={() => setShowAdd((v) => !v)}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" aria-hidden />
            Log Reading
          </button>
        </div>

        <div className="rounded-lg border bg-card p-4 inline-flex flex-col gap-0.5">
          <p className="text-xs text-muted-foreground">Pages read this week</p>
          <p className="text-3xl font-bold tabular-nums">{totalWeeklyPages}</p>
        </div>

        {showAdd && (
          <form
            onSubmit={(e) => void handleAdd(e)}
            className="rounded-lg border bg-card p-4 space-y-3"
          >
            <h2 className="font-medium">Log a Reading Entry</h2>
            {formError && <p className="text-sm text-destructive">{formError}</p>}
            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label htmlFor="log-title" className="block text-sm font-medium mb-1">Book Title</label>
                <input
                  id="log-title"
                  type="text"
                  value={form.bookTitle ?? ''}
                  onChange={(e) => setForm((f) => ({ ...f, bookTitle: e.target.value }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                  placeholder="e.g. Charlotte's Web"
                />
              </div>
              <div>
                <label htmlFor="log-date" className="block text-sm font-medium mb-1">Date *</label>
                <input
                  id="log-date"
                  type="date"
                  value={form.logDate}
                  onChange={(e) => setForm((f) => ({ ...f, logDate: e.target.value }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                  required
                />
              </div>
              <div>
                <label htmlFor="log-pages" className="block text-sm font-medium mb-1">Pages Read</label>
                <input
                  id="log-pages"
                  type="number"
                  min={1}
                  value={form.pagesRead ?? ''}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      pagesRead: e.target.value ? Number(e.target.value) : undefined,
                    }))
                  }
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
            </div>
            <div>
              <label htmlFor="log-reflection" className="block text-sm font-medium mb-1">
                Reflection <span className="text-muted-foreground text-xs">(1–3 sentences)</span>
              </label>
              <textarea
                id="log-reflection"
                rows={2}
                value={form.reflection ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, reflection: e.target.value }))}
                className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                maxLength={500}
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={saving}
                className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {saving ? 'Saving…' : 'Save Entry'}
              </button>
              <button
                type="button"
                onClick={() => { setShowAdd(false); setFormError(null) }}
                className="rounded-md border px-3 py-1.5 text-sm"
              >
                Cancel
              </button>
            </div>
          </form>
        )}

        {error && <p className="text-sm text-destructive">{error}</p>}

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : entries.length === 0 ? (
          <p className="text-sm text-muted-foreground">No reading entries yet. Log your first book above!</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-start text-xs text-muted-foreground">
                <th className="pb-2 font-medium">Date</th>
                <th className="pb-2 font-medium">Book</th>
                <th className="pb-2 font-medium text-end">Pages</th>
                <th className="pb-2 font-medium hidden sm:table-cell">Reflection</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.id} className="border-b last:border-0">
                  <td className="py-2 pe-3 whitespace-nowrap">{entry.logDate}</td>
                  <td className="py-2 pe-3">{entry.bookTitle ?? '—'}</td>
                  <td className="py-2 pe-3 text-end tabular-nums">{entry.pagesRead ?? '—'}</td>
                  <td className="py-2 text-muted-foreground hidden sm:table-cell line-clamp-1">
                    {entry.reflection ?? '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </LmsPage>
  )
}
