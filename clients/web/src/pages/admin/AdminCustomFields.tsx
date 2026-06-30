import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { GripVertical, Plus, Trash2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createCustomFieldDefinition,
  deleteCustomFieldDefinition,
  listCustomFieldDefinitions,
  reorderCustomFieldDefinitions,
  type CreateCustomFieldInput,
  type CustomFieldDefinition,
  type CustomFieldEntityType,
} from '../../lib/custom-fields-api'

const ENTITY_TABS: { id: CustomFieldEntityType; label: string }[] = [
  { id: 'user', label: 'Users' },
  { id: 'course', label: 'Courses' },
  { id: 'enrollment', label: 'Enrollments' },
]

const FIELD_TYPES = ['text', 'number', 'boolean', 'date', 'select'] as const
const VISIBILITY_OPTIONS = [
  { value: 'admin_only', label: 'Admin only' },
  { value: 'instructor', label: 'Instructor' },
  { value: 'student', label: 'Student' },
] as const

export default function AdminCustomFieldsPage() {
  const titleId = useId()
  const formId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? ''
  const { customFieldsEnabled, adminConsoleEnabled, loading: featuresLoading } = usePlatformFeatures()

  const [entityType, setEntityType] = useState<CustomFieldEntityType>('user')
  const [fields, setFields] = useState<CustomFieldDefinition[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [form, setForm] = useState<CreateCustomFieldInput>({
    entityType: 'user',
    key: '',
    label: '',
    fieldType: 'text',
    selectOptions: [],
    isRequired: false,
    visibility: 'admin_only',
  })

  const loadFields = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const data = await listCustomFieldDefinitions(orgId, entityType)
      setFields(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load custom fields')
    } finally {
      setLoading(false)
    }
  }, [orgId, entityType])

  useEffect(() => {
    void loadFields()
  }, [loadFields])

  useEffect(() => {
    setForm((f) => ({ ...f, entityType }))
  }, [entityType])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!orgId) return
    setError(null)
    try {
      const options =
        form.fieldType === 'select'
          ? (form.selectOptions ?? []).map((s) => s.trim()).filter(Boolean)
          : undefined
      await createCustomFieldDefinition(orgId, { ...form, selectOptions: options })
      setDrawerOpen(false)
      setForm({
        entityType,
        key: '',
        label: '',
        fieldType: 'text',
        selectOptions: [],
        isRequired: false,
        visibility: 'admin_only',
      })
      await loadFields()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create field')
    }
  }

  async function handleDelete(fieldId: string) {
    if (!orgId || !window.confirm('Delete this custom field? Existing values are retained but hidden.')) return
    try {
      await deleteCustomFieldDefinition(orgId, fieldId)
      await loadFields()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete field')
    }
  }

  async function handleReorder(from: number, to: number) {
    if (from === to || !orgId) return
    const next = [...fields]
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    setFields(next)
    try {
      const updated = await reorderCustomFieldDefinitions(
        orgId,
        entityType,
        next.map((f) => f.id),
      )
      setFields(updated)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reorder fields')
      await loadFields()
    }
  }

  if (featuresLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>
  }
  if (!adminConsoleEnabled || !customFieldsEnabled) {
    return (
      <div className="mx-auto max-w-lg text-center">
        <h1 className="text-lg font-semibold">Custom fields unavailable</h1>
        <p className="mt-2 text-sm text-slate-600">Enable the custom fields platform feature to use this page.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
            Custom fields
          </h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            Define org metadata fields for users, courses, and enrollments.
          </p>
        </div>
        <button
          type="button"
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          onClick={() => setDrawerOpen(true)}
        >
          <Plus className="h-4 w-4" aria-hidden />
          Add field
        </button>
      </header>

      <div role="tablist" aria-label="Entity type" className="mb-4 flex gap-1 border-b border-slate-200 dark:border-neutral-800">
        {ENTITY_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={entityType === tab.id}
            className={`px-4 py-2 text-sm font-medium ${
              entityType === tab.id
                ? 'border-b-2 border-indigo-600 text-indigo-700 dark:text-indigo-300'
                : 'text-slate-600 hover:text-slate-900 dark:text-slate-400'
            }`}
            onClick={() => setEntityType(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {error ? (
        <p role="alert" className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="text-sm text-slate-500">Loading fields…</p>
      ) : fields.length === 0 ? (
        <p className="text-sm text-slate-500">No custom fields defined for {entityType} yet.</p>
      ) : (
        <ul className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
          {fields.map((field, index) => (
            <li
              key={field.id}
              draggable
              onDragStart={() => setDragIndex(index)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => {
                if (dragIndex !== null) void handleReorder(dragIndex, index)
                setDragIndex(null)
              }}
              className="flex items-center gap-3 px-3 py-3"
            >
              <button
                type="button"
                className="cursor-grab text-slate-400 hover:text-slate-600"
                aria-label={`Reorder ${field.label}`}
              >
                <GripVertical className="h-4 w-4" aria-hidden />
              </button>
              <div className="min-w-0 flex-1">
                <p className="font-medium text-slate-900 dark:text-slate-100">{field.label}</p>
                <p className="text-xs text-slate-500">
                  {field.key} · {field.fieldType} · {field.visibility.replace('_', ' ')}
                  {field.isRequired ? ' · required' : ''}
                </p>
              </div>
              <button
                type="button"
                className="rounded p-2 text-slate-500 hover:bg-red-50 hover:text-red-700"
                aria-label={`Delete ${field.label}`}
                onClick={() => void handleDelete(field.id)}
              >
                <Trash2 className="h-4 w-4" aria-hidden />
              </button>
            </li>
          ))}
        </ul>
      )}

      {drawerOpen ? (
        <div className="fixed inset-0 z-40 flex justify-end bg-black/30" role="presentation">
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby={formId}
            className="h-full w-full max-w-md overflow-y-auto bg-white p-6 shadow-xl dark:bg-neutral-950"
          >
            <h2 id={formId} className="text-lg font-semibold">
              Add custom field
            </h2>
            <form className="mt-4 space-y-4" onSubmit={(e) => void handleCreate(e)}>
              <label className="block text-sm">
                <span className="font-medium">Key</span>
                <input
                  required
                  pattern="[a-z][a-z0-9_]*"
                  className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  value={form.key}
                  onChange={(e) => setForm({ ...form, key: e.target.value })}
                  placeholder="student_id"
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">Label</span>
                <input
                  required
                  className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  value={form.label}
                  onChange={(e) => setForm({ ...form, label: e.target.value })}
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">Type</span>
                <select
                  className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  value={form.fieldType}
                  onChange={(e) =>
                    setForm({ ...form, fieldType: e.target.value as CreateCustomFieldInput['fieldType'] })
                  }
                >
                  {FIELD_TYPES.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>
              </label>
              {form.fieldType === 'select' ? (
                <label className="block text-sm">
                  <span className="font-medium">Options (comma-separated)</span>
                  <input
                    className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                    value={(form.selectOptions ?? []).join(', ')}
                    onChange={(e) =>
                      setForm({
                        ...form,
                        selectOptions: e.target.value.split(',').map((s) => s.trim()),
                      })
                    }
                    placeholder="Math, Science"
                  />
                </label>
              ) : null}
              <label className="block text-sm">
                <span className="font-medium">Visibility</span>
                <select
                  className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  value={form.visibility}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      visibility: e.target.value as CreateCustomFieldInput['visibility'],
                    })
                  }
                >
                  {VISIBILITY_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.isRequired ?? false}
                  onChange={(e) => setForm({ ...form, isRequired: e.target.checked })}
                />
                Required
              </label>
              <div className="flex gap-2 pt-2">
                <button
                  type="submit"
                  className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                >
                  Save
                </button>
                <button
                  type="button"
                  className="rounded-lg border border-slate-300 px-4 py-2 text-sm dark:border-neutral-700"
                  onClick={() => setDrawerOpen(false)}
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}
