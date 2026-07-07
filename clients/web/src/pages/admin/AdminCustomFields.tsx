import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useConfirm } from '../../components/use-confirm'
import { GripVertical, Plus, Trash2 } from 'lucide-react'
import {
  createCustomField,
  deleteCustomField,
  fetchCustomFields,
  reorderCustomFields,
  updateCustomField,
  type CustomFieldDefinition,
  type CustomFieldEntityType,
  type CustomFieldType,
  type CustomFieldVisibility,
} from '../../lib/custom-fields-api'
import { fetchAdminConsoleCapabilities } from '../../lib/admin-console-api'
import { toastMutationError } from '../../lib/lms-toast'

const ENTITY_TABS: { id: CustomFieldEntityType; label: string }[] = [
  { id: 'user', label: 'Users' },
  { id: 'course', label: 'Courses' },
  { id: 'enrollment', label: 'Enrollments' },
]

const FIELD_TYPES: CustomFieldType[] = ['text', 'number', 'boolean', 'date', 'select']
const VISIBILITIES: { value: CustomFieldVisibility; label: string }[] = [
  { value: 'admin_only', label: 'Admin only' },
  { value: 'instructor', label: 'Instructor' },
  { value: 'student', label: 'Student' },
]

type DraftField = {
  key: string
  label: string
  fieldType: CustomFieldType
  selectOptions: string
  isRequired: boolean
  visibility: CustomFieldVisibility
}

const emptyDraft = (): DraftField => ({
  key: '',
  label: '',
  fieldType: 'text',
  selectOptions: '',
  isRequired: false,
  visibility: 'admin_only',
})

export default function AdminCustomFields() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const titleId = useId()
  const drawerTitleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [entityType, setEntityType] = useState<CustomFieldEntityType>('user')
  const [fields, setFields] = useState<CustomFieldDefinition[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [canManage, setCanManage] = useState(false)
  const [customFieldsEnabled, setCustomFieldsEnabled] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<CustomFieldDefinition | null>(null)
  const [draft, setDraft] = useState<DraftField>(emptyDraft())
  const [busy, setBusy] = useState(false)
  const [dragId, setDragId] = useState<string | null>(null)

  useEffect(() => {
    void fetchAdminConsoleCapabilities()
      .then((c) => {
        setCanManage(c.canManage)
        setCustomFieldsEnabled(c.customFieldsEnabled)
      })
      .catch(() => {
        setCanManage(false)
        setCustomFieldsEnabled(false)
      })
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setFields(await fetchCustomFields(entityType, orgId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load custom fields.')
    } finally {
      setLoading(false)
    }
  }, [entityType, orgId])

  useEffect(() => {
    if (customFieldsEnabled) void load()
  }, [customFieldsEnabled, load])

  function openCreate() {
    setEditing(null)
    setDraft(emptyDraft())
    setDrawerOpen(true)
  }

  function openEdit(field: CustomFieldDefinition) {
    setEditing(field)
    setDraft({
      key: field.key,
      label: field.label,
      fieldType: field.fieldType,
      selectOptions: (field.selectOptions ?? []).join(', '),
      isRequired: field.isRequired,
      visibility: field.visibility,
    })
    setDrawerOpen(true)
  }

  async function saveField() {
    setBusy(true)
    setError(null)
    try {
      const selectOptions = draft.selectOptions
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      if (editing) {
        await updateCustomField(editing.id, {
          label: draft.label,
          fieldType: draft.fieldType,
          selectOptions: draft.fieldType === 'select' ? selectOptions : undefined,
          isRequired: draft.isRequired,
          visibility: draft.visibility,
        }, orgId)
      } else {
        await createCustomField({
          entityType,
          key: draft.key,
          label: draft.label,
          fieldType: draft.fieldType,
          selectOptions: draft.fieldType === 'select' ? selectOptions : undefined,
          isRequired: draft.isRequired,
          visibility: draft.visibility,
          sortOrder: fields.length,
        }, orgId)
      }
      setDrawerOpen(false)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save custom field.')
    } finally {
      setBusy(false)
    }
  }

  async function removeField(field: CustomFieldDefinition) {
    if (
      !(await confirm({
        title: t('admin.deleteCustomField.title', { label: field.label }),
        variant: 'danger',
      }))
    ) {
      return
    }
    setBusy(true)
    try {
      await deleteCustomField(field.id, orgId)
      await load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to delete custom field.')
    } finally {
      setBusy(false)
    }
  }

  async function onDrop(targetId: string) {
    if (!dragId || dragId === targetId || !canManage) return
    const ids = fields.map((f) => f.id)
    const from = ids.indexOf(dragId)
    const to = ids.indexOf(targetId)
    if (from < 0 || to < 0) return
    ids.splice(from, 1)
    ids.splice(to, 0, dragId)
    setBusy(true)
    try {
      const reordered = await reorderCustomFields(entityType, ids, orgId)
      setFields(reordered)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to reorder fields.')
    } finally {
      setBusy(false)
      setDragId(null)
    }
  }

  if (!customFieldsEnabled) {
    return <p className="text-sm text-slate-500">Custom fields are not enabled for this platform.</p>
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
          Custom fields
        </h1>
        {canManage && (
          <button
            type="button"
            onClick={openCreate}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            <Plus className="h-4 w-4" aria-hidden />
            Add field
          </button>
        )}
      </div>

      <div role="tablist" aria-label="Entity type" className="mb-4 flex gap-1 border-b border-slate-200 dark:border-neutral-800">
        {ENTITY_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={entityType === tab.id}
            onClick={() => setEntityType(tab.id)}
            className={`px-4 py-2 text-sm font-medium ${
              entityType === tab.id
                ? 'border-b-2 border-indigo-600 text-indigo-700 dark:text-indigo-300'
                : 'text-slate-600 hover:text-slate-900 dark:text-slate-400'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {error && (
        <p className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-slate-500">Loading custom fields…</p>
      ) : fields.length === 0 ? (
        <p className="text-sm text-slate-500">No custom fields defined for {entityType}s yet.</p>
      ) : (
        <ul className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
          {fields.map((field) => (
            <li
              key={field.id}
              className="flex items-center gap-3 px-3 py-3"
              draggable={canManage}
              onDragStart={() => setDragId(field.id)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => void onDrop(field.id)}
            >
              {canManage && <GripVertical className="h-4 w-4 shrink-0 text-slate-400" aria-hidden />}
              <div className="min-w-0 flex-1">
                <p className="font-medium text-slate-900 dark:text-slate-100">{field.label}</p>
                <p className="text-xs text-slate-500">
                  {field.key} · {field.fieldType} · {field.visibility.replace('_', ' ')}
                  {field.isRequired ? ' · required' : ''}
                </p>
              </div>
              {canManage && (
                <div className="flex gap-2">
                  <button type="button" onClick={() => openEdit(field)} className="text-sm text-indigo-600 hover:underline">
                    Edit
                  </button>
                  <button
                    type="button"
                    onClick={() => void removeField(field)}
                    className="inline-flex items-center gap-1 text-sm text-red-600 hover:underline"
                  >
                    <Trash2 className="h-3.5 w-3.5" aria-hidden />
                    Delete
                  </button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      {drawerOpen && (
        <div className="fixed inset-0 z-40 flex justify-end bg-black/30" role="presentation" onClick={() => setDrawerOpen(false)}>
          <div
            role="dialog"
            aria-labelledby={drawerTitleId}
            className="h-full w-full max-w-md overflow-y-auto bg-white p-6 shadow-xl dark:bg-neutral-950"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id={drawerTitleId} className="mb-4 text-lg font-semibold">
              {editing ? 'Edit custom field' : 'Add custom field'}
            </h2>
            <div className="space-y-4">
              {!editing && (
                <label className="block text-sm">
                  <span className="mb-1 block font-medium">Key</span>
                  <input
                    value={draft.key}
                    onChange={(e) => setDraft({ ...draft, key: e.target.value })}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                    placeholder="student_id"
                  />
                </label>
              )}
              <label className="block text-sm">
                <span className="mb-1 block font-medium">Label</span>
                <input
                  value={draft.label}
                  onChange={(e) => setDraft({ ...draft, label: e.target.value })}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                />
              </label>
              <label className="block text-sm">
                <span className="mb-1 block font-medium">Type</span>
                <select
                  value={draft.fieldType}
                  onChange={(e) => setDraft({ ...draft, fieldType: e.target.value as CustomFieldType })}
                  disabled={!!editing}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                >
                  {FIELD_TYPES.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>
              </label>
              {draft.fieldType === 'select' && (
                <label className="block text-sm">
                  <span className="mb-1 block font-medium">Options (comma-separated)</span>
                  <input
                    value={draft.selectOptions}
                    onChange={(e) => setDraft({ ...draft, selectOptions: e.target.value })}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                    placeholder="Math, Science"
                  />
                </label>
              )}
              <label className="block text-sm">
                <span className="mb-1 block font-medium">Visibility</span>
                <select
                  value={draft.visibility}
                  onChange={(e) => setDraft({ ...draft, visibility: e.target.value as CustomFieldVisibility })}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                >
                  {VISIBILITIES.map((v) => (
                    <option key={v.value} value={v.value}>
                      {v.label}
                    </option>
                  ))}
                </select>
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={draft.isRequired}
                  onChange={(e) => setDraft({ ...draft, isRequired: e.target.checked })}
                />
                Required
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button type="button" onClick={() => setDrawerOpen(false)} className="rounded-lg px-3 py-2 text-sm">
                Cancel
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => void saveField()}
                className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
      {ConfirmDialogHost}
    </div>
  )
}
