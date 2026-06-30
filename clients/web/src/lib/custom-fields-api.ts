import { authorizedFetch } from './api'

export type CustomFieldEntityType = 'user' | 'course' | 'enrollment'

export type CustomFieldDefinition = {
  id: string
  orgId: string
  entityType: CustomFieldEntityType
  key: string
  label: string
  fieldType: 'text' | 'number' | 'boolean' | 'date' | 'select'
  selectOptions?: string[]
  isRequired: boolean
  visibility: 'admin_only' | 'instructor' | 'student'
  sortOrder: number
  createdAt: string
}

export type CreateCustomFieldInput = {
  entityType: CustomFieldEntityType
  key: string
  label: string
  fieldType: CustomFieldDefinition['fieldType']
  selectOptions?: string[]
  isRequired?: boolean
  visibility?: CustomFieldDefinition['visibility']
  sortOrder?: number
}

function orgQuery(orgId: string) {
  return orgId ? `?orgId=${encodeURIComponent(orgId)}` : ''
}

export async function listCustomFieldDefinitions(
  orgId: string,
  entityType: CustomFieldEntityType,
): Promise<CustomFieldDefinition[]> {
  const res = await authorizedFetch(
    `/api/v1/admin-console/custom-fields?entity_type=${entityType}${orgId ? `&orgId=${encodeURIComponent(orgId)}` : ''}`,
  )
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<CustomFieldDefinition[]>
}

export async function createCustomFieldDefinition(
  orgId: string,
  input: CreateCustomFieldInput,
): Promise<CustomFieldDefinition> {
  const res = await authorizedFetch(`/api/v1/admin-console/custom-fields${orgQuery(orgId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<CustomFieldDefinition>
}

export async function updateCustomFieldDefinition(
  orgId: string,
  fieldId: string,
  patch: Partial<CreateCustomFieldInput>,
): Promise<CustomFieldDefinition> {
  const res = await authorizedFetch(
    `/api/v1/admin-console/custom-fields/${fieldId}${orgQuery(orgId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<CustomFieldDefinition>
}

export async function deleteCustomFieldDefinition(orgId: string, fieldId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/admin-console/custom-fields/${fieldId}${orgQuery(orgId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) throw new Error(await res.text())
}

export async function reorderCustomFieldDefinitions(
  orgId: string,
  entityType: CustomFieldEntityType,
  fieldIds: string[],
): Promise<CustomFieldDefinition[]> {
  const res = await authorizedFetch(
    `/api/v1/admin-console/custom-fields/reorder${orgQuery(orgId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ entityType, fieldIds }),
    },
  )
  if (!res.ok) throw new Error(await res.text())
  return res.json() as Promise<CustomFieldDefinition[]>
}

export async function downloadUsersExport(orgId: string): Promise<Blob> {
  const res = await authorizedFetch(`/api/v1/admin-console/users/export${orgQuery(orgId)}`)
  if (!res.ok) throw new Error(await res.text())
  return res.blob()
}
