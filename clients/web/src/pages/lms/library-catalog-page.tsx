import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { BookOpen, Plus, Trash2 } from 'lucide-react'
import {
  createLibraryBook,
  deleteLibraryBook,
  listLibraryBooks,
  type CreateBookPayload,
  type LibraryBook,
} from '../../lib/library-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

type FilterState = {
  lexile_min: string
  lexile_max: string
  grade_band: string
}

type AddBookForm = CreateBookPayload & { title: string }

const GRADE_BANDS = ['', 'K-2', '3-5', '6-8', '9-12', 'K-12']

export default function LibraryCatalogPage() {
  const { orgId } = useParams<{ orgId: string }>()
  const { ffLibrary } = usePlatformFeatures()
  const [books, setBooks] = useState<LibraryBook[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<FilterState>({ lexile_min: '', lexile_max: '', grade_band: '' })
  const [showAdd, setShowAdd] = useState(false)
  const [addForm, setAddForm] = useState<Partial<AddBookForm>>({ title: '' })
  const [addError, setAddError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const f = {
        lexile_min: filter.lexile_min ? Number(filter.lexile_min) : undefined,
        lexile_max: filter.lexile_max ? Number(filter.lexile_max) : undefined,
        grade_band: filter.grade_band || undefined,
      }
      const result = await listLibraryBooks(orgId, f)
      setBooks(result)
    } catch {
      setError('Failed to load library.')
    } finally {
      setLoading(false)
    }
  }, [orgId, filter])

  useEffect(() => {
    if (!ffLibrary) return
    void load()
  }, [orgId, ffLibrary, load])

  const handleFilterSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    void load()
  }

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orgId) return
    if (!addForm.title?.trim()) {
      setAddError('Title is required.')
      return
    }
    setSaving(true)
    setAddError(null)
    try {
      const created = await createLibraryBook(orgId, {
        title: addForm.title.trim(),
        author: addForm.author?.trim() || undefined,
        isbn: addForm.isbn?.trim() || undefined,
        lexileLevel: addForm.lexileLevel,
        fpBand: addForm.fpBand?.trim() || undefined,
        gradeBand: addForm.gradeBand?.trim() || undefined,
        summary: addForm.summary?.trim() || undefined,
      })
      setBooks((prev) => [created, ...prev])
      setShowAdd(false)
      setAddForm({ title: '' })
    } catch (err) {
      setAddError(err instanceof Error ? err.message : 'Failed to add book.')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (book: LibraryBook) => {
    if (!orgId) return
    if (!window.confirm(`Delete "${book.title}"?`)) return
    try {
      await deleteLibraryBook(orgId, book.id)
      setBooks((prev) => prev.filter((b) => b.id !== book.id))
    } catch {
      alert('Failed to delete book.')
    }
  }

  if (!ffLibrary) {
    return (
      <LmsPage title="Library">
        <p className="text-muted-foreground">The library feature is not enabled for this school.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Library Catalog">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="h-5 w-5" aria-hidden />
            <h1 className="text-xl font-semibold">Library Catalog</h1>
          </div>
          <button
            onClick={() => setShowAdd((v) => !v)}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" aria-hidden />
            Add Book
          </button>
        </div>

        {showAdd && (
          <form
            onSubmit={(e) => void handleAdd(e)}
            className="rounded-lg border bg-card p-4 space-y-3"
          >
            <h2 className="font-medium">Add Book</h2>
            {addError && <p className="text-sm text-destructive">{addError}</p>}
            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label htmlFor="add-title" className="block text-sm font-medium mb-1">Title *</label>
                <input
                  id="add-title"
                  type="text"
                  value={addForm.title ?? ''}
                  onChange={(e) => setAddForm((f) => ({ ...f, title: e.target.value }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                  required
                />
              </div>
              <div>
                <label htmlFor="add-author" className="block text-sm font-medium mb-1">Author</label>
                <input
                  id="add-author"
                  type="text"
                  value={addForm.author ?? ''}
                  onChange={(e) => setAddForm((f) => ({ ...f, author: e.target.value }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
              <div>
                <label htmlFor="add-lexile" className="block text-sm font-medium mb-1">Lexile Level</label>
                <input
                  id="add-lexile"
                  type="number"
                  min={0}
                  value={addForm.lexileLevel ?? ''}
                  onChange={(e) =>
                    setAddForm((f) => ({
                      ...f,
                      lexileLevel: e.target.value ? Number(e.target.value) : undefined,
                    }))
                  }
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
              <div>
                <label htmlFor="add-fp" className="block text-sm font-medium mb-1">F&P Band (A–Z)</label>
                <input
                  id="add-fp"
                  type="text"
                  maxLength={2}
                  value={addForm.fpBand ?? ''}
                  onChange={(e) => setAddForm((f) => ({ ...f, fpBand: e.target.value.toUpperCase() }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
              <div>
                <label htmlFor="add-grade" className="block text-sm font-medium mb-1">Grade Band</label>
                <select
                  id="add-grade"
                  value={addForm.gradeBand ?? ''}
                  onChange={(e) => setAddForm((f) => ({ ...f, gradeBand: e.target.value }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                >
                  {GRADE_BANDS.map((b) => (
                    <option key={b} value={b}>
                      {b || '— Any —'}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor="add-isbn" className="block text-sm font-medium mb-1">ISBN</label>
                <input
                  id="add-isbn"
                  type="text"
                  value={addForm.isbn ?? ''}
                  onChange={(e) => setAddForm((f) => ({ ...f, isbn: e.target.value }))}
                  className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
            </div>
            <div>
              <label htmlFor="add-summary" className="block text-sm font-medium mb-1">Summary</label>
              <textarea
                id="add-summary"
                rows={2}
                value={addForm.summary ?? ''}
                onChange={(e) => setAddForm((f) => ({ ...f, summary: e.target.value }))}
                className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={saving}
                className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {saving ? 'Adding…' : 'Add Book'}
              </button>
              <button
                type="button"
                onClick={() => { setShowAdd(false); setAddError(null) }}
                className="rounded-md border px-3 py-1.5 text-sm"
              >
                Cancel
              </button>
            </div>
          </form>
        )}

        <form onSubmit={handleFilterSubmit} className="flex flex-wrap gap-3 items-end">
          <div>
            <label htmlFor="filter-lexile-min" className="block text-xs text-muted-foreground mb-1">
              Lexile min
            </label>
            <input
              id="filter-lexile-min"
              type="number"
              min={0}
              value={filter.lexile_min}
              onChange={(e) => setFilter((f) => ({ ...f, lexile_min: e.target.value }))}
              className="w-24 rounded-md border bg-background px-2 py-1 text-sm"
            />
          </div>
          <div>
            <label htmlFor="filter-lexile-max" className="block text-xs text-muted-foreground mb-1">
              Lexile max
            </label>
            <input
              id="filter-lexile-max"
              type="number"
              min={0}
              value={filter.lexile_max}
              onChange={(e) => setFilter((f) => ({ ...f, lexile_max: e.target.value }))}
              className="w-24 rounded-md border bg-background px-2 py-1 text-sm"
            />
          </div>
          <div>
            <label htmlFor="filter-grade" className="block text-xs text-muted-foreground mb-1">
              Grade band
            </label>
            <select
              id="filter-grade"
              value={filter.grade_band}
              onChange={(e) => setFilter((f) => ({ ...f, grade_band: e.target.value }))}
              className="rounded-md border bg-background px-2 py-1 text-sm"
            >
              {GRADE_BANDS.map((b) => (
                <option key={b} value={b}>
                  {b || 'All grades'}
                </option>
              ))}
            </select>
          </div>
          <button
            type="submit"
            className="rounded-md border px-3 py-1 text-sm hover:bg-muted"
          >
            Filter
          </button>
        </form>

        {error && <p className="text-sm text-destructive">{error}</p>}

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : books.length === 0 ? (
          <p className="text-sm text-muted-foreground">No books found. Add the first one above.</p>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {books.map((book) => (
              <div key={book.id} className="rounded-lg border bg-card p-4 flex flex-col gap-1">
                {book.coverUrl && (
                  <img
                    src={book.coverUrl}
                    alt={`Cover of ${book.title}`}
                    className="lex-content-img h-32 w-auto object-contain mb-2 self-start"
                  />
                )}
                <p className="font-medium leading-snug">{book.title}</p>
                {book.author && <p className="text-sm text-muted-foreground">{book.author}</p>}
                <div className="flex flex-wrap gap-1.5 mt-1">
                  {book.lexileLevel != null && (
                    <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-800 dark:bg-blue-900/30 dark:text-blue-200">
                      Lexile {book.lexileLevel}
                    </span>
                  )}
                  {book.fpBand && (
                    <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-800 dark:bg-green-900/30 dark:text-green-200">
                      F&P {book.fpBand}
                    </span>
                  )}
                  {book.gradeBand && (
                    <span className="rounded-full bg-purple-100 px-2 py-0.5 text-xs text-purple-800 dark:bg-purple-900/30 dark:text-purple-200">
                      Grade {book.gradeBand}
                    </span>
                  )}
                </div>
                {book.summary && (
                  <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{book.summary}</p>
                )}
                <div className="mt-auto pt-2">
                  <button
                    onClick={() => void handleDelete(book)}
                    className="inline-flex items-center gap-1 text-xs text-destructive hover:underline"
                    aria-label={`Delete ${book.title}`}
                  >
                    <Trash2 className="h-3 w-3" aria-hidden />
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </LmsPage>
  )
}
