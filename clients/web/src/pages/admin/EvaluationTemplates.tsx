import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createEvaluationTemplate,
  deleteEvaluationTemplate,
  listEvaluationTemplates,
  updateEvaluationTemplate,
  type EvaluationQuestion,
  type EvaluationTemplate,
} from '../../lib/course-evaluations-api'

const QUESTION_TYPES: { value: EvaluationQuestion['type']; label: string }[] = [
  { value: 'rating', label: 'Rating scale (1–5)' },
  { value: 'multiple_choice', label: 'Multiple choice' },
  { value: 'open_text', label: 'Open text' },
]

function emptyQuestion(): EvaluationQuestion {
  return { type: 'rating', text: '', required: false }
}

function QuestionEditor({
  question,
  index,
  onChange,
  onRemove,
}: {
  question: EvaluationQuestion
  index: number
  onChange: (q: EvaluationQuestion) => void
  onRemove: () => void
}) {
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-neutral-700 dark:bg-neutral-800/50">
      <div className="mb-3 flex items-center justify-between">
        <span className="text-xs font-semibold text-slate-500 dark:text-neutral-400">
          Question {index + 1}
        </span>
        <button
          type="button"
          onClick={onRemove}
          className="text-xs text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
        >
          Remove
        </button>
      </div>

      <div className="mb-3">
        <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-neutral-300">
          Type
        </label>
        <select
          value={question.type}
          onChange={(e) =>
            onChange({ ...question, type: e.target.value as EvaluationQuestion['type'], options: undefined })
          }
          className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
        >
          {QUESTION_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>

      <div className="mb-3">
        <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-neutral-300">
          Question text
        </label>
        <input
          type="text"
          value={question.text}
          onChange={(e) => onChange({ ...question, text: e.target.value })}
          placeholder="Enter your question…"
          className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
        />
      </div>

      {question.type === 'multiple_choice' && (
        <div className="mb-3">
          <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-neutral-300">
            Options (one per line)
          </label>
          <textarea
            value={(question.options ?? []).join('\n')}
            onChange={(e) =>
              onChange({
                ...question,
                options: e.target.value.split('\n').filter(Boolean),
              })
            }
            rows={3}
            placeholder="Option A&#10;Option B&#10;Option C"
            className="w-full resize-y rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
          />
        </div>
      )}

      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-neutral-400">
        <input
          type="checkbox"
          checked={question.required ?? false}
          onChange={(e) => onChange({ ...question, required: e.target.checked })}
          className="h-3.5 w-3.5 accent-indigo-500"
        />
        Required
      </label>
    </div>
  )
}

function TemplateForm({
  initial,
  onSave,
  onCancel,
}: {
  initial?: EvaluationTemplate | null
  onSave: (name: string, questions: EvaluationQuestion[]) => Promise<void>
  onCancel: () => void
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [questions, setQuestions] = useState<EvaluationQuestion[]>(
    initial?.questions ?? [emptyQuestion()],
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      setError('Template name is required.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave(name.trim(), questions)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSave} className="space-y-4">
      <div>
        <label className="mb-1 block text-sm font-medium text-slate-900 dark:text-neutral-100">
          Template name
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Fall 2025 Standard Evaluation"
          className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
        />
      </div>

      <div className="space-y-3">
        <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">Questions</p>
        {questions.map((q, i) => (
          <QuestionEditor
            key={i}
            question={q}
            index={i}
            onChange={(updated) => {
              const next = [...questions]
              next[i] = updated
              setQuestions(next)
            }}
            onRemove={() => setQuestions(questions.filter((_, idx) => idx !== i))}
          />
        ))}
        <button
          type="button"
          onClick={() => setQuestions([...questions, emptyQuestion()])}
          className="text-sm text-indigo-600 hover:underline dark:text-indigo-400"
        >
          + Add question
        </button>
      </div>

      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      <div className="flex gap-3">
        <button
          type="submit"
          disabled={saving}
          className="rounded-xl bg-indigo-600 px-5 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-60 dark:bg-indigo-500"
        >
          {saving ? 'Saving…' : 'Save template'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-xl border border-slate-200 px-5 py-2 text-sm text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

export default function EvaluationTemplates() {
  const { ffCourseEvaluations, loading: featuresLoading } = usePlatformFeatures()
  const [templates, setTemplates] = useState<EvaluationTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<EvaluationTemplate | null | 'new'>(null)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      setTemplates(await listEvaluationTemplates())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load templates.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!featuresLoading && ffCourseEvaluations) {
      void load()
    }
  }, [featuresLoading, ffCourseEvaluations, load])

  if (featuresLoading) return <p>Loading…</p>
  if (!ffCourseEvaluations) {
    return (
      <div role="alert">
        <p>Course evaluations are not enabled for this institution.</p>
      </div>
    )
  }

  const handleSave = async (name: string, questions: EvaluationQuestion[]) => {
    if (editing === 'new') {
      await createEvaluationTemplate(name, questions)
    } else if (editing) {
      await updateEvaluationTemplate(editing.id, name, questions)
    }
    setEditing(null)
    await load()
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this template? This cannot be undone.')) return
    try {
      await deleteEvaluationTemplate(id)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete template.')
    }
  }

  if (editing !== null) {
    return (
      <main className="mx-auto max-w-2xl px-4 py-8">
        <h1 className="mb-6 text-xl font-bold text-slate-900 dark:text-neutral-100">
          {editing === 'new' ? 'New evaluation template' : 'Edit template'}
        </h1>
        <TemplateForm
          initial={editing === 'new' ? null : editing}
          onSave={handleSave}
          onCancel={() => setEditing(null)}
        />
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">
          Evaluation Templates
        </h1>
        <button
          type="button"
          onClick={() => setEditing('new')}
          className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 dark:bg-indigo-500"
        >
          New template
        </button>
      </div>

      {error && (
        <p className="mb-4 text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      {loading ? (
        <div className="flex justify-center py-16">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      ) : templates.length === 0 ? (
        <p className="py-12 text-center text-slate-500 dark:text-neutral-400">
          No templates yet. Create one to get started.
        </p>
      ) : (
        <ul className="space-y-3">
          {templates.map((t) => (
            <li
              key={t.id}
              className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900"
            >
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">{t.name}</p>
                <p className="text-xs text-slate-500 dark:text-neutral-400">
                  {t.questions.length} {t.questions.length === 1 ? 'question' : 'questions'}
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setEditing(t)}
                  className="text-xs text-indigo-600 hover:underline dark:text-indigo-400"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(t.id)}
                  className="text-xs text-red-500 hover:underline dark:text-red-400"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}
