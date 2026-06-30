import { authorizedFetch } from './api'
import type { AdminUser } from './admin-console-api'

export type CustomFieldEntityType = 'user' | 'course' | 'enrollment'
export type CustomFieldType = 'text' | 'number' | 'boolean' | 'date' | 'select'
export type CustomFieldVisibility = 'admin_only' | 'instructor' | 'student'

export type CustomFieldDefinition = {
  id: string
  orgId: string
  entityType: CustomFieldEntityType
  key: string
  label: string
  fieldType: CustomFieldType
  selectOptions?: string[]
  isRequired: boolean
  visibility: CustomFieldVisibility
  sortOrder: number
  createdAt: string
}

function orgQuery(orgId?: string | null): string {
  if (!orgId) return ''
  return `?orgId=${encodeURIComponent(orgId)}`
}

function orgAmp(orgId?: string | null): string {
  if (!orgId) return ''
  return `&orgId=${encodeURIComponent(orgId)}`
}

async function parseJson<T>(res: Response): Promise<T> {
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg =
      typeof raw === 'object' && raw !== null && 'message' in raw
        ? String((raw as { message: string }).message)
        : res.statusText
    throw new Error(msg || 'Request failed')
  }
  return raw as T
}

export async function fetchCustomFields(
  entityType: CustomFieldEntityType,
  orgId?: string | null,
): Promise<CustomFieldDefinition[]> {
  const sp = new URLSearchParams({ entity_type: entityType })
  const res = await authorizedFetch(`/api/v1/admin-console/custom-fields?${sp}${orgAmp(orgId)}`)
  return parseJson(res)
}

export async function createCustomField(
  body: {
    entityType: CustomFieldEntityType
    key: string
    label: string
    fieldType: CustomFieldType
    selectOptions?: string[]
    isRequired: boolean
    visibility: CustomFieldVisibility
    sortOrder?: number
  },
  orgId?: string | null,
): Promise<CustomFieldDefinition> {
  const res = await authorizedFetch(`/api/v1/admin-console/custom-fields${orgQuery(orgId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function updateCustomField(
  fieldId: string,
  body: {
    label?: string
    fieldType?: CustomFieldType
    selectOptions?: string[]
    isRequired?: boolean
    visibility?: CustomFieldVisibility
    sortOrder?: number
  },
  orgId?: string | null,
): Promise<CustomFieldDefinition> {
  const res = await authorizedFetch(`/api/v1/admin-console/custom-fields/${fieldId}${orgQuery(orgId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function deleteCustomField(fieldId: string, orgId?: string | null): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin-console/custom-fields/${fieldId}${orgQuery(orgId)}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    await parseJson(res)
  }
}

export async function reorderCustomFields(
  entityType: CustomFieldEntityType,
  fieldIds: string[],
  orgId?: string | null,
): Promise<CustomFieldDefinition[]> {
  const res = await authorizedFetch(`/api/v1/admin-console/custom-fields/reorder${orgQuery(orgId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ entityType, fieldIds }),
  })
  return parseJson(res)
}

export async function fetchAdminUser(
  userId: string,
  orgId?: string | null,
  includeCustomFields = false,
): Promise<AdminUser & { customFields?: Record<string, unknown> }> {
  const include = includeCustomFields ? '&include=custom_fields' : ''
  const q = orgId ? `?orgId=${encodeURIComponent(orgId)}${include}` : includeCustomFields ? '?include=custom_fields' : ''
  const res = await authorizedFetch(`/api/v1/admin-console/users/${userId}${q}`)
  return parseJson(res)
}

export async function patchAdminUserCustomFields(
  userId: string,
  customFields: Record<string, unknown>,
  orgId?: string | null,
): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin-console/users/${userId}${orgQuery(orgId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ customFields }),
  })
  if (!res.ok) await parseJson(res)
}

export function usersExportUrl(orgId?: string | null): string {
  const base = import.meta.env.VITE_API_URL ?? ''
  const q = orgId ? `?orgId=${encodeURIComponent(orgId)}` : ''
  return `${base}/api/v1/admin-console/users/export.csv${q}`
}
